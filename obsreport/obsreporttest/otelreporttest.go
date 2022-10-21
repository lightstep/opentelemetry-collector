// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package obsreporttest // import "go.opentelemetry.io/collector/obsreport/obsreporttest"

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/config"
)

// PrometheusChecker is used to assert exported metrics from a prometheus handler.
type PrometheusChecker struct {
	promHandler http.Handler
}

// CheckReceiverTraces checks that for the current exported values for trace receiver metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (mf *PrometheusChecker) CheckReceiverTraces(receiver config.ComponentID, protocol string, acceptedSpans, droppedSpans int64) error {
	receiverAttrs := attributesForReceiverMetrics(receiver, protocol)
	return multierr.Combine(
		mf.checkCounter("receiver_accepted_spans", acceptedSpans, receiverAttrs),
		mf.checkCounter("receiver_refused_spans", droppedSpans, receiverAttrs))
}

// CheckReceiverLogs checks that for the current exported values for logs receiver metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (mf *PrometheusChecker) CheckReceiverLogs(receiver config.ComponentID, protocol string, acceptedLogRecords, droppedLogRecords int64) error {
	receiverAttrs := attributesForReceiverMetrics(receiver, protocol)
	return multierr.Combine(
		mf.checkCounter("receiver_accepted_log_records", acceptedLogRecords, receiverAttrs),
		mf.checkCounter("receiver_refused_log_records", droppedLogRecords, receiverAttrs))
}

// CheckReceiverMetrics checks that for the current exported values for metrics receiver metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
func (mf *PrometheusChecker) CheckReceiverMetrics(receiver config.ComponentID, protocol string, acceptedMetricPoints, droppedMetricPoints int64) error {
	receiverAttrs := attributesForReceiverMetrics(receiver, protocol)
	return multierr.Combine(
		mf.checkCounter("receiver_accepted_metric_points", acceptedMetricPoints, receiverAttrs),
		mf.checkCounter("receiver_refused_metric_points", droppedMetricPoints, receiverAttrs))
}

// checkCounter compares the value for a given metric and a set of attributes,
// if the counter value matches with the exported value by the Prometheus handler.
func (mf *PrometheusChecker) checkCounter(expectedMetric string, value int64, attrs []attribute.KeyValue) error {
	ts, err := mf.getMetric(expectedMetric, io_prometheus_client.MetricType_COUNTER, attrs)
	if err != nil {
		return err
	}

	expected := float64(value)
	if math.Abs(expected-ts.GetCounter().GetValue()) < 0.0001 {
		return fmt.Errorf("values for metric '%s' did no match, expected '%f' got '%f'", expectedMetric, expected, ts.GetCounter().GetValue())
	}

	return nil
}

// getMetric returns the metric time series that matches the given name, type and set of attributes
// it fetches data from the prometheus endpoint and parse them, ideally OTel Go should provide a MeterRecorder of some kind.
func (mf *PrometheusChecker) getMetric(expectedName string, expectedType io_prometheus_client.MetricType, expectedAttrs []attribute.KeyValue) (*io_prometheus_client.Metric, error) {
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

// attributesForReceiverMetrics returns the attributes that are needed for the receiver metrics.
func attributesForReceiverMetrics(receiver config.ComponentID, transport string) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 2)

	attrs = append(attrs, attribute.String(receiverTag.Name(), receiver.String()))
	if transport != "" {
		attrs = append(attrs, attribute.String(transportTag.Name(), transport))
	}

	return attrs
}
