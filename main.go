package prometheusremotewritev2

import (
	"fmt"
	typesv2 "prometheusrwexporter-demo/types"
	"sort"
	"time"

	"github.com/prometheus/prometheus/prompb"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
)

func main() {
	PrepareDummyExportRequest()
}

type V2WriteRequestBuilder struct {
	resources pmetric.ResourceMetricsSlice
	symbols   []string
}

func NewV2RequestBuilder(exportReq pmetricotlp.ExportRequest) V2WriteRequestBuilder {
	resourceMetrics := exportReq.Metrics().ResourceMetrics()

	return V2WriteRequestBuilder{
		resources: resourceMetrics,
	}
}

// Seems like we might have to do this loop twice: 1. For creating the symbols table, 2. For converting Metrics to TS
func (builder V2WriteRequestBuilder) makeSymbols() {
	// For this, you have to loop through each  metric of each Scope of each Resource: That means 1
	// nested loop.
	// This is ofcourse, too much complexity. What can we do?
	resourceMetricsSlice := builder.resources
	for i := 0; i < resourceMetricsSlice.Len(); i++ {

		for j := 0; j < resourceMetricsSlice.At(i).ScopeMetrics().Len(); j++ {
			scopeMetricsSlice := resourceMetricsSlice.At(i).ScopeMetrics()

			for k := 0; k < scopeMetricsSlice.At(j).Metrics().Len(); k++ {
                scopeMetricsSlice.At(j).Metrics().At(k).Attributes()


			}
		}
	}
}

func createRequest(symbols []string, timeSeries []typesv2.TimeSeries) typesv2.Request {
	return typesv2.Request{
		Symbols:    symbols,
		Timeseries: timeSeries,
	}
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

func generateSymbolsFromLabels(labels []prompb.Label) (symbols []string) {
	for _, label := range labels {
		symbols = append(symbols, label.Name, label.Value)
	}

	sort.Strings(symbols)

	return symbols
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
