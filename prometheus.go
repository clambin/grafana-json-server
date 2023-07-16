package grafana_json_server

import "github.com/prometheus/client_golang/prometheus"

var _ prometheus.Collector = &prometheusMetrics{}

type prometheusMetrics struct {
	duration *prometheus.SummaryVec
	errors   *prometheus.CounterVec
}

func createPrometheusMetrics(namespace, subsystem, application string) *prometheusMetrics {
	return &prometheusMetrics{
		duration: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "json_query_duration_seconds",
			Help:        "Grafana JSON Data server duration of query requests in seconds",
			ConstLabels: map[string]string{"application": application},
		}, []string{"target"}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "json_query_error_count",
			Help:        "Grafana JSON Data server count of failed requests",
			ConstLabels: prometheus.Labels{"application": application},
		}, []string{"target"}),
	}
}

func (p prometheusMetrics) Describe(descs chan<- *prometheus.Desc) {
	p.duration.Describe(descs)
	p.errors.Describe(descs)
}

func (p prometheusMetrics) Collect(metrics chan<- prometheus.Metric) {
	p.duration.Collect(metrics)
	p.errors.Collect(metrics)
}
