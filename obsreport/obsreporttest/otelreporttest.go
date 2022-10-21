package obsreporttest

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/config"
)

type OtelMetricsFetcher struct {
	promHandler http.Handler
}

type exportedMetric struct {
	metric     *io_prometheus_client.MetricFamily
	timeseries map[attribute.Set]*io_prometheus_client.Metric
}

// CheckReceiverTraces checks that for the current exported values for trace receiver metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (mf *OtelMetricsFetcher) CheckReceiverTraces(receiver config.ComponentID, protocol string, acceptedSpans, droppedSpans int64) error {
	receiverAttrs := attributesForReceiverView(receiver, protocol)
	return multierr.Combine(
		mf.CheckCounter("receiver_accepted_spans", acceptedSpans, receiverAttrs),
		mf.CheckCounter("receiver_refused_spans", droppedSpans, receiverAttrs))
}

// CheckReceiverLogs checks that for the current exported values for logs receiver metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (mf *OtelMetricsFetcher) CheckReceiverLogs(receiver config.ComponentID, protocol string, acceptedLogRecords, droppedLogRecords int64) error {
	receiverAttrs := attributesForReceiverView(receiver, protocol)
	return multierr.Combine(
		mf.CheckCounter("receiver_accepted_log_records", acceptedLogRecords, receiverAttrs),
		mf.CheckCounter("receiver_refused_log_records", droppedLogRecords, receiverAttrs))
}

// CheckReceiverMetrics checks that for the current exported values for metrics receiver metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (mf *OtelMetricsFetcher) CheckReceiverMetrics(receiver config.ComponentID, protocol string, acceptedMetricPoints, droppedMetricPoints int64) error {
	receiverAttrs := attributesForReceiverView(receiver, protocol)
	return multierr.Combine(
		mf.CheckCounter("receiver_accepted_metric_points", acceptedMetricPoints, receiverAttrs),
		mf.CheckCounter("receiver_refused_metric_points", droppedMetricPoints, receiverAttrs))
}

// CheckCounter compares the value for a given metric and a set of attributes,
// if the counter value matches with the exported value by the OpenTelemetry Go SDK.
func (mf *OtelMetricsFetcher) CheckCounter(metric string, value int64, attrs []attribute.KeyValue) error {
	ts, err := mf.getMetric(metric, io_prometheus_client.MetricType_COUNTER, attrs)
	if err != nil {
		return err
	}

	fv := float64(value)
	if ts.GetCounter().GetValue() != fv {
		return fmt.Errorf("values for metric %s did no match, wanted %f got %f", metric, fv, ts.GetCounter().GetValue())
	}

	return nil
}

// getMetric returns the metric time series that matches the given name, type and set of attributes
// it fetches data from the prometheus endpoint and parse them, ideally OTel Go should provide a MeterRecorder of some kind.
func (mf *OtelMetricsFetcher) getMetric(expectedName string, expectedType io_prometheus_client.MetricType, expectedAttrs []attribute.KeyValue) (*io_prometheus_client.Metric, error) {
	parsed, err := fetchPrometheusMetrics(mf.promHandler)
	if err != nil {
		return nil, err
	}

	metricFamily, ok := parsed[expectedName]
	if !ok {
		return nil, fmt.Errorf("metric '%s' not found", expectedName)
	}

	if metricFamily.Type.String() != expectedType.String() {
		return nil, fmt.Errorf("metric '%v' has type '%s' instead of '%s'", expectedName, metricFamily.Type.String(), expectedType.String())
	}

	expectedSet := attribute.NewSet(expectedAttrs...)

	for _, metric := range metricFamily.Metric {
		var attrs []attribute.KeyValue

		for _, label := range metric.Label {
			attrs = append(attrs, attribute.String(label.GetName(), label.GetValue()))
		}
		set := attribute.NewSet(attrs...)

		if expectedSet.Equals(&set) {
			return metric, nil
		}
	}

	return nil, fmt.Errorf("metric '%s' doesn't have a timeseries with the given attributes: %s", expectedName, expectedSet.Encoded(attribute.DefaultEncoder()))
}

func fetchPrometheusMetrics(handler http.Handler) (map[string]*io_prometheus_client.MetricFamily, error) {
	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		return nil, err
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	var parser expfmt.TextParser
	return parser.TextToMetricFamilies(rr.Body)
}
