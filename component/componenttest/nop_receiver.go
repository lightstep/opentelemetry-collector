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
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

// Deprecated: [v0.67.0] use receivertest.NewNopCreateSettings.
func NewNopReceiverCreateSettings() receiver.CreateSettings {
	return receiver.CreateSettings{
		TelemetrySettings: NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

// Deprecated: [v0.67.0] use receivertest.NewNopFactory
func NewNopReceiverFactory() receiver.Factory {
	return receiver.NewFactory(
		"nop",
		func() component.Config { return &nopConfig{} },
		receiver.WithTraces(createTraces, component.StabilityLevelStable),
		receiver.WithMetrics(createMetrics, component.StabilityLevelStable),
		receiver.WithLogs(createLogs, component.StabilityLevelStable))
}

func createTraces(context.Context, receiver.CreateSettings, component.Config, consumer.Traces) (receiver.Traces, error) {
	return nopReceiverInstance, nil
}

func createMetrics(context.Context, receiver.CreateSettings, component.Config, consumer.Metrics) (receiver.Metrics, error) {
	return nopReceiverInstance, nil
}

func createLogs(context.Context, receiver.CreateSettings, component.Config, consumer.Logs) (receiver.Logs, error) {
	return nopReceiverInstance, nil
}

var nopReceiverInstance = &nopReceiver{}

// nopReceiver stores consumed traces and metrics for testing purposes.
type nopReceiver struct {
	nopComponent
}
