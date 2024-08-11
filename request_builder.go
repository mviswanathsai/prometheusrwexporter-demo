package prometheusremotewritev2

import (
	"fmt"
	typesv2 "prometheusrwexporter-demo/types"
	"time"

	"github.com/prometheus/prometheus/prompb"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
)

type V2WriteRequestBuilder struct {
	resources         pmetric.ResourceMetricsSlice
	symbols           symbolsTable
	scopeMetricSlices map[resourceID][]pmetric.Metric
	request           typesv2.Request
	tsSlice           []*ts
	encoder           encoder
}

func NewV2RequestBuilder(exportReq pmetricotlp.ExportRequest) (*V2WriteRequestBuilder, error) {
	scopeMetricSlices := make(map[resourceID][]pmetric.Metric)

	resourceMetricsSlice := exportReq.Metrics().ResourceMetrics()
	if resourceMetricsSlice.Len() == 0 {
		return nil, fmt.Errorf("Invalid Request")
	}
	// This just makes one giant scopeMetricSlice per resource, instead of there being multiple
	// scopeMetricSlices hidden inside each scope for the resource.
	//
	// In the grand scheme of things this just means we are omitting any scope related
	// attributes in the final PRW export.
	// For the sake of this POC, I think that's fine.
	for i := 0; i < resourceMetricsSlice.Len(); i++ {
		resourceMetric := resourceMetricsSlice.At(i)
		for j := 0; j < resourceMetricsSlice.At(i).ScopeMetrics().Len(); j++ {
			scopeMetric := resourceMetric.ScopeMetrics().At(j)

			for k := 0; k < scopeMetric.Metrics().Len(); k++ {
				scopeMetricSlices[resourceID(i)] = append(scopeMetricSlices[resourceID(i)], scopeMetric.Metrics().At(k))
			}
		}
	}

	return &V2WriteRequestBuilder{
		scopeMetricSlices: scopeMetricSlices,
		resources:         resourceMetricsSlice,
		symbols:           NewSymbolsTable(),
		request:           typesv2.Request{},
	}, nil
}

func (builder *V2WriteRequestBuilder) CreateV2WriteRequest() {
	var timeSeries []typesv2.TimeSeries
	builder.makeTimeSeriesSlice()

	for _, ts := range builder.tsSlice {
		// Lets just consider histograms because thats what we generated.
		switch ts.metric.Type() {
		default:
			ts.addNativeHistograms()
		}

		v2ts := typesv2.TimeSeries{
			LabelsRefs:       ts.labelRef,
			Metadata:         typesv2.Metadata{},
			CreatedTimestamp: int64(time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))),
			Histograms:       ts.histograms,
		}

		timeSeries = append(timeSeries, v2ts)
	}

	builder.request = typesv2.Request{
		Symbols:    builder.symbols.symbols,
		Timeseries: timeSeries,
	}
}

// neglecting scope attributes for now.
func (builder *V2WriteRequestBuilder) makeTimeSeriesSlice() {
	for resourceID, metricSlice := range builder.scopeMetricSlices {
		// get the resource attributes as well and append it to the Timeseries
		resourceAttrs := builder.resources.At(int(resourceID)).Resource().Attributes()
		labels := getLabelsFromAttrs(resourceAttrs)

		for i := 0; i < len(metricSlice); i++ {
			metric := metricSlice[i]
			switch metric.Type() {
			case pmetric.MetricTypeExponentialHistogram:
				dataPoints := metric.ExponentialHistogram().DataPoints()
				labels = append(labels, getLabelsFromExpDataPoints(dataPoints)...)
				builder.symbolizeLabels(labels)

				// While we are at it, we can also create neat ts objects which will come
				// in handy later.
				ts := newTS(metric, labels)
				ts.generateLabelRefs()
				builder.appendTS(ts)

			default:
				dataPoints := metric.ExponentialHistogram().DataPoints()
				builder.symbolizeLabels(getLabelsFromExpDataPoints(dataPoints))
			}
		}
	}
}

func (builder *V2WriteRequestBuilder) appendTS(ts *ts) {
	builder.tsSlice = append(builder.tsSlice, ts)
}

func (builder *V2WriteRequestBuilder) symbolizeLabels(labels []prompb.Label) {
	for _, label := range labels {
		builder.symbols.Symbolize(label.Name)
		builder.symbols.Symbolize(label.Value)
	}
}

type ts struct {
	metric     pmetric.Metric
	labelSet   []prompb.Label
	labelRef   stack
	histograms []typesv2.Histogram
}

// Maybe we should just initialize an empty TS
func newTS(metric pmetric.Metric, labelSet []prompb.Label) *ts {
	return &ts{
		metric:   metric,
		labelSet: labelSet,
	}
}

func (ts *ts) generateLabelRefs() {
	labelRefStack := newStack()
	offset := uint32(0)

	for _, label := range ts.labelSet {
		nameSymbol := packSymbol(uint32(len(label.Name)), offset)
		labelRefStack.push(nameSymbol)
		offset += uint32(32)

		valueSymbol := packSymbol(uint32(len(label.Value)), offset)
		labelRefStack.push(valueSymbol)
		offset += uint32(32)
	}

	ts.labelRef = labelRefStack
}

// might need to change this again later.
func (ts *ts) addNativeHistograms() {
	var nativeHistograms []typesv2.Histogram
	histogramDPs := ts.metric.ExponentialHistogram().DataPoints()
	for j := 0; j < histogramDPs.Len(); j++ {
		nativeHistograms = append(nativeHistograms, exponentialToNativeHistogram(histogramDPs.At(j)))
	}
	ts.histograms = nativeHistograms
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

// Since the discussion related to encoding is still ongoing, we would benefit from decoupling the
// encoder logic from the request builder.
type encoder interface {
	Encode()
	Decode()
}

type resourceID int

type stack []uint32

func newStack() stack {
	return stack([]uint32{})
}

func (s stack) push(v uint32) stack {
	return append(s, v)
}

func (s stack) pop() (stack, uint32) {
	l := len(s)

	if l == 0 {
		return s, 0
	}

	return s[:l-1], s[l-1]
}

func (s stack) last() (bool, uint32) {
	if len(s) == 0 {
		return false, 0
	}
	return true, s[len(s)-1]
}
