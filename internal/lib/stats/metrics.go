package statsserver

import types "k8s.io/cri-api/pkg/apis/runtime/v1"

var baseLabelKeys = []string{"id", "name", "image"}

type metricForPod struct {
	current *types.PodSandboxMetrics
	next    *types.PodSandboxMetrics
}

func metricsStructureforPod() {
}

// store metricdescriptors statically at startup, populate the list
func (ss *StatsServer) populateMetricDescriptors(includedKeys []string) {
	// TODO: add default container labels
	descriptorLists := map[string][]*types.MetricDescriptor{
		"misc": {
			{
				Name:   "container_scrape_error",
				Help:   "1 if there was an error while getting container metrics, 0 otherwise",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_last_seen",
				Help:   "Last time a container was seen by the exporter",
				Labels: baseLabelKeys,
			}, {
				Name:   "cadvisor_version_info",
				Help:   "A metric with a constant '1' value labeled by kernel version, OS version, docker version, cadvisor version & cadvisor revision.",
				Labels: []string{"kernelVersion", "osVersion", "dockerVersion", "cadvisorVersion", "cadvisorRevision"},
			}, {
				Name:   "container_start_time_seconds",
				Help:   "Start time of the container since unix epoch in seconds.",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_spec_cpu_period",
				Help:   "CPU period of the container.",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_spec_cpu_quota",
				Help:   "CPU quota of the container.",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_spec_cpu_shares",
				Help:   "CPU share of the container.",
				Labels: baseLabelKeys,
			},
		},
		"network": {
			{
				Name:   "container_network_receive_bytes_total",
				Help:   "Cumulative count of bytes received",
				Labels: append(baseLabelKeys, "interface"),
			}, {
				Name:   "container_network_receive_packets_total",
				Help:   "Cumulative count of packets received",
				Labels: append(baseLabelKeys, "interface"),
			}, {
				Name:   "container_network_receive_packets_total",
				Help:   "Cumulative count of packets received",
				Labels: append(baseLabelKeys, "interface"),
			}, {
				Name:   "container_network_receive_packets_dropped_total",
				Help:   "Cumulative count of packets dropped while receiving",
				Labels: append(baseLabelKeys, "interface"),
			}, {
				Name:   "container_network_receive_errors_total",
				Help:   "Cumulative count of errors encountered while receiving",
				Labels: append(baseLabelKeys, "interface"),
			}, {
				Name:   "container_network_transmit_bytes_total",
				Help:   "Cumulative count of bytes transmitted",
				Labels: append(baseLabelKeys, "interface"),
			}, {
				Name:   "container_network_transmit_packets_total",
				Help:   "Cumulative count of packets transmitted",
				Labels: append(baseLabelKeys, "interface"),
			}, {
				Name:   "container_network_transmit_packets_dropped_total",
				Help:   "Cumulative count of packets dropped while transmitting",
				Labels: append(baseLabelKeys, "interface"),
			}, {
				Name:   "container_network_transmit_errors_total",
				Help:   "Cumulative count of errors encountered while transmitting",
				Labels: append(baseLabelKeys, "interface"),
			},
		},
		"cpu": {
			{
				Name:   "container_cpu_user_seconds_total",
				Help:   "Cumulative user cpu time consumed in seconds.",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_cpu_system_seconds_total",
				Help:   "Cumulative system cpu time consumed in seconds.",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_cpu_usage_seconds_total",
				Help:   "Cumulative cpu time consumed in seconds.",
				Labels: append(baseLabelKeys, "cpu"),
			}, {
				Name:   "container_cpu_cfs_periods_total",
				Help:   "Number of elapsed enforcement period intervals.",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_cpu_cfs_throttled_periods_total",
				Help:   "Number of throttled period intervals.",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_cpu_cfs_throttled_seconds_total",
				Help:   "Total time duration the container has been throttled.",
				Labels: baseLabelKeys,
			},
		},
		"memory": {
			{
				Name:   "container_memory_cache",
				Help:   "Number of bytes of page cache memory.",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_memory_rss",
				Help:   "Size of RSS in bytes.",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_memory_mapped_file",
				Help:   "Size of memory mapped files in bytes.",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_memory_swap",
				Help:   "Container swap usage in bytes.",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_memory_failcnt",
				Help:   "Number of memory usage hits limits",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_memory_usage_bytes",
				Help:   "Current memory usage in bytes, including all memory regardless of when it was accessed",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_memory_max_usage_bytes",
				Help:   "Maximum memory usage recorded in bytes",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_memory_working_set_bytes",
				Help:   "Current working set in bytes.",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_memory_failures_total",
				Help:   "Cumulative count of memory allocation failures.",
				Labels: append(baseLabelKeys, "failure_type", "scope"),
			},
		},
		"processes": {
			{
				Name:   "container_processes",
				Help:   "Number of processes running inside the container.",
				Labels: baseLabelKeys,
			},
			{
				Name:   "container_file_descriptors",
				Help:   "Number of open file descriptors for the container.",
				Labels: baseLabelKeys,
			},
			{
				Name:   "container_sockets",
				Help:   "Number of open sockets for the container.",
				Labels: baseLabelKeys,
			},
			{
				Name:   "container_threads_max",
				Help:   "Maximum number of threads allowed inside the container, infinity if value is zero",
				Labels: baseLabelKeys,
			},
			{
				Name:   "container_threads",
				Help:   "Number of threads running inside the container",
				Labels: baseLabelKeys,
			},
			{
				Name:   "container_ulimits_soft",
				Help:   "Soft ulimit values for the container root process. Unlimited if -1, except priority and nice",
				Labels: append(baseLabelKeys, "ulimit"),
			},
		},
		"disk": {
			{
				Name:   "container_fs_inodes_free",
				Help:   "Number of available Inodes",
				Labels: append(baseLabelKeys, "device"),
			}, {
				Name:   "container_fs_inodes_total",
				Help:   "Number of Inodes",
				Labels: append(baseLabelKeys, "device"),
			}, {
				Name:   "container_fs_limit_bytes",
				Help:   "Number of bytes that can be consumed by the container on this filesystem.",
				Labels: append(baseLabelKeys, "device"),
			}, {
				Name:   "container_fs_usage_bytes",
				Help:   "Number of bytes that are consumed by the container on this filesystem.",
				Labels: append(baseLabelKeys, "device"),
			},
		},
		"cpuLoad": {
			{
				Name:   "container_cpu_load_average_10s",
				Help:   "Value of container cpu load average over the last 10 seconds.",
				Labels: baseLabelKeys,
			}, {
				Name:   "container_tasks_state",
				Help:   "Number of tasks in given state",
				Labels: append(baseLabelKeys, "state"),
			},
		},
	}
}
