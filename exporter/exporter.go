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

package exporter // import "go.opentelemetry.io/collector/exporter"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
)

// Traces is an exporter that can consume traces.
type Traces = component.TracesExporter //nolint:staticcheck

// Metrics is an exporter that can consume metrics.
type Metrics = component.MetricsExporter //nolint:staticcheck

// Logs is an exporter that can consume logs.
type Logs = component.LogsExporter //nolint:staticcheck

// CreateSettings configures exporter creators.
type CreateSettings = component.ExporterCreateSettings //nolint:staticcheck

// Factory is factory interface for exporters.
//
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
type Factory = component.ExporterFactory //nolint:staticcheck

// FactoryOption apply changes to Factory.
type FactoryOption interface {
	// applyExporterFactoryOption applies the option.
	applyExporterFactoryOption(o *factory)
}

var _ FactoryOption = (*factoryOptionFunc)(nil)

// factoryOptionFunc is an ExporterFactoryOption created through a function.
type factoryOptionFunc func(*factory)

func (f factoryOptionFunc) applyExporterFactoryOption(o *factory) {
	f(o)
}

// CreateTracesFunc is the equivalent of Factory.CreateTraces.
type CreateTracesFunc func(context.Context, CreateSettings, component.Config) (Traces, error)

// CreateTracesExporter implements ExporterFactory.CreateTracesExporter().
func (f CreateTracesFunc) CreateTracesExporter(ctx context.Context, set CreateSettings, cfg component.Config) (Traces, error) {
	if f == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg)
}

// CreateMetricsFunc is the equivalent of Factory.CreateMetrics.
type CreateMetricsFunc func(context.Context, CreateSettings, component.Config) (Metrics, error)

// CreateMetricsExporter implements ExporterFactory.CreateMetricsExporter().
func (f CreateMetricsFunc) CreateMetricsExporter(ctx context.Context, set CreateSettings, cfg component.Config) (Metrics, error) {
	if f == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg)
}

// CreateLogsFunc is the equivalent of Factory.CreateLogs.
type CreateLogsFunc func(context.Context, CreateSettings, component.Config) (Logs, error)

// CreateLogsExporter implements Factory.CreateLogsExporter().
func (f CreateLogsFunc) CreateLogsExporter(ctx context.Context, set CreateSettings, cfg component.Config) (Logs, error) {
	if f == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg)
}

type factory struct {
	component.Factory
	cfgType component.Type
	component.CreateDefaultConfigFunc
	CreateTracesFunc
	tracesStabilityLevel component.StabilityLevel
	CreateMetricsFunc
	metricsStabilityLevel component.StabilityLevel
	CreateLogsFunc
	logsStabilityLevel component.StabilityLevel
}

func (f *factory) Type() component.Type {
	return f.cfgType
}

// CreateDefaultConfig creates the default configuration for the Component.
//
// TODO: Remove this when we remove the private func from component.Factory and add it to every specialized Factory.
func (f *factory) CreateDefaultConfig() component.Config {
	return f.CreateDefaultConfigFunc()
}

func (f *factory) TracesExporterStability() component.StabilityLevel {
	return f.tracesStabilityLevel
}

func (f *factory) MetricsExporterStability() component.StabilityLevel {
	return f.metricsStabilityLevel
}

func (f *factory) LogsExporterStability() component.StabilityLevel {
	return f.logsStabilityLevel
}

// WithTraces overrides the default "error not supported" implementation for CreateTracesExporter and the default "undefined" stability level.
func WithTraces(createTraces CreateTracesFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.tracesStabilityLevel = sl
		o.CreateTracesFunc = createTraces
	})
}

// WithMetrics overrides the default "error not supported" implementation for CreateMetricsExporter and the default "undefined" stability level.
func WithMetrics(createMetrics CreateMetricsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.metricsStabilityLevel = sl
		o.CreateMetricsFunc = createMetrics
	})
}

// WithLogs overrides the default "error not supported" implementation for CreateLogsExporter and the default "undefined" stability level.
func WithLogs(createLogs CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factory) {
		o.logsStabilityLevel = sl
		o.CreateLogsFunc = createLogs
	})
}

// NewFactory returns a Factory.
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	f := &factory{
		cfgType:                 cfgType,
		CreateDefaultConfigFunc: createDefaultConfig,
	}
	for _, opt := range options {
		opt.applyExporterFactoryOption(f)
	}
	return f
}

// MakeFactoryMap takes a list of factories and returns a map with Factory type as keys.
// It returns a non-nil error when there are factories with duplicate type.
func MakeFactoryMap(factories ...Factory) (map[component.Type]Factory, error) {
	fMap := map[component.Type]Factory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate exporter factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
