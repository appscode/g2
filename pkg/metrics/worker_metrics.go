package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	workerNamespace = "gearman_worker"
)

type WorkerData interface {
	Running() (string, int)
	Agents() int
}

func NewWorkerCollector(w WorkerData) prometheus.Collector {
	return &collector{
		metrics: []*element{
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(workerNamespace, "", "worker_running"),
					"Worker Running",
					[]string{"worker"}, nil,
				),
				collect: func(d *prometheus.Desc, ch chan<- prometheus.Metric) {
					id, running := w.Running()
					ch <- prometheus.MustNewConstMetric(
						d,
						prometheus.GaugeValue,
						float64(running),
						id,
					)
				},
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(workerNamespace, "", "agent_count"),
					"Count Connected Agents",
					nil, nil,
				),
				collect: func(d *prometheus.Desc, ch chan<- prometheus.Metric) {
					ch <- prometheus.MustNewConstMetric(
						d,
						prometheus.GaugeValue,
						float64(w.Agents()),
					)
				},
			},
		},
	}
}
