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
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/prometheusdiscoveryreceiver/internal"
	"go.uber.org/zap"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

// pReceiver is the type that provides Prometheus scraper/receiver functionality.
type pReceiver struct {
	cfg        *Config
	consumer   consumer.Metrics
	cancelFunc context.CancelFunc

	logger *zap.Logger
}

// New creates a new prometheus.Receiver reference.
func newPrometheusDiscoveryReceiver(logger *zap.Logger, cfg *Config, next consumer.Metrics) *pReceiver {
	pr := &pReceiver{
		cfg:      cfg,
		consumer: next,
		logger:   logger,
	}
	return pr
}

// Start is the method that starts Prometheus discovery and it
// is controlled by having previously defined a Configuration using perhaps New.
func (r *pReceiver) Start(_ context.Context, host component.Host) error {
	discoveryCtx, cancel := context.WithCancel(context.Background())
	r.cancelFunc = cancel

	logger := internal.NewZapToGokitLogAdapter(r.logger)

	discoveryManager := discovery.NewManager(discoveryCtx, logger)
	discoveryCfg := make(map[string]discovery.Configs)
	for _, scrapeConfig := range r.cfg.PrometheusConfig.ScrapeConfigs {
		discoveryCfg[scrapeConfig.JobName] = scrapeConfig.ServiceDiscoveryConfigs
	}
	if err := discoveryManager.ApplyConfig(discoveryCfg); err != nil {
		return err
	}
	go func() {
		if err := discoveryManager.Run(); err != nil {
			r.logger.Error("Discovery manager failed", zap.Error(err))
			host.ReportFatalError(err)
		}
	}()

	go r.generatePresent(discoveryCtx, discoveryManager.SyncCh())

	return nil
}

func (r *pReceiver) generatePresent(ctx context.Context, syncCh <-chan map[string][]*targetgroup.Group) {
	for {
		select {
		case <-ctx.Done():
			return
		case tgs := <-syncCh:
			r.formatGroups(ctx, tgs)
		}
	}
}

// Shutdown stops and cancels the underlying Prometheus scrapers.
func (r *pReceiver) Shutdown(context.Context) error {
	r.cancelFunc()
	return nil
}

func (r *pReceiver) formatGroups(ctx context.Context, tgs map[string][]*targetgroup.Group) {
	ts := pdata.TimestampFromTime(time.Now())

	ms := pdata.NewMetrics()
	resourceMetrics := ms.ResourceMetrics()

	for job, groups := range tgs {
		for _, group := range groups {
			attrs := pdata.NewAttributeMap()

			//TODO: should we validate name and value?
			//TODO: find a way to create this labels once, for a group.
			for name, value := range group.Labels {
				attrs.InsertString(cleanLabelName(string(name)), string(value))
			}

			for _, target := range group.Targets {
				tAttrs := pdata.NewAttributeMap()
				attrs.CopyTo(tAttrs)

				for name, value := range target {
					if string(name) == "__address__" {
						// The target address is used to identify an instance.
						tAttrs.InsertString("instance", string(value))
					} else {
						tAttrs.InsertString(cleanLabelName(string(name)), string(value))
					}
				}

				tAttrs.InsertString("job", job)
				tAttrs.InsertString("source", "prometheus_discovery")
				tAttrs.InsertString("instance", string(target["__address__"]))

				resourceMetrics.Append(newPresentResourceMetric(tAttrs, ts))
			}
		}
	}

	// do some error handling here.
	_ = r.consumer.ConsumeMetrics(ctx, ms)
}

func newPresentResourceMetric(attr pdata.AttributeMap, ts pdata.Timestamp) pdata.ResourceMetrics {
	m := pdata.NewResourceMetrics()
	attr.CopyTo(m.Resource().Attributes())
	m.Resource().Attributes()

	metric := pdata.NewMetric()
	metric.SetDataType(pdata.MetricDataTypeIntGauge)
	metric.SetName("present")
	metric.SetDescription("Clear description of what present means")
	metric.IntGauge().DataPoints().Append(newIntDataPoint(1, ts))

	instMetrics := pdata.NewInstrumentationLibraryMetrics()
	instMetrics.Metrics().Append(metric)

	m.InstrumentationLibraryMetrics().Append(instMetrics)
	return m
}

func newIntDataPoint(value int64, timestamp pdata.Timestamp) pdata.IntDataPoint {
	intDataPoint := pdata.NewIntDataPoint()
	intDataPoint.SetValue(value)
	intDataPoint.SetTimestamp(timestamp)
	return intDataPoint
}

func cleanLabelName(str string) string {
	return strings.TrimPrefix(str, "__meta")
}
