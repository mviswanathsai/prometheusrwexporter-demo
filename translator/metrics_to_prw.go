package prometheusremotewritev2

import (
	"errors"
	"fmt"
	typesv2 "prometheusrwexporter-demo/types"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/multierr"
)

type Settings struct {
	Namespace                 string
	ExternalLabels            map[string]string
	DisableTargetInfo         bool
	ExportCreatedMetric       bool
	AddMetricSuffixes         bool
	SendMetadata              bool
	PromoteResourceAttributes []string
}

// PrometheusConverter converts from OTel write format to Prometheus remote write format.
type PrometheusConverter struct {
	unique    map[uint64]typesv2.TimeSeries
	conflicts map[uint64][]typesv2.TimeSeries
}

func NewPrometheusConverter() *PrometheusConverter {
	return &PrometheusConverter{
		unique:    map[uint64]typesv2.TimeSeries{},
		conflicts: map[uint64][]typesv2.TimeSeries{},
	}
}

// FromMetrics converts pmetric.Metrics to Prometheus remote write format.
func (c *PrometheusConverter) FromMetrics(md pmetric.Metrics, settings Settings) (errs error) {
	resourceMetricsSlice := md.ResourceMetrics()
	for i := 0; i < resourceMetricsSlice.Len(); i++ {
		resourceMetrics := resourceMetricsSlice.At(i)
		// the resource emitting the metrics
		resource := resourceMetrics.Resource()
		// the list of metrics emitted by the resource
		scopeMetricsSlice := resourceMetrics.ScopeMetrics()
		// keep track of the most recent timestamp in the ResourceMetrics for
		// use with the "target" info metric
		var mostRecentTimestamp pcommon.Timestamp
		for j := 0; j < scopeMetricsSlice.Len(); j++ {
			metricSlice := scopeMetricsSlice.At(j).Metrics()

			// TODO: decide if instrumentation library information should be exported as labels
			for k := 0; k < metricSlice.Len(); k++ {
				metric := metricSlice.At(k)

				promName := "default-prom-name"

				// handle individual metrics based on type
				//exhaustive:enforce
				switch metric.Type() {
				case pmetric.MetricTypeExponentialHistogram:
					dataPoints := metric.ExponentialHistogram().DataPoints()
					if dataPoints.Len() == 0 {
						errs = multierr.Append(errs, fmt.Errorf("empty data points. %s is dropped", metric.Name()))
						break
					}
					errs = multierr.Append(errs, c.addExponentialHistogramDataPoints(
						dataPoints,
						resource,
						settings,
						promName,
					))
				default:
					errs = multierr.Append(errs, errors.New("unsupported metric type"))
				}
			}
		}
		addResourceTargetInfo(resource, settings, mostRecentTimestamp, c)
	}

	return
}
