package statsserver

import (
	"time"

	"github.com/cri-o/cri-o/internal/lib/sandbox"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	types "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type cpuMetric struct {
	desc         *types.MetricDescriptor
	valueFunc    func(*cgroups.CpuStats) uint64
	cpuValueFunc func(*cgroups.CpuStats)
}

var (
	cpuKey     string      = "cpu"
	cpuMetrics []cpuMetric = []cpuMetric{
		{
			desc: &types.MetricDescriptor{
				Name:      "container_cpu_user_seconds_total", // stats.CpuStats.CpuUsage.UsageInUsermode (converted from nano)
				Help:      "Cumulative user cpu time consumed in seconds.",
				LabelKeys: baseLabelKeys,
			},
			valueFunc: func(cpu *cgroups.CpuStats) uint64 {
				return cpu.CpuUsage.UsageInUsermode / uint64(time.Second)
			},
		}, {
			desc: &types.MetricDescriptor{
				Name:      "container_cpu_system_seconds_total", // stats.CpuStats.CpuUsage.UsageInKernelmode (converted from nano)
				Help:      "Cumulative system cpu time consumed in seconds.",
				LabelKeys: baseLabelKeys,
			},
			valueFunc: func(cpu *cgroups.CpuStats) uint64 {
				return cpu.CpuUsage.UsageInKernelmode / uint64(time.Second)
			},
		}, {
			desc: &types.MetricDescriptor{
				Name:      "container_cpu_usage_seconds_total", // stats.CpuStats.CpuUsage.TotalUsage (converted from nano)
				Help:      "Cumulative cpu time consumed in seconds.",
				LabelKeys: append(baseLabelKeys, "cpu"), // TODO FIXME: need to loop through, basically need to adopt cadvisor's metricsValues structure
			},
			valueFunc: func(cpu *cgroups.CpuStats) uint64 {
				var totalUsage uint64

				// Check if per-core CPU usage is available and calculate the sum of all per-core usages
				for _, usage := range cpu.CpuUsage.PercpuUsage {
					totalUsage += usage
				}

				// If per-core usage is not available, use the total usage directly
				if totalUsage == 0 {
					totalUsage = cpu.CpuUsage.TotalUsage
				}

				// Convert the total usage from nanoseconds to seconds
				return totalUsage / uint64(time.Second)
			},
		}, {
			desc: &types.MetricDescriptor{
				Name:      "container_cpu_cfs_periods_total", // stats.CpuStats.ThrottlingData.Periods
				Help:      "Number of elapsed enforcement period intervals.",
				LabelKeys: baseLabelKeys,
			},
			valueFunc: func(cpu *cgroups.CpuStats) uint64 {
				return cpu.ThrottlingData.Periods
			},
		}, {
			desc: &types.MetricDescriptor{
				Name:      "container_cpu_cfs_throttled_periods_total", // stats.CpuStats.ThrottlingData.ThrottledPeriods
				Help:      "Number of throttled period intervals.",
				LabelKeys: baseLabelKeys,
			},
			valueFunc: func(cpu *cgroups.CpuStats) uint64 {
				return cpu.ThrottlingData.ThrottledPeriods
			},
		}, {
			desc: &types.MetricDescriptor{
				Name:      "container_cpu_cfs_throttled_seconds_total", // stats.CpuStats.ThrottlingData.ThrottledTime (converted from nano)
				Help:      "Total time duration the container has been throttled.",
				LabelKeys: baseLabelKeys,
			},
			valueFunc: func(cpu *cgroups.CpuStats) uint64 {
				return cpu.ThrottlingData.ThrottledTime / uint64(time.Second)
			},
		},
	}
)

func generateSandboxCpuMetrics(sb *sandbox.Sandbox, cpu *cgroups.CpuStats, timestamp int64) []*types.Metric {
	values := append(sandboxBaseLabelValues(sb), cpuKey)
	metrics := make([]*types.Metric, 0, len(cpuMetrics))
	for _, m := range cpuMetrics {
		metrics = append(metrics, &types.Metric{
			Name:        m.desc.Name,
			Timestamp:   timestamp,
			MetricType:  types.MetricType_COUNTER,
			Value:       &types.UInt64Value{Value: m.valueFunc(cpu)},
			LabelValues: values,
		})
	}
	return metrics
}
