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

package prometheusdiscoveryreceiver

import (
	"context"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"path"
	"testing"
	"time"

	"go.uber.org/zap"
)

var logger = zap.NewNop()

func TestEndToEnd(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[typeStr] = factory
	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config_discovery.yaml"), factories)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	config := cfg.Receivers["prometheus_discovery/discovery"].(*Config)

	cms := new(consumertest.MetricsSink)
	r := newPrometheusDiscoveryReceiver(logger, config, cms)

	err = r.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	// default sync time of the discovery channel is 5 seconds.
	time.Sleep(6 * time.Second)

	dataPoints := cms.AllMetrics()[0].ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).IntGauge().DataPoints()
	dataPoints.At(0)
	// TODO: verify these dataPoints, probably need to fix the formatter.
}

func TestFormatter(t *testing.T) {
	tg := map[string][]*targetgroup.Group{
		"pod/default/testpod": {{
			Targets: []model.LabelSet{
				{
					"__address__":                                   "1.2.3.4:9000",
					"__meta_kubernetes_pod_container_name":          "testcontainer0",
					"__meta_kubernetes_pod_container_port_name":     "testport0",
					"__meta_kubernetes_pod_container_port_number":   "9000",
					"__meta_kubernetes_pod_container_port_protocol": "TCP",
					"__meta_kubernetes_pod_container_init":          "false",
				},
				{
					"__address__":                                   "1.2.3.4:9001",
					"__meta_kubernetes_pod_container_name":          "testcontainer0",
					"__meta_kubernetes_pod_container_port_name":     "testport1",
					"__meta_kubernetes_pod_container_port_number":   "9001",
					"__meta_kubernetes_pod_container_port_protocol": "UDP",
					"__meta_kubernetes_pod_container_init":          "false",
				},
				{
					"__address__":                          "1.2.3.4",
					"__meta_kubernetes_pod_container_name": "testcontainer1",
					"__meta_kubernetes_pod_container_init": "false",
				},
			},
			Labels: model.LabelSet{
				"__meta_kubernetes_pod_name":                              "testpod",
				"__meta_kubernetes_namespace":                             "default",
				"__meta_kubernetes_pod_label_test_label":                  "testvalue",
				"__meta_kubernetes_pod_labelpresent_test_label":           "true",
				"__meta_kubernetes_pod_annotation_test_annotation":        "testannotationvalue",
				"__meta_kubernetes_pod_annotationpresent_test_annotation": "true",
				"__meta_kubernetes_pod_node_name":                         "testnode",
				"__meta_kubernetes_pod_ip":                                "1.2.3.4",
				"__meta_kubernetes_pod_host_ip":                           "2.3.4.5",
				"__meta_kubernetes_pod_ready":                             "true",
				"__meta_kubernetes_pod_phase":                             "Running",
				"__meta_kubernetes_pod_uid":                               "abc123",
				"__meta_kubernetes_pod_controller_kind":                   "testcontrollerkind",
				"__meta_kubernetes_pod_controller_name":                   "testcontrollername",
			},
			Source: "pod/default/testpod",
		}},
	}

	cms := new(consumertest.MetricsSink)
	r := newPrometheusDiscoveryReceiver(logger, &Config{}, cms)

	r.formatGroups(context.Background(), tg)

	m := cms.AllMetrics()
	rscMetricsSlice := m[0].ResourceMetrics()

	require.Equal(t, 3, rscMetricsSlice.Len())

	for i := 0; i < rscMetricsSlice.Len(); i++ {
		rscMetric := rscMetricsSlice.At(i)
		rscAttrs := rscMetric.Resource().Attributes()
		metric := rscMetric.InstrumentationLibraryMetrics().At(0).Metrics().At(0)

		job, ok := rscAttrs.Get("job")
		require.Truef(t, ok, "resource must have job attribute")
		targets := tg[job.StringVal()][0]

		for name, value := range targets.Labels {
			attr, ok := rscAttrs.Get(cleanLabelName(string(name)))
			require.Truef(t, ok, "resource must have target group labels")
			require.Equal(t, string(value), attr.StringVal())
		}

		target := targets.Targets[i]
		for name, value := range target {
			if string(name) == "__address__" {
				continue
			}
			attr, ok := rscAttrs.Get(cleanLabelName(string(name)))
			require.Truef(t, ok, "resource must have target label: %v", name)
			require.Equal(t, string(value), attr.StringVal())
		}

		require.Equal(t, "present", metric.Name())
		require.Equal(t, int64(1), metric.IntGauge().DataPoints().At(0).Value())
	}
}
