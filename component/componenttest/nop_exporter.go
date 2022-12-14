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

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
)

// NewNopExporterCreateSettings returns a new nop settings for Create*Exporter functions.
func NewNopExporterCreateSettings() exporter.CreateSettings {
	return exporter.CreateSettings{
		TelemetrySettings: NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

type nopExporterConfig struct {
	config.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
}

// NewNopExporterFactory returns an exporter.Factory that constructs nop exporters.
func NewNopExporterFactory() exporter.Factory {
	return exporter.NewFactory(
		"nop",
		func() component.Config {
			return &nopExporterConfig{
				ExporterSettings: config.NewExporterSettings(component.NewID("nop")),
			}
		},
		exporter.WithTraces(createTracesExporter, component.StabilityLevelStable),
		exporter.WithMetrics(createMetricsExporter, component.StabilityLevelStable),
		exporter.WithLogs(createLogsExporter, component.StabilityLevelStable),
	)
}

func createTracesExporter(context.Context, exporter.CreateSettings, component.Config) (exporter.Traces, error) {
	return nopExporterInstance, nil
}

func createMetricsExporter(context.Context, exporter.CreateSettings, component.Config) (exporter.Metrics, error) {
	return nopExporterInstance, nil
}

func createLogsExporter(context.Context, exporter.CreateSettings, component.Config) (exporter.Logs, error) {
	return nopExporterInstance, nil
}

var nopExporterInstance = &nopExporter{
	Consumer: consumertest.NewNop(),
}

// nopExporter stores consumed traces and metrics for testing purposes.
type nopExporter struct {
	nopComponent
	consumertest.Consumer
}
