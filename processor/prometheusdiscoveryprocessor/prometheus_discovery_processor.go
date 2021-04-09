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

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
)

const (
	jobKey      = "job"
	instanceKey = "instance"
)

type prometheusDiscoveryProcessor struct {
	cfg            *Config
	logger         *zap.Logger
	attributeCache map[*cacheKey]pdata.AttributeMap
}

type cacheKey struct {
	job      string
	instance string
}

func newPrometheusDiscoveryProcessor(logger *zap.Logger, cfg *Config) (*prometheusDiscoveryProcessor, error) {

	logger.Info("Prometheus Discovery Processor configured")

	return &prometheusDiscoveryProcessor{
		cfg:            cfg,
		logger:         logger,
		attributeCache: make(map[*cacheKey]pdata.AttributeMap),
	}, nil
}

// ProcessMetrics looks for "up" metrics and when found will store attributes collected from service
// discovery. Non "up" metrics with matching job and instance attributes will be enriched with the
// discovered attributes.
func (pdp *prometheusDiscoveryProcessor) ProcessMetrics(_ context.Context, pdm pdata.Metrics) (pdata.Metrics, error) {
	rms := pdm.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)

		attrs := rm.Resource().Attributes()

		if key, ok := getCacheKeyForResource(rm.Resource()); ok {

			if source, ok := attrs.Get("source"); ok && source.StringVal() == "prometheus_discovery" {
				// attrs came from prometheus_discovery; cache them
				pdp.attributeCache[key] = attrs
			} else {
				// these are "normal" metrics, enrich the resource with cached attrs
				pdp.enrichResource(rm.Resource())
			}
		}

		// I don't think we need to go this deep given that we are just dealing with resources, for now
		/*
			ilms := rm.InstrumentationLibraryMetrics()
			for j := 0; j < ilms.Len(); j++ {
				ms := ilms.At(j).Metrics()
				for k := 0; k < ms.Len(); k++ {
					met := ms.At(k)
					if met.Name() == "present" {

					}
				}
			}
		*/
	}

	return pdm, nil
}

// enrichResource inspects the incoming pdata.Resource for job and instance labels. If found, it
// looks up cached attributes from service discovery, and merge them with the Resources attributes.
func (pdp *prometheusDiscoveryProcessor) enrichResource(resource pdata.Resource) {
	key, ok := getCacheKeyForResource(resource)
	if !ok {
		return
	}

	cachedAttributes, ok := pdp.attributeCache[key]
	if !ok {
		return
	}

	cachedAttributes.CopyTo(resource.Attributes())
}

func getCacheKeyForResource(resource pdata.Resource) (*cacheKey, bool) {
	jobValue, ok := resource.Attributes().Get(jobKey)

	if !ok {
		return nil, false
	}

	instanceValue, ok := resource.Attributes().Get(instanceKey)

	if !ok {
		return nil, false
	}

	// @todo: don't assume StringVal will work
	return &cacheKey{job: jobValue.StringVal(), instance: instanceValue.StringVal()}, false
}
