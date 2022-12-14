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

package proctelemetry // import "go.opentelemetry.io/collector/service/internal/proctelemetry"

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	ocmetric "go.opencensus.io/metric"
	"go.opencensus.io/stats"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/unit"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
)

const (
	scopeName = "go.opentelemetry.io/collector/service/internal/proctelemetry"
	processNameKey = "process_name"

)

// processMetrics is a struct that contains views related to process metrics (cpu, mem, etc)
type processMetrics struct {
	startTimeUnixNano int64
	ballastSizeBytes  uint64
	proc              *process.Process

	processUptime *ocmetric.Float64DerivedCumulative
	allocMem      *ocmetric.Int64DerivedGauge
	totalAllocMem *ocmetric.Int64DerivedCumulative
	sysMem        *ocmetric.Int64DerivedGauge
	cpuSeconds    *ocmetric.Float64DerivedCumulative
	rssMemory     *ocmetric.Int64DerivedGauge

	// otel metrics
	otelProcessUptime asyncfloat64.Counter
	otelAllocMem      asyncint64.Counter
	otelTotalAllocMem asyncint64.Counter
	otelSysMem        asyncint64.Counter
	otelCpuSeconds    asyncfloat64.Counter
	otelRssMemory     asyncint64.Counter

	meter                otelmetric.Meter
	logger               *zap.Logger
	useOtelForMetrics    bool
	otelAttrs            []attribute.KeyValue

	// mu protects everything bellow.
	mu         sync.Mutex
	lastMsRead time.Time
	ms         *runtime.MemStats
}

func NewProcessMetrics(logger *zap.Logger, registry *featuregate.Registry, meter otelmetric.MeterProvider, ballastSizeBytes uint64) *processMetrics {
	pm := &processMetrics{
		startTimeUnixNano: time.Now().UnixNano(),
		ballastSizeBytes:  ballastSizeBytes,
		ms:                &runtime.MemStats{},
	    useOtelForMetrics: registry.IsEnabled(obsreportconfig.UseOtelForInternalMetricsfeatureGateID),
		logger:            logger,
	}

	if pm.useOtelForMetrics {
		pm.meter = meter.Meter(scopeName)
	}

	return pm
}

// RegisterProcessMetrics creates a new set of processMetrics (mem, cpu) that can be used to measure
// basic information about this process.
func (pm *processMetrics) RegisterProcessMetrics(ctx context.Context, ocRegistry *ocmetric.Registry) error {
	var err error
	pm.proc, err = process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return err
	}

	if pm.useOtelForMetrics {
		err = pm.createOtelMetrics()
		if err != nil {
			return err
		}

		err = pm.recordWithOtel(context.Background())
		if err != nil {
			return err
		}
	} else { // record with OC
		err = pm.createOCMetrics(ocRegistry)
		if err != nil {
			return err
		}

		err = pm.recordWithOC()
		if err != nil {
			return err
		}
	}

	return nil
}

func (pm *processMetrics) createOtelMetrics() error {
	var errors, err error

	pm.otelProcessUptime, err = pm.meter.AsyncFloat64().Counter(
		"process/uptime",
		instrument.WithDescription("Uptime of the process"),
		instrument.WithUnit(unit.Milliseconds))
	errors = multierr.Append(errors, err)

	pm.otelAllocMem, err = pm.meter.AsyncInt64().Counter(
		"process/runtime/heap_alloc_bytes",
		instrument.WithDescription("Bytes of allocated heap objects (see 'go doc runtime.MemStats.HeapAlloc')"),
		instrument.WithUnit(unit.Bytes))
	errors = multierr.Append(errors, err)

	pm.otelTotalAllocMem, err = pm.meter.AsyncInt64().Counter(
		"process/runtime/total_alloc_bytes",
		instrument.WithDescription("Cumulative bytes allocated for heap objects (see 'go doc runtime.MemStats.TotalAlloc')"),
		instrument.WithUnit(unit.Bytes))
	errors = multierr.Append(errors, err)

	pm.otelSysMem, err = pm.meter.AsyncInt64().Counter(
		"process/runtime/total_sys_memory_bytes",
		instrument.WithDescription("Total bytes of memory obtained from the OS (see 'go doc runtime.MemStats.Sys')"),
		instrument.WithUnit(unit.Bytes))
	errors = multierr.Append(errors, err)

	pm.otelCpuSeconds, err = pm.meter.AsyncFloat64().Counter(
		"process/cpu_seconds",
		instrument.WithDescription("Total CPU user and system time in seconds"),
		instrument.WithUnit(unit.Milliseconds))
	errors = multierr.Append(errors, err)

	pm.otelRssMemory, err = pm.meter.AsyncInt64().Counter(
		"process/memory/rss",
		instrument.WithDescription("Total physical memory (resident set size)"),
		instrument.WithUnit(unit.Bytes))
	errors = multierr.Append(errors, err)

	return errors
}

func (pm *processMetrics) recordWithOtel(ctx context.Context) error {
	var errors, err error
	err = pm.meter.RegisterCallback([]instrument.Asynchronous{pm.otelProcessUptime}, func(ctx context.Context) {
		processTimeMs := 1000 * pm.updateProcessUptime()
		pm.otelProcessUptime.Observe(ctx, processTimeMs, pm.otelAttrs...)
	})
	errors = multierr.Append(errors, err)

	err = pm.meter.RegisterCallback([]instrument.Asynchronous{pm.otelAllocMem}, func(ctx context.Context) {
		pm.otelAllocMem.Observe(ctx, pm.updateAllocMem(), pm.otelAttrs...)
	})
	errors = multierr.Append(errors, err)

	err = pm.meter.RegisterCallback([]instrument.Asynchronous{pm.otelTotalAllocMem}, func(ctx context.Context) {
		pm.otelTotalAllocMem.Observe(ctx, pm.updateTotalAllocMem(), pm.otelAttrs...)
	})
	errors = multierr.Append(errors, err)

	err = pm.meter.RegisterCallback([]instrument.Asynchronous{pm.otelSysMem}, func(ctx context.Context) {
		pm.otelSysMem.Observe(ctx, pm.updateSysMem(), pm.otelAttrs...)
	})
	errors = multierr.Append(errors, err)

	err = pm.meter.RegisterCallback([]instrument.Asynchronous{pm.otelCpuSeconds}, func(ctx context.Context) {
		cpuMs := 1000 * pm.updateCPUSeconds()
		pm.otelCpuSeconds.Observe(ctx, cpuMs, pm.otelAttrs...)
	})
	errors = multierr.Append(errors, err)

	err = pm.meter.RegisterCallback([]instrument.Asynchronous{pm.otelRssMemory}, func(ctx context.Context) {
		pm.otelRssMemory.Observe(ctx, pm.updateRSSMemory(), pm.otelAttrs...)
	})
	errors = multierr.Append(errors, err)

	return errors
}

func (pm *processMetrics) createOCMetrics(registry *ocmetric.Registry) error {
	var errors, err error

	pm.processUptime, err = registry.AddFloat64DerivedCumulative(
		"process/uptime",
		ocmetric.WithDescription("Uptime of the process"),
		ocmetric.WithUnit(stats.UnitSeconds))
	errors = multierr.Append(errors, err)

	pm.allocMem, err = registry.AddInt64DerivedGauge(
		"process/runtime/heap_alloc_bytes",
		ocmetric.WithDescription("Bytes of allocated heap objects (see 'go doc runtime.MemStats.HeapAlloc')"),
		ocmetric.WithUnit(stats.UnitBytes))
	errors = multierr.Append(errors, err)

	pm.totalAllocMem, err = registry.AddInt64DerivedCumulative(
		"process/runtime/total_alloc_bytes",
		ocmetric.WithDescription("Cumulative bytes allocated for heap objects (see 'go doc runtime.MemStats.TotalAlloc')"),
		ocmetric.WithUnit(stats.UnitBytes))
	errors = multierr.Append(errors, err)

	pm.sysMem, err = registry.AddInt64DerivedGauge(
		"process/runtime/total_sys_memory_bytes",
		ocmetric.WithDescription("Total bytes of memory obtained from the OS (see 'go doc runtime.MemStats.Sys')"),
		ocmetric.WithUnit(stats.UnitBytes))
	errors = multierr.Append(errors, err)

	pm.cpuSeconds, err = registry.AddFloat64DerivedCumulative(
		"process/cpu_seconds",
		ocmetric.WithDescription("Total CPU user and system time in seconds"),
		ocmetric.WithUnit(stats.UnitSeconds))
	errors = multierr.Append(errors, err)

	pm.rssMemory, err = registry.AddInt64DerivedGauge(
		"process/memory/rss",
		ocmetric.WithDescription("Total physical memory (resident set size)"),
		ocmetric.WithUnit(stats.UnitBytes))
	errors = multierr.Append(errors, err)

	return errors
}

func (pm *processMetrics) recordWithOC() error {
	var errors, err error
	err = pm.processUptime.UpsertEntry(pm.updateProcessUptime)
	errors = multierr.Append(errors, err)

	err = pm.allocMem.UpsertEntry(pm.updateAllocMem)
	errors = multierr.Append(errors, err)

	err = pm.totalAllocMem.UpsertEntry(pm.updateTotalAllocMem)
	errors = multierr.Append(errors, err)

	err = pm.sysMem.UpsertEntry(pm.updateSysMem)
	errors = multierr.Append(errors, err)

	err = pm.cpuSeconds.UpsertEntry(pm.updateCPUSeconds)
	errors = multierr.Append(errors, err)

	err = pm.rssMemory.UpsertEntry(pm.updateRSSMemory)
	errors = multierr.Append(errors, err)

	return errors
}

func (pm *processMetrics) updateProcessUptime() float64 {
	now := time.Now().UnixNano()
	return float64(now-pm.startTimeUnixNano) / 1e9
}

func (pm *processMetrics) updateAllocMem() int64 {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.readMemStatsIfNeeded()
	return int64(pm.ms.Alloc)
}

func (pm *processMetrics) updateTotalAllocMem() int64 {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.readMemStatsIfNeeded()
	return int64(pm.ms.TotalAlloc)
}

func (pm *processMetrics) updateSysMem() int64 {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.readMemStatsIfNeeded()
	return int64(pm.ms.Sys)
}

func (pm *processMetrics) updateCPUSeconds() float64 {
	times, err := pm.proc.Times()
	if err != nil {
		return 0
	}

	return times.User + times.System + times.Idle + times.Nice +
		times.Iowait + times.Irq + times.Softirq + times.Steal
}

func (pm *processMetrics) updateRSSMemory() int64 {
	mem, err := pm.proc.MemoryInfo()
	if err != nil {
		return 0
	}
	return int64(mem.RSS)
}

func (pm *processMetrics) readMemStatsIfNeeded() {
	now := time.Now()
	// If last time we read was less than one second ago just reuse the values
	if now.Sub(pm.lastMsRead) < time.Second {
		return
	}
	pm.lastMsRead = now
	runtime.ReadMemStats(pm.ms)
	if pm.ballastSizeBytes > 0 {
		pm.ms.Alloc -= pm.ballastSizeBytes
		pm.ms.HeapAlloc -= pm.ballastSizeBytes
		pm.ms.HeapSys -= pm.ballastSizeBytes
		pm.ms.HeapInuse -= pm.ballastSizeBytes
	}
}
