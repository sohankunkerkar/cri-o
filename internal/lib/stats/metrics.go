package statsserver

import (
	"github.com/cri-o/cri-o/internal/lib/sandbox"
	"github.com/cri-o/cri-o/internal/oci"
	types "k8s.io/cri-api/pkg/apis/runtime/v1"
)

var baseLabelKeys = []string{"id", "name", "image"}

type sandboxMetrics struct {
	current *types.PodSandboxMetrics
	// next    *types.PodSandboxMetrics
	cMetrics map[string]*containerMetrics
}

type containerMetrics struct {
	current *types.ContainerMetrics
	// next    *types.PodSandboxMetrics
}

func NewSandboxMetrics(sb *sandbox.Sandbox) *sandboxMetrics {
	return &sandboxMetrics{
		current: &types.PodSandboxMetrics{
			PodSandboxId:     sb.ID(),
			Metrics:          []*types.Metric{}, // TODO population function
			ContainerMetrics: []*types.ContainerMetrics{},
		},
		cMetrics: make(map[string]*containerMetrics),
	}
}

func (sm *sandboxMetrics) ResetMetricsForSandbox() {
	sm.current.Metrics = []*types.Metric{} // TODO population function
}

func (sm *sandboxMetrics) AddMetricToSandbox(m *types.Metric) {
	sm.current.Metrics = append(sm.current.Metrics, m)
}

func NewContainerMetrics(ctr *oci.Container) *containerMetrics {
	return &containerMetrics{
		current: &types.ContainerMetrics{
			ContainerId: ctr.ID(),
			Metrics:     []*types.Metric{}, // TODO population function
		},
	}
}

//func (sm *sandboxMetrics) ResetMetricsForSandbox() {
//	sm.current.Metrics = []*types.Metric{} // TODO population function
//}
//
//func (sm *sandboxMetrics) AddMetricToSandbox(m *types.Metric) {
//	sm.current.Metrics = append(sm.current, m)
//}

type metricsServer struct {
	includedMetrics []string
}

// store metricdescriptors statically at startup, populate the list
func (ss *StatsServer) populateMetricDescriptors(includedKeys []string) {
	// TODO: add default container labels
	_ = map[string][]*types.MetricDescriptor{
		"misc": {
			{
				Name:      "container_scrape_error",
				Help:      "1 if there was an error while getting container metrics, 0 otherwise",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_last_seen",
				Help:      "Last time a container was seen by the exporter",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "cadvisor_version_info",
				Help:      "A metric with a constant '1' value labeled by kernel version, OS version, docker version, cadvisor version & cadvisor revision.",
				LabelKeys: []string{"kernelVersion", "osVersion", "dockerVersion", "cadvisorVersion", "cadvisorRevision"},
			}, {
				Name:      "container_start_time_seconds",
				Help:      "Start time of the container since unix epoch in seconds.",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_spec_cpu_period",
				Help:      "CPU period of the container.",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_spec_cpu_quota",
				Help:      "CPU quota of the container.",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_spec_cpu_shares",
				Help:      "CPU share of the container.",
				LabelKeys: baseLabelKeys,
			},
		},
		"cpu": {
			{
				Name:      "container_cpu_user_seconds_total", // stats.CpuStats.CpuUsage.UsageInUsermode (converted from nano)
				Help:      "Cumulative user cpu time consumed in seconds.",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_cpu_system_seconds_total", // stats.CpuStats.CpuUsage.UsageInKernelmode (converted from nano)
				Help:      "Cumulative system cpu time consumed in seconds.",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_cpu_usage_seconds_total", // stats.CpuStats.CpuUsage.TotalUsage (converted from nano)
				Help:      "Cumulative cpu time consumed in seconds.",
				LabelKeys: append(baseLabelKeys, "cpu"),
			}, {
				Name:      "container_cpu_cfs_periods_total", // stats.CpuStats.ThrottlingData.Periods
				Help:      "Number of elapsed enforcement period intervals.",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_cpu_cfs_throttled_periods_total", // stats.CpuStats.ThrottlingData.ThrottledPeriods
				Help:      "Number of throttled period intervals.",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_cpu_cfs_throttled_seconds_total", // stats.CpuStats.ThrottlingData.ThrottledTime (converted from nano)
				Help:      "Total time duration the container has been throttled.",
				LabelKeys: baseLabelKeys,
			},
		},
		"memory": {
			{
				Name:      "container_memory_cache", // stats.MemoryStats.Cache
				Help:      "Number of bytes of page cache memory.",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_memory_rss", // stats.MemoryStats.Usage.Usage ???
				Help:      "Size of RSS in bytes.",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_memory_mapped_file", // ??? TODO FIXME
				Help:      "Size of memory mapped files in bytes.",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_memory_swap", // stats.MemoryStats.SwapUsage.Usage
				Help:      "Container swap usage in bytes.",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_memory_failcnt", // stats.MemoryStats.Usage.Failcnt
				Help:      "Number of memory usage hits limits",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_memory_usage_bytes", // TODO FIXME
				Help:      "Current memory usage in bytes, including all memory regardless of when it was accessed",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_memory_max_usage_bytes", // stats.MemoryStats.Usage.MaxUsage
				Help:      "Maximum memory usage recorded in bytes",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_memory_working_set_bytes", // TODO FIXME
				Help:      "Current working set in bytes.",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_memory_failures_total", // TODO FIXME
				Help:      "Cumulative count of memory allocation failures.",
				LabelKeys: append(baseLabelKeys, "failure_type", "scope"),
			},
		},
		"processes": {
			{
				Name:      "container_processes",
				Help:      "Number of processes running inside the container.",
				LabelKeys: baseLabelKeys,
			},
			{
				Name:      "container_file_descriptors",
				Help:      "Number of open file descriptors for the container.",
				LabelKeys: baseLabelKeys,
			},
			{
				Name:      "container_sockets",
				Help:      "Number of open sockets for the container.",
				LabelKeys: baseLabelKeys,
			},
			{
				Name:      "container_threads_max",
				Help:      "Maximum number of threads allowed inside the container, infinity if value is zero",
				LabelKeys: baseLabelKeys,
			},
			{
				Name:      "container_threads",
				Help:      "Number of threads running inside the container",
				LabelKeys: baseLabelKeys,
			},
			{
				Name:      "container_ulimits_soft",
				Help:      "Soft ulimit values for the container root process. Unlimited if -1, except priority and nice",
				LabelKeys: append(baseLabelKeys, "ulimit"),
			},
		},
		"disk": {
			{
				Name:      "container_fs_inodes_free",
				Help:      "Number of available Inodes",
				LabelKeys: append(baseLabelKeys, "device"),
			}, {
				Name:      "container_fs_inodes_total",
				Help:      "Number of Inodes",
				LabelKeys: append(baseLabelKeys, "device"),
			}, {
				Name:      "container_fs_limit_bytes",
				Help:      "Number of bytes that can be consumed by the container on this filesystem.",
				LabelKeys: append(baseLabelKeys, "device"),
			}, {
				Name:      "container_fs_usage_bytes",
				Help:      "Number of bytes that are consumed by the container on this filesystem.",
				LabelKeys: append(baseLabelKeys, "device"),
			},
		},
		"cpuLoad": {
			{
				Name:      "container_cpu_load_average_10s",
				Help:      "Value of container cpu load average over the last 10 seconds.",
				LabelKeys: baseLabelKeys,
			}, {
				Name:      "container_tasks_state",
				Help:      "Number of tasks in given state",
				LabelKeys: append(baseLabelKeys, "state"),
			},
		},
	}
}

func sandboxBaseLabelValues(sb *sandbox.Sandbox) []string {
	// TODO FIXME: image?
	return []string{sb.ID(), "POD", ""}
}
