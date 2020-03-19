package metric

import (
	"fmt"

	"github.com/kckecheng/goflow_exporter/common"
	"github.com/kckecheng/goflow_exporter/message"
	"github.com/prometheus/client_golang/prometheus"
)

const internal = 10

var sflowMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "sflow_bytes",
		Help: "sFlow traffic",
	},
	[]string{"recevived_at", "sample_rate", "src_ip", "dst_ip", "src_if", "dst_if", "src_port", "dst_port", "l3_proto", "l4_proto"},
)

func init() {
	common.Logger.Info("Register prometheus metric: sflow_bytes")
	prometheus.MustRegister(sflowMetric)
}

// UpdateMetric update metric
func UpdateMetric(r *message.FlowRecord) {
	common.Logger.Debugf("Update metric with sflow record: %+v", r)
	sflowMetric.With(prometheus.Labels{
		"recevived_at": r.TimeRecvd.String(),
		"sample_rate":  fmt.Sprintf("%d", r.SamplingRate),
		"src_ip":       r.SrcIP,
		"dst_ip":       r.DstIP,
		"src_if":       fmt.Sprintf("%d", r.SrcIf),
		"dst_if":       fmt.Sprintf("%d", r.DstIf),
		"src_port":     fmt.Sprintf("%d", r.SrcPort),
		"dst_port":     fmt.Sprintf("%d", r.DstPort),
		"l3_proto":     fmt.Sprintf("%d", r.Etype),
		"l4_proto":     fmt.Sprintf("%d", r.Proto),
	}).Set(float64(r.Bytes))
}

// Reset clear previous metric instance values
func Reset() {
	sflowMetric.Reset()
}
