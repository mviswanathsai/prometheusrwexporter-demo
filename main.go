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
	request := pmetricotlp.NewExportRequest()
	resourceMetric := request.Metrics().ResourceMetrics().AppendEmpty()

	// The attributes of the Resource will also be tagged along with the samples/histogramDPs
	resourceAttrs := resourceMetric.Resource().Attributes()
	generateAttributes(resourceAttrs, "demo-resource", 10)
	resourceLabelSet := getLabelsFromAttrs(resourceAttrs)

	metrics := resourceMetric.ScopeMetrics().AppendEmpty().Metrics()
	generateExponentialHistograms(metrics, "demo-histogram", 10, 10)

	var labelSet []prompb.Label
	// Should the resource's attributes also be part of the labelset of the metrics it produces?
	// lets avoid complexity and generate one timeseries per metric
	for i := 0; i < metrics.Len(); i++ {
		histogramDPs := metrics.At(i).ExponentialHistogram().DataPoints()
		for j := 0; j < histogramDPs.Len(); j++ {
			// I reckon there is a way to check for duplicates by hashing, this is done by the PrometheusConverter in
			// the Prometheus Repo. But just to outline the high level flow, I do not think it is necessary.
			labelSet = append(resourceLabelSet, getLabelsFromAttrs(histogramDPs.At(j).Attributes())...)
		}
	}

	symbols := generateSymbolsTableFromLabels(labelSet)
	timeSeries := []typesv2.TimeSeries{}

	// I reckon these can be done (possibly) with goroutines.
	for i := 0; i < metrics.Len(); i++ {
		labelSet = []prompb.Label{}
		histogramDPs := metrics.At(i).ExponentialHistogram().DataPoints()
		for j := 0; j < histogramDPs.Len(); j++ {
			labelSet = append(resourceLabelSet, getLabelsFromAttrs(histogramDPs.At(j).Attributes())...)
		}
		timeSeries = append(timeSeries, createV2TimeSeries(symbols, metrics.At(i), labelSet))
	}

	createRequest(symbols, timeSeries)

	// then we have to encode this request and send it using protobuf.
	// but how?
}


type V2WriteRequestBuilder struct {
	metrics    pmetric.ResourceMetricsSlice
	attributes pcommon.Map
	labelSet   []prompb.Label
	symbols    []string
}

func NewRequestBuilder(exportReq pmetricotlp.ExportRequest) V2WriteRequestBuilder{
	return V2WriteRequestBuilder{}
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

func generateSymbolsTableFromLabels(labels []prompb.Label) (symbols []string) {
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

func generateExponentialHistograms(metrics pmetric.MetricSlice, prefix string, histogramCount int, attrCount int) {
	ts := pcommon.NewTimestampFromTime(time.Now())
	for i := 1; i <= histogramCount; i++ {
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
