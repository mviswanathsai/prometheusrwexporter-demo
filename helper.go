package prometheusremotewritev2

import (
	"fmt"
	"time"

	"github.com/prometheus/prometheus/prompb"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	typesv2 "prometheusrwexporter-demo/types"
)

// Since we are assuming that for one metric all its datapoints are from the same TimeSeries
// So just getting the attributes from the first datapoint is enough.
func getLabelsFromExpDataPoints(dp pmetric.ExponentialHistogramDataPointSlice) (labels []prompb.Label) {
	return getLabelsFromAttrs(dp.At(0).Attributes())
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

func exponentialToNativeHistogram(p pmetric.ExponentialHistogramDataPoint) typesv2.Histogram {
	// return a dummy histogram
	return typesv2.Histogram{
		Sum:    p.Sum(),
		Schema: 7,
	}
}

func nativeToExponentialHistogram(p typesv2.Histogram) (h pmetric.ExponentialHistogramDataPoint) {
    // return a dummy ExponentialHistogramDataPoint
	ts := pcommon.NewTimestampFromTime(time.Now())
	h.SetTimestamp(ts)

	// Set 50 samples, 10 each with values 0.5, 1, 2, 4, and 8
	h.SetCount(50)
	h.SetSum(155)
	h.Positive().BucketCounts().FromRaw([]uint64{10, 10, 10, 10, 10, 0})
	return h
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
		m.SetEmptyExponentialHistogram().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
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
	// 1 resource, 1 scope and 5 metrics = 5 TS total

	return request
}

func packSymbol(length, offset uint32) uint32 {
	// Ensure that length and offset fit within the required bits
	if length > 0xFFF || offset > 0xFFFFF {
		panic("length or offset exceeds bit limits")
	}
	return (length << 20) | offset
}

func unpackSymbol(packed uint32) (length, offset uint32) {
	length = packed >> 20     // Extract the upper 12 bits for length
	offset = packed & 0xFFFFF // Extract the lower 20 bits for offset
	return
}

func buildLabelsFromLabelRef(symbols string, labelRefs []uint32) map[string]string {
	// Unpack couples of each int in labelRefs
	// from that, build a map of labelName: labelValue
	labels := make(map[string]string, len(labelRefs)/2)

	for i := 0; i < len(labelRefs)-1; i += 2 {
		// Unpack the length and offset for the label name
		nameLength, nameOffset := unpackSymbol(labelRefs[i])
		// Unpack the length and offset for the label value
		valueLength, valueOffset := unpackSymbol(labelRefs[i+1])

		// Extract the label name and value from the symbols string
		labelName := symbols[nameOffset : nameOffset+nameLength]
		labelValue := symbols[valueOffset : valueOffset+valueLength]

		// Append the label to the list
		labels[labelName] = labelValue
	}

	return labels
}
