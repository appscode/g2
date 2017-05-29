package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	serverNamespace = "gearman_server"
)

type ServerData interface {
	Stats() map[string]int
	Clients() int
	Workers() int
	Jobs() int
	RunningJobsByWorker() map[string]int
	RunningJobsByFunction() map[string]int
}

// TODO: Add Some More Complex Matrics As Needed
func NewServerCollector(s ServerData) prometheus.Collector {
	return &collector{
		metrics: []*element{
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(serverNamespace, "", "worker_count"),
					"Count Connected Workers",
					nil, nil,
				),
				collect: func(d *prometheus.Desc, ch chan<- prometheus.Metric) {
					ch <- prometheus.MustNewConstMetric(
						d,
						prometheus.GaugeValue,
						float64(s.Workers()),
					)
				},
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(serverNamespace, "", "job_count"),
					"Count jobs",
					nil, nil,
				),
				collect: func(d *prometheus.Desc, ch chan<- prometheus.Metric) {
					ch <- prometheus.MustNewConstMetric(
						d,
						prometheus.GaugeValue,
						float64(s.Jobs()),
					)
				},
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(serverNamespace, "", "client_count"),
					"Count Connected Clients",
					nil, nil,
				),
				collect: func(d *prometheus.Desc, ch chan<- prometheus.Metric) {
					ch <- prometheus.MustNewConstMetric(
						d,
						prometheus.GaugeValue,
						float64(s.Clients()),
					)
				},
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(serverNamespace, "", "worker_running_job_count"),
					"Running Job By workers",
					[]string{"worker"}, nil,
				),
				collect: func(d *prometheus.Desc, ch chan<- prometheus.Metric) {
					for k, v := range s.RunningJobsByWorker() {
						ch <- prometheus.MustNewConstMetric(
							d,
							prometheus.GaugeValue,
							float64(v),
							k,
						)
					}
				},
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(serverNamespace, "", "function_running_job_count"),
					"Running job count by functions",
					[]string{"function"}, nil,
				),
				collect: func(d *prometheus.Desc, ch chan<- prometheus.Metric) {
					for k, v := range s.RunningJobsByFunction() {
						ch <- prometheus.MustNewConstMetric(
							d,
							prometheus.GaugeValue,
							float64(v),
							k,
						)
					}
				},
			},
			{
				desc: prometheus.NewDesc(
					prometheus.BuildFQName(serverNamespace, "", "stats"),
					"Running job count by functions",
					[]string{"stats"}, nil,
				),
				collect: func(d *prometheus.Desc, ch chan<- prometheus.Metric) {
					for k, v := range s.Stats() {
						ch <- prometheus.MustNewConstMetric(
							d,
							prometheus.GaugeValue,
							float64(v),
							k,
						)
					}
				},
			},
		},
	}
}
