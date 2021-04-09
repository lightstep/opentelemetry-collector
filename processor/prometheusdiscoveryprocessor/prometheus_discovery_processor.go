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

package prometheusdiscoveryprocessor

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
)

const (
	jobKey      = "job"
	instanceKey = "instance"
)

type cacheKey struct {
	job      string
	instance string
}

var emptyCacheKey = cacheKey{}

type attributeCache struct {
	mutex *sync.Mutex
	cache map[cacheKey]pdata.AttributeMap
}

func (ac *attributeCache) get(key cacheKey) (pdata.AttributeMap, bool) {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()
	attributes, ok := ac.cache[key]
	return attributes, ok
}

func (ac *attributeCache) set(key cacheKey, attributes pdata.AttributeMap) {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()
	ac.cache[key] = attributes
}

func newAttributeCache() *attributeCache {
	return &attributeCache{
		mutex: &sync.Mutex{},
		cache: make(map[cacheKey]pdata.AttributeMap),
	}
}

type prometheusDiscoveryProcessor struct {
	cfg            *Config
	logger         *zap.Logger
	attributeCache *attributeCache
}

func newPrometheusDiscoveryProcessor(logger *zap.Logger, cfg *Config) (*prometheusDiscoveryProcessor, error) {

	logger.Info("Prometheus Discovery Processor configured")

	return &prometheusDiscoveryProcessor{
		cfg:            cfg,
		logger:         logger,
		attributeCache: newAttributeCache(),
	}, nil
}

// ProcessMetrics inspects the Resource on incoming pdata to determine if it came from prometheus
// discovery or if it contains regular metrics that should be enriched. If the resource has the
// sentinel attribute `source: prometheus_discovery` along with job and instance attributes, the
// resource labels are cached. Normal pdata  with matching job and instance attributes will be have
// their resource enriched with the cached attributes.
func (pdp *prometheusDiscoveryProcessor) ProcessMetrics(_ context.Context, pdm pdata.Metrics) (pdata.Metrics, error) {
	rms := pdm.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		resource := rm.Resource()
		attrs := resource.Attributes()

		key, ok := getCacheKeyForResource(resource)

		if !ok {
			continue
		}

		if source, ok := attrs.Get("source"); ok && source.StringVal() == "prometheus_discovery" {
			// attrs came from prometheus_discovery; cache them
			pdp.attributeCache.set(key, attrs)
		} else if cachedAttributes, ok := pdp.attributeCache.get(key); ok {
			// these are "normal" metrics, enrich the resource with cached attrs
			cachedAttributes.CopyTo(resource.Attributes())
		}

	}

	return pdm, nil
}

func getCacheKeyForResource(resource pdata.Resource) (cacheKey, bool) {
	jobValue, ok := resource.Attributes().Get(jobKey)

	if !ok {
		return emptyCacheKey, false
	}

	instanceValue, ok := resource.Attributes().Get(instanceKey)

	if !ok {
		return emptyCacheKey, false
	}

	// @todo: don't assume StringVal will work
	return cacheKey{job: jobValue.StringVal(), instance: instanceValue.StringVal()}, true
}
