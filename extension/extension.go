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

package extension // import "go.opentelemetry.io/collector/extension"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
)

// Extension is the interface for objects hosted by the OpenTelemetry Collector that
// don't participate directly on data pipelines but provide some functionality
// to the service, examples: health check endpoint, z-pages, etc.
type Extension = component.Extension //nolint:staticcheck

// PipelineWatcher is an extra interface for Extension hosted by the OpenTelemetry
// Collector that is to be implemented by extensions interested in changes to pipeline
// states. Typically this will be used by extensions that change their behavior if data is
// being ingested or not, e.g.: a k8s readiness probe.
type PipelineWatcher interface {
	// Ready notifies the Extension that all pipelines were built and the
	// receivers were started, i.e.: the service is ready to receive data
	// (note that it may already have received data when this method is called).
	Ready() error

	// NotReady notifies the Extension that all receivers are about to be stopped,
	// i.e.: pipeline receivers will not accept new data.
	// This is sent before receivers are stopped, so the Extension can take any
	// appropriate actions before that happens.
	NotReady() error
}

// CreateSettings is passed to Factory.Create(...) function.
type CreateSettings = component.ExtensionCreateSettings //nolint:staticcheck

// CreateFunc is the equivalent of Factory.Create(...) function.
type CreateFunc func(context.Context, CreateSettings, component.Config) (Extension, error)

// CreateExtension implements ExtensionFactory.CreateExtension.
func (f CreateFunc) CreateExtension(ctx context.Context, set CreateSettings, cfg component.Config) (Extension, error) {
	return f(ctx, set, cfg)
}

// Factory is a factory for extensions to the service.
type Factory = component.ExtensionFactory //nolint:staticcheck

type factory struct {
	component.Factory
	cfgType component.Type
	component.CreateDefaultConfigFunc
	CreateFunc
	extensionStability component.StabilityLevel
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

func (f *factory) ExtensionStability() component.StabilityLevel {
	return f.extensionStability
}

// NewFactory returns a new Factory  based on this configuration.
func NewFactory(
	cfgType component.Type,
	createDefaultConfig component.CreateDefaultConfigFunc,
	createServiceExtension CreateFunc,
	sl component.StabilityLevel) Factory {
	return &factory{
		cfgType:                 cfgType,
		CreateDefaultConfigFunc: createDefaultConfig,
		CreateFunc:              createServiceExtension,
		extensionStability:      sl,
	}
}

// MakeFactoryMap takes a list of factories and returns a map with Factory type as keys.
// It returns a non-nil error when there are factories with duplicate type.
func MakeFactoryMap(factories ...Factory) (map[component.Type]Factory, error) {
	fMap := map[component.Type]Factory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate extension factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
