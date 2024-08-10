package prometheusremotewritev2

import (
	"fmt"
	typesv2 "prometheusrwexporter-demo/types"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/prometheus/prompb"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
)

func main() {
	PrepareDummyExportRequest()
}

type symbolsTable struct {
	symbolRef map[string]uint32
	symbols   []string
}

func NewSymbolsTable() symbolsTable {
	return symbolsTable{
		symbolRef: map[string]uint32{"": 0},
		symbols:   []string{""},
	}
}

func (t *symbolsTable) Symbolize(str string) uint32 {
	if ref, ok := t.symbolRef[str]; ok {
		return ref
	}
	ref := uint32(len(t.symbols))
	t.symbols = append(t.symbols, str)
	t.symbolRef[str] = ref
	return ref
}

type V2WriteRequestBuilder struct {
	resources pmetric.ResourceMetricsSlice
	symbols   symbolsTable
	metricMap map[scopeID][]pmetric.MetricSlice
}

type scopeID string

type resourceID int

func (scopeID scopeID) getResourceId() resourceID {
	// Split the string by "-scope-"
	parts := strings.Split(string(scopeID), "-scope-")
	// Convert the first part to an integer
	num, _ := strconv.Atoi(parts[0])

	return resourceID(num)
}

func NewV2RequestBuilder(exportReq pmetricotlp.ExportRequest) (builder V2WriteRequestBuilder) {
	resourceMetricsSlice := exportReq.Metrics().ResourceMetrics()
	builder.resources = resourceMetricsSlice

	for i := 0; i < resourceMetricsSlice.Len(); i++ {
		resourceMetric := resourceMetricsSlice.At(i)

		for j := 0; j < resourceMetricsSlice.At(i).ScopeMetrics().Len(); j++ {
			scopeMetric := resourceMetric.ScopeMetrics().At(j)
			scopeName := scopeMetric.Scope().Name()

			// store each metric slice by scope
			builder.addToMetricMap(i, scopeName, scopeMetric.Metrics())
		}

	}

	builder.symbols = NewSymbolsTable()
	return builder
}

// This could be de-duplicated.
func (builder V2WriteRequestBuilder) addToMetricMap(resourcePosition int, scopeName string, metricSlice pmetric.MetricSlice) {
	scopeMetricsName := fmt.Sprintf("%d-scope-%s", resourcePosition, scopeName)
	builder.metricMap[scopeID(scopeMetricsName)] = append(builder.metricMap[scopeID(scopeMetricsName)], metricSlice)
}

// empty string is required as first element in symbols table
// Seems like we might have to do this loop twice: 1. For creating the symbols table, 2. For converting Metrics to TS
// neglecting scope attributes for now.
func (builder V2WriteRequestBuilder) makeSymbols() {
	resourceMetricsSlice := builder.resources
	for i := 0; i < resourceMetricsSlice.Len(); i++ {

		for j := 0; j < resourceMetricsSlice.At(i).ScopeMetrics().Len(); j++ {
			scopeMetricsSlice := resourceMetricsSlice.At(i).ScopeMetrics()

			for k := 0; k < scopeMetricsSlice.At(j).Metrics().Len(); k++ {
				metric := scopeMetricsSlice.At(j).Metrics().At(k)
				switch metric.Type() {
				case pmetric.MetricTypeExponentialHistogram:
					dataPoints := metric.ExponentialHistogram().DataPoints()
					builder.generateSymbolsFromLabels(getLabelsFromExpDataPoints(dataPoints))
				}
			}
		}
	}
}

func (builder V2WriteRequestBuilder) generateSymbolsFromLabels(labels []prompb.Label) (symbols []string) {
	for _, label := range labels {
		builder.symbols.Symbolize(label.Name)
		builder.symbols.Symbolize(label.Value)
	}

	return symbols
}

func (builder V2WriteRequestBuilder) createV2WriteRequest() typesv2.Request {
	var request typesv2.Request
	resourceMetricsSlice := builder.resources
	// Loop through each metric, generate the respective TS (1:1 relationship assumption)
	// Lets get to work.
	// If we store all these metrics to the builder, we won't need to loop through each time.
	for i := 0; i < resourceMetricsSlice.Len(); i++ {

		for j := 0; j < resourceMetricsSlice.At(i).ScopeMetrics().Len(); j++ {
			scopeMetricsSlice := resourceMetricsSlice.At(i).ScopeMetrics()

			for k := 0; k < scopeMetricsSlice.At(j).Metrics().Len(); k++ {
				metric := scopeMetricsSlice.At(j).Metrics().At(k)
				switch metric.Type() {
				case pmetric.MetricTypeExponentialHistogram:

				}
			}
		}
	}

	return request

}

func getLabelsFromExpDataPoints(dp pmetric.ExponentialHistogramDataPointSlice) (labels []prompb.Label) {
	for i := 0; i < dp.Len(); i++ {
		labels = append(labels, getLabelsFromAttrs(dp.At(i).Attributes())...)
	}
	return labels
}

// for simplicity, we will assume each histogram datapoint from a given metric is from the same ts
func createV2TimeSeries(symbols []string, metric pmetric.Metric, labelSet []prompb.Label) (ts typesv2.TimeSeries) {
	ts = typesv2.TimeSeries{
		Metadata: typesv2.Metadata{},
	}
	addNativeHistograms(metric, ts)
	addLabelRefs(symbols, labelSet, ts)

	return ts
}

func addNativeHistograms(metric pmetric.Metric, ts typesv2.TimeSeries) {
	var nativeHistograms []typesv2.Histogram
	histogramDPs := metric.ExponentialHistogram().DataPoints()
	for j := 0; j < histogramDPs.Len(); j++ {
		nativeHistograms = append(nativeHistograms, exponentialToNativeHistogram(histogramDPs.At(j)))
	}
	ts.Histograms = nativeHistograms
}

func exponentialToNativeHistogram(p pmetric.ExponentialHistogramDataPoint) typesv2.Histogram {
	// return a dummy histogram
	return typesv2.Histogram{
		Sum:    p.Sum(),
		Schema: 7,
	}
}

// TODO: this needs to be updated
// create a labelref table from the symbols and labelset and return it with the timeseries.
func addLabelRefs(symbols []string, labelSet []prompb.Label, ts typesv2.TimeSeries) {
	// ensure the symbols slice is sorted.
	var labelRefs []uint32

	for _, label := range labelSet {
		i := sort.SearchStrings(symbols, label.Name)
		j := sort.SearchStrings(symbols, label.Value)

		labelRefs = append(labelRefs, uint32(i), uint32(j))
	}

	ts.LabelsRefs = labelRefs
}

func getLabelsFromAttrs(attributes pcommon.Map) []prompb.Label {
	labels := make([]prompb.Label, attributes.Len())
	attributes.Range(func(k string, v pcommon.Value) bool {
		for i := 0; i < attributes.Len(); i++ {
			labels[i].Name = k
			labels[i].Value = v.AsString()
			return true
		}
		return false
	})

	return labels
}

func generateAttributes(m pcommon.Map, prefix string, count int) {
	for i := 1; i <= count; i++ {
		m.PutStr(fmt.Sprintf("%v-name-%v", prefix, i), fmt.Sprintf("value-%v", i))
	}
}

// For each Metric inside ScopeMetrics, we are appending histograms with the same attributes each time.
func genHistogramsPerMetricIn(metrics pmetric.MetricSlice, prefix string, metricCount int, attrCount int) {
	ts := pcommon.NewTimestampFromTime(time.Now())
	for i := 1; i <= metricCount; i++ {
		m := metrics.AppendEmpty()
		m.SetEmptyHistogram()
		m.SetName(fmt.Sprintf("histogram-%v", i))
		m.ExponentialHistogram().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		h := m.ExponentialHistogram().DataPoints().AppendEmpty()
		h.SetTimestamp(ts)

		// Set 50 samples, 10 each with values 0.5, 1, 2, 4, and 8
		h.SetCount(50)
		h.SetSum(155)
		h.Positive().BucketCounts().FromRaw([]uint64{10, 10, 10, 10, 10, 0})

		generateAttributes(h.Attributes(), prefix, attrCount)
	}
}

func PrepareDummyExportRequest() pmetricotlp.ExportRequest {
	request := pmetricotlp.NewExportRequest()
	// This adds one Resource to MetricsSlice.
	resourceMetric := request.Metrics().ResourceMetrics().AppendEmpty()

	resourceAttrs := resourceMetric.Resource().Attributes()
	// This generates 10 ResourceAttributes for the Resource.
	generateAttributes(resourceAttrs, "demo-resource", 10)

	// This will add one ScopeMetrics to the list of ScopeMetricsSlice.
	// Essentially, adding one Scope to the given resource.
	metrics := resourceMetric.ScopeMetrics().AppendEmpty().Metrics()
	// This generates 5 `Metric` objects each with Histogram datapoints of count 50 having the same set of 10 attrs.
	genHistogramsPerMetricIn(metrics, "demo-histogram", 5, 10)

	return request
}
