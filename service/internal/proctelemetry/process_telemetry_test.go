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

package proctelemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/metric"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/stats/view"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
)

type testTelemetry struct {
	component.TelemetrySettings
	views           []*view.View
	promHandler     http.Handler
	meterProvider   *sdkmetric.MeterProvider
	expectedMetrics []string
}

var expectedMetrics = []string{
	// Changing a metric name is a breaking change.
	// Adding new metrics is ok as long it follows the conventions described at
	// https://pkg.go.dev/go.opentelemetry.io/collector/obsreport?tab=doc#hdr-Naming_Convention_for_New_Metrics
	"process/uptime",
	"process/runtime/heap_alloc_bytes",
	"process/runtime/total_alloc_bytes",
	"process/runtime/total_sys_memory_bytes",
	"process/cpu_seconds",
	"process/memory/rss",
}

var otelExpectedMetrics = []string{
	// OTel Go adds `_total` suffix
	"process_uptime",
	"process_runtime_heap_alloc_bytes",
	"process_runtime_total_alloc_bytes",
	"process_runtime_total_sys_memory_bytes",
	"process_cpu_seconds",
	"process_memory_rss",
}

func setupTelemetry(t *testing.T) testTelemetry {
	settings := testTelemetry{
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		expectedMetrics:   otelExpectedMetrics,
	}
	settings.TelemetrySettings.MetricsLevel = configtelemetry.LevelNormal

	settings.views = obsreportconfig.AllViews(configtelemetry.LevelNormal)
	err := view.Register(settings.views...)
	require.NoError(t, err)

	promReg := prometheus.NewRegistry()
	exporter, err := otelprom.New(otelprom.WithRegisterer(promReg), otelprom.WithoutUnits())
	require.NoError(t, err)

	settings.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource.Empty()),
		sdkmetric.WithReader(exporter),
	)
	settings.TelemetrySettings.MeterProvider = settings.meterProvider

	settings.promHandler = promhttp.HandlerFor(promReg, promhttp.HandlerOpts{})

	t.Cleanup(func() { assert.NoError(t, settings.meterProvider.Shutdown(context.Background())) })

	return settings
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

func TestOtelProcessTelemetry(t *testing.T) {
	tel := setupTelemetry(t)

	pm := &processMetrics{
		startTimeUnixNano: time.Now().UnixNano(),
		ballastSizeBytes:  0,
		ms:                &runtime.MemStats{},
		useOtelForMetrics: true,
	}

	pm.meter = tel.MeterProvider.Meter("test")
	require.NoError(t, pm.RegisterProcessMetrics(context.Background(), nil))

	mp, err := fetchPrometheusMetrics(tel.promHandler)
	require.NoError(t, err)

	for _, metricName := range tel.expectedMetrics {
		metric, ok := mp[metricName]
		if !ok {
			withSuffix := metricName + "_total"
			metric, ok = mp[withSuffix]
		}
		require.True(t, ok)
		require.True(t, len(metric.Metric) == 1)
		// require.True(t, metric.GetType() == io_prometheus_client.MetricType_COUNTER)
		var metricValue float64
		if metric.GetType() == io_prometheus_client.MetricType_COUNTER {
			metricValue = metric.Metric[0].GetCounter().GetValue()
		} else {
			metricValue = metric.Metric[0].GetGauge().GetValue()
		}
		assert.True(t, metricValue > 0)
	}
}

func TestOCProcessTelemetry(t *testing.T) {
	registry := metric.NewRegistry()
	pm := &processMetrics{
		startTimeUnixNano: time.Now().UnixNano(),
		ballastSizeBytes:  0,
		ms:                &runtime.MemStats{},
		useOtelForMetrics: false,
	}

	require.NoError(t, pm.RegisterProcessMetrics(context.Background(), registry))

	// Check that the metrics are actually filled.
	<-time.After(200 * time.Millisecond)

	metrics := registry.Read()

	for _, metricName := range expectedMetrics {
		m := findMetric(metrics, metricName)
		require.NotNil(t, m)
		require.Len(t, m.TimeSeries, 1)
		ts := m.TimeSeries[0]
		assert.Len(t, ts.LabelValues, 0)
		require.Len(t, ts.Points, 1)

		var value float64
		if metricName == "process/uptime" || metricName == "process/cpu_seconds" {
			value = ts.Points[0].Value.(float64)
		} else {
			value = float64(ts.Points[0].Value.(int64))
		}

		if metricName == "process/uptime" || metricName == "process/cpu_seconds" {
			// This likely will still be zero when running the test.
			assert.True(t, value >= 0, metricName)
			continue
		}

		assert.True(t, value > 0, metricName)
	}
}

func TestProcessTelemetryFailToRegister(t *testing.T) {
	pm := &processMetrics{
		useOtelForMetrics: false,
	}
	for _, metricName := range expectedMetrics {
		t.Run(metricName, func(t *testing.T) {
			registry := metric.NewRegistry()
			_, err := registry.AddFloat64Gauge(metricName)
			require.NoError(t, err)
			assert.Error(t, pm.RegisterProcessMetrics(context.Background(), registry))
		})
	}
}

func findMetric(metrics []*metricdata.Metric, name string) *metricdata.Metric {
	for _, m := range metrics {
		if m.Descriptor.Name == name {
			return m
		}
	}
	return nil
}
