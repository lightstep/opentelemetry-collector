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

package obsreport // import "go.opentelemetry.io/collector/obsreport"

import (
	"context"
	"strings"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"go.opentelemetry.io/otel/metric/unit"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

var (
	processorName  = "processor"
	processorScope = scopeName + nameSep + processorName
)

// BuildProcessorCustomMetricName is used to be build a metric name following
// the standards used in the Collector. The configType should be the same
// value used to identify the type on the config.
func BuildProcessorCustomMetricName(configType, metric string) string {
	componentPrefix := obsmetrics.ProcessorPrefix
	if !strings.HasSuffix(componentPrefix, obsmetrics.NameSep) {
		componentPrefix += obsmetrics.NameSep
	}
	if configType == "" {
		return componentPrefix
	}
	return componentPrefix + configType + obsmetrics.NameSep + metric
}

// Processor is a helper to add observability to a component.Processor.
type Processor struct {
	level    configtelemetry.Level
	mutators []tag.Mutator

	logger *zap.Logger

	useOtelForMetrics    bool
	otelAttrs            []attribute.KeyValue

	acceptedSpansCounter        syncint64.Counter
	refusedSpansCounter         syncint64.Counter
	droppedSpansCounter         syncint64.Counter
	acceptedMetricPointsCounter syncint64.Counter
	refusedMetricPointsCounter  syncint64.Counter
	droppedMetricPointsCounter  syncint64.Counter
	acceptedLogRecordsCounter   syncint64.Counter
	refusedLogRecordsCounter    syncint64.Counter
	droppedLogRecordsCounter    syncint64.Counter

}

// ProcessorSettings are settings for creating a Processor.
type ProcessorSettings struct {
	ProcessorID             component.ID
	ProcessorCreateSettings component.ProcessorCreateSettings
}

// NewProcessor creates a new Processor.
func NewProcessor(cfg ProcessorSettings) *Processor {
	return newProcessor(cfg, featuregate.GetRegistry())
}

func newProcessor(cfg ProcessorSettings, registry *featuregate.Registry) *Processor {
	proc := &Processor{
		level:    cfg.ProcessorCreateSettings.MetricsLevel,
		mutators: []tag.Mutator{tag.Upsert(obsmetrics.TagKeyProcessor, cfg.ProcessorID.String(), tag.WithTTL(tag.TTLNoPropagation))},
		logger:            cfg.ProcessorCreateSettings.Logger,
		useOtelForMetrics: registry.IsEnabled(obsreportconfig.UseOtelForInternalMetricsfeatureGateID),
		otelAttrs: []attribute.KeyValue{ 
			attribute.String(obsmetrics.ProcessorKey, cfg.ProcessorID.String()),
		},

	}

	proc.createOtelMetrics(cfg)

	return proc

}

func (proc *Processor) createOtelMetrics(cfg ProcessorSettings) {
	if !proc.useOtelForMetrics {
		return
	}

	meter := cfg.ProcessorCreateSettings.MeterProvider.Meter(processorScope)

	var err error
	handleError := func(metricName string, err error) {
		if err != nil {
			proc.logger.Warn("failed to create otel instrument", zap.Error(err), zap.String("metric", metricName))
		}
	}

	proc.acceptedSpansCounter, err = meter.SyncInt64().Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.AcceptedSpansKey,
		instrument.WithDescription("Number of spans successfully pushed into the next component in the pipeline."),
		instrument.WithUnit(unit.Dimensionless),
	)
	handleError(obsmetrics.ProcessorPrefix+obsmetrics.AcceptedSpansKey, err)

	proc.refusedSpansCounter, err = meter.SyncInt64().Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.RefusedSpansKey,
		instrument.WithDescription("Number of spans that were rejected by the next component in the pipeline."),
		instrument.WithUnit(unit.Dimensionless),
	)
	handleError(obsmetrics.ProcessorPrefix+obsmetrics.RefusedSpansKey, err)

	proc.droppedSpansCounter, err = meter.SyncInt64().Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.DroppedSpansKey,
		instrument.WithDescription("Number of spans that were dropped."),
		instrument.WithUnit(unit.Dimensionless),
	)
	handleError(obsmetrics.ProcessorPrefix+obsmetrics.DroppedSpansKey, err)

	proc.acceptedMetricPointsCounter, err = meter.SyncInt64().Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.AcceptedMetricPointsKey,
		instrument.WithDescription("Number of metric points successfully pushed into the next component in the pipeline."),
		instrument.WithUnit(unit.Dimensionless),
	)
	handleError(obsmetrics.ProcessorPrefix+obsmetrics.AcceptedMetricPointsKey, err)

	proc.refusedMetricPointsCounter, err = meter.SyncInt64().Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.RefusedMetricPointsKey,
		instrument.WithDescription("Number of metric points that were rejected by the next component in the pipeline."),
		instrument.WithUnit(unit.Dimensionless),
	)
	handleError(obsmetrics.ProcessorPrefix+obsmetrics.RefusedMetricPointsKey, err)

	proc.droppedMetricPointsCounter, err = meter.SyncInt64().Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.DroppedMetricPointsKey,
		instrument.WithDescription("Number of metric points that were dropped."),
		instrument.WithUnit(unit.Dimensionless),
	)
	handleError(obsmetrics.ProcessorPrefix+obsmetrics.DroppedMetricPointsKey, err)

	proc.acceptedLogRecordsCounter, err = meter.SyncInt64().Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.AcceptedLogRecordsKey,
		instrument.WithDescription("Number of log records successfully pushed into the next component in the pipeline."),
		instrument.WithUnit(unit.Dimensionless),
	)
	handleError(obsmetrics.ProcessorPrefix+obsmetrics.AcceptedLogRecordsKey, err)

	proc.refusedLogRecordsCounter, err = meter.SyncInt64().Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.RefusedLogRecordsKey,
		instrument.WithDescription("Number of log records that were rejected by the next component in the pipeline."),
		instrument.WithUnit(unit.Dimensionless),
	)
	handleError(obsmetrics.ProcessorPrefix+obsmetrics.RefusedLogRecordsKey, err)

	proc.droppedLogRecordsCounter, err = meter.SyncInt64().Counter(
		obsmetrics.ProcessorPrefix+obsmetrics.DroppedLogRecordsKey,
		instrument.WithDescription("Number of log records that were dropped."),
		instrument.WithUnit(unit.Dimensionless),
	)
	handleError(obsmetrics.ProcessorPrefix+obsmetrics.DroppedLogRecordsKey, err)
}

// TracesAccepted reports that the trace data was accepted.
func (por *Processor) TracesAccepted(ctx context.Context, numSpans int) {
	if por.level == configtelemetry.LevelNone {
		return
	}

	if por.useOtelForMetrics {
		por.acceptedSpansCounter.Add(ctx, int64(numSpans), por.otelAttrs...)
	} else {
		// ignore the error for now; should not happen
		_ = stats.RecordWithTags(
			ctx,
			por.mutators,
			obsmetrics.ProcessorAcceptedSpans.M(int64(numSpans)),
			obsmetrics.ProcessorRefusedSpans.M(0),
			obsmetrics.ProcessorDroppedSpans.M(0),
		)
	}
}

// TracesRefused reports that the trace data was refused.
func (por *Processor) TracesRefused(ctx context.Context, numSpans int) {
	if por.level == configtelemetry.LevelNone {
		return
	}

	if por.useOtelForMetrics {
		por.refusedSpansCounter.Add(ctx, int64(numSpans), por.otelAttrs...)
	} else {
		// ignore the error for now; should not happen
		_ = stats.RecordWithTags(
			ctx,
			por.mutators,
			obsmetrics.ProcessorAcceptedSpans.M(0),
			obsmetrics.ProcessorRefusedSpans.M(int64(numSpans)),
			obsmetrics.ProcessorDroppedSpans.M(0),
		)

	}
}

// TracesDropped reports that the trace data was dropped.
func (por *Processor) TracesDropped(ctx context.Context, numSpans int) {
	if por.level == configtelemetry.LevelNone {
		return
	}

	if por.useOtelForMetrics {
		por.droppedSpansCounter.Add(ctx, int64(numSpans), por.otelAttrs...)
	} else {
		// ignore the error for now; should not happen
		_ = stats.RecordWithTags(
			ctx,
			por.mutators,
			obsmetrics.ProcessorAcceptedSpans.M(0),
			obsmetrics.ProcessorRefusedSpans.M(0),
			obsmetrics.ProcessorDroppedSpans.M(int64(numSpans)),
		)
	}
}

// MetricsAccepted reports that the metrics were accepted.
func (por *Processor) MetricsAccepted(ctx context.Context, numPoints int) {
	if por.level == configtelemetry.LevelNone {
		return
	}

	if por.useOtelForMetrics {
		por.acceptedMetricPointsCounter.Add(ctx, int64(numPoints), por.otelAttrs...)
	} else {
		// ignore the error for now; should not happen
		_ = stats.RecordWithTags(
			ctx,
			por.mutators,
			obsmetrics.ProcessorAcceptedMetricPoints.M(int64(numPoints)),
			obsmetrics.ProcessorRefusedMetricPoints.M(0),
			obsmetrics.ProcessorDroppedMetricPoints.M(0),
		)

	}
}

// MetricsRefused reports that the metrics were refused.
func (por *Processor) MetricsRefused(ctx context.Context, numPoints int) {
	if por.level == configtelemetry.LevelNone {
		return
	}

	if por.useOtelForMetrics {
		por.refusedMetricPointsCounter.Add(ctx, int64(numPoints), por.otelAttrs...)
	} else {
		// ignore the error for now; should not happen
		_ = stats.RecordWithTags(
			ctx,
			por.mutators,
			obsmetrics.ProcessorAcceptedMetricPoints.M(0),
			obsmetrics.ProcessorRefusedMetricPoints.M(int64(numPoints)),
			obsmetrics.ProcessorDroppedMetricPoints.M(0),
		)
	}
}

// MetricsDropped reports that the metrics were dropped.
func (por *Processor) MetricsDropped(ctx context.Context, numPoints int) {
	if por.level == configtelemetry.LevelNone {
		return
	}

	if por.useOtelForMetrics {
		por.droppedMetricPointsCounter.Add(ctx, int64(numPoints), por.otelAttrs...)
	} else {
		// ignore the error for now; should not happen
		_ = stats.RecordWithTags(
			ctx,
			por.mutators,
			obsmetrics.ProcessorAcceptedMetricPoints.M(0),
			obsmetrics.ProcessorRefusedMetricPoints.M(0),
			obsmetrics.ProcessorDroppedMetricPoints.M(int64(numPoints)),
		)
	}
}

// LogsAccepted reports that the logs were accepted.
func (por *Processor) LogsAccepted(ctx context.Context, numRecords int) {
	if por.level == configtelemetry.LevelNone {
		return
	}

	if por.useOtelForMetrics {
		por.acceptedLogRecordsCounter.Add(ctx, int64(numRecords), por.otelAttrs...)
	} else {
		// ignore the error for now; should not happen
		_ = stats.RecordWithTags(
			ctx,
			por.mutators,
			obsmetrics.ProcessorAcceptedLogRecords.M(int64(numRecords)),
			obsmetrics.ProcessorRefusedLogRecords.M(0),
			obsmetrics.ProcessorDroppedLogRecords.M(0),
		)
	}
}

// LogsRefused reports that the logs were refused.
func (por *Processor) LogsRefused(ctx context.Context, numRecords int) {
	if por.level == configtelemetry.LevelNone {
		return
	}

	if por.useOtelForMetrics {
		por.refusedLogRecordsCounter.Add(ctx, int64(numRecords), por.otelAttrs...)
	} else {
		// ignore the error for now; should not happen
		_ = stats.RecordWithTags(
			ctx,
			por.mutators,
			obsmetrics.ProcessorAcceptedLogRecords.M(0),
			obsmetrics.ProcessorRefusedLogRecords.M(int64(numRecords)),
			obsmetrics.ProcessorDroppedMetricPoints.M(0),
		)
	}
}

// LogsDropped reports that the logs were dropped.
func (por *Processor) LogsDropped(ctx context.Context, numRecords int) {
	if por.level == configtelemetry.LevelNone {
		return
	}

	if por.useOtelForMetrics {
		por.droppedLogRecordsCounter.Add(ctx, int64(numRecords), por.otelAttrs...)
	} else {
		// ignore the error for now; should not happen
		_ = stats.RecordWithTags(
			ctx,
			por.mutators,
			obsmetrics.ProcessorAcceptedLogRecords.M(0),
			obsmetrics.ProcessorRefusedLogRecords.M(0),
			obsmetrics.ProcessorDroppedLogRecords.M(int64(numRecords)),
		)
	}
}
