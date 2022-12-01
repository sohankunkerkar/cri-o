package statsserver

import (
	"github.com/cri-o/cri-o/internal/lib/sandbox"
	"github.com/vishvananda/netlink"
	types "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// networkMetric is  a structure for simplifying housekeeping for metrics and desciptors
// by keeping the unique values independent and close to each other.
// valueFunc takes a netlink.LinkAttr and returns the pertinent value for that metric.
// Its associated desciptor can be aggregated when returning MetricDescriptors.
type networkMetric struct {
	desc      *types.MetricDescriptor
	valueFunc func(*netlink.LinkAttrs) uint64
}

var (
	networkKey     string          = "network"
	networkMetrics []networkMetric = []networkMetric{
		{
			desc: &types.MetricDescriptor{
				Name:      "container_network_receive_bytes_total",
				Help:      "Cumulative count of bytes received",
				LabelKeys: append(baseLabelKeys, "interface"),
			},
			valueFunc: func(attr *netlink.LinkAttrs) uint64 {
				return attr.Statistics.RxBytes
			},
		}, {
			desc: &types.MetricDescriptor{
				Name:      "container_network_receive_packets_total",
				Help:      "Cumulative count of packets received",
				LabelKeys: append(baseLabelKeys, "interface"),
			},
			valueFunc: func(attr *netlink.LinkAttrs) uint64 {
				return attr.Statistics.RxPackets
			},
		}, {
			desc: &types.MetricDescriptor{
				Name:      "container_network_receive_packets_dropped_total",
				Help:      "Cumulative count of packets dropped while receiving",
				LabelKeys: append(baseLabelKeys, "interface"),
			},
			valueFunc: func(attr *netlink.LinkAttrs) uint64 {
				return attr.Statistics.RxDropped
			},
		}, {
			desc: &types.MetricDescriptor{
				Name:      "container_network_receive_errors_total",
				Help:      "Cumulative count of errors encountered while receiving",
				LabelKeys: append(baseLabelKeys, "interface"),
			},
			valueFunc: func(attr *netlink.LinkAttrs) uint64 {
				return attr.Statistics.RxErrors
			},
		}, {
			desc: &types.MetricDescriptor{
				Name:      "container_network_transmit_bytes_total",
				Help:      "Cumulative count of bytes transmitted",
				LabelKeys: append(baseLabelKeys, "interface"),
			},
			valueFunc: func(attr *netlink.LinkAttrs) uint64 {
				return attr.Statistics.TxBytes
			},
		}, {
			desc: &types.MetricDescriptor{
				Name:      "container_network_transmit_packets_total",
				Help:      "Cumulative count of packets transmitted",
				LabelKeys: append(baseLabelKeys, "interface"),
			},
			valueFunc: func(attr *netlink.LinkAttrs) uint64 {
				return attr.Statistics.TxPackets
			},
		}, {
			desc: &types.MetricDescriptor{
				Name:      "container_network_transmit_packets_dropped_total",
				Help:      "Cumulative count of packets dropped while transmitting",
				LabelKeys: append(baseLabelKeys, "interface"),
			},
			valueFunc: func(attr *netlink.LinkAttrs) uint64 {
				return attr.Statistics.TxDropped
			},
		}, {
			desc: &types.MetricDescriptor{
				Name:      "container_network_transmit_errors_total",
				Help:      "Cumulative count of errors encountered while transmitting",
				LabelKeys: append(baseLabelKeys, "interface"),
			},
			valueFunc: func(attr *netlink.LinkAttrs) uint64 {
				return attr.Statistics.TxErrors
			},
		},
	}
)

func generateNetworkMetrics(sb *sandbox.Sandbox, attr *netlink.LinkAttrs, timestamp int64) []*types.Metric {
	values := append(sandboxBaseLabelValues(sb), attr.Name)
	metrics := make([]*types.Metric, 0, len(networkMetrics))
	for _, m := range networkMetrics {
		metrics = append(metrics, &types.Metric{
			Name:        m.desc.Name,
			Timestamp:   timestamp,
			MetricType:  types.MetricType_COUNTER,
			Value:       &types.UInt64Value{Value: m.valueFunc(attr)},
			LabelValues: values,
		})
	}
	return metrics
}
