package grafana_json_server

import (
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

type PrometheusQueryMetrics interface {
	Measure(target string, duration time.Duration, err error)
	prometheus.Collector
}

var _ PrometheusQueryMetrics = &defaultPrometheusQueryMetrics{}

type defaultPrometheusQueryMetrics struct {
	duration *prometheus.SummaryVec
	errors   *prometheus.CounterVec
}

// NewDefaultPrometheusQueryMetrics returns the default PrometheusQueryMetrics implementation. It created two Prometheus metrics:
//   - json_query_duration_seconds records the duration of each query
//   - json_query_error_count counts the total number of errors executing a query
//
// If namespace and/or subsystem are not blank, they are prepended to the metric name.
// Application is added as a label "application".
// The query target is added as a label "target".
func NewDefaultPrometheusQueryMetrics(namespace, subsystem, application string) PrometheusQueryMetrics {
	return defaultPrometheusQueryMetrics{
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

func (m defaultPrometheusQueryMetrics) Measure(target string, duration time.Duration, err error) {
	if err != nil {
		m.errors.WithLabelValues(target).Add(1)
	}
	m.duration.WithLabelValues(target).Observe(duration.Seconds())
}

func (m defaultPrometheusQueryMetrics) Describe(descs chan<- *prometheus.Desc) {
	m.duration.Describe(descs)
	m.errors.Describe(descs)
}

func (m defaultPrometheusQueryMetrics) Collect(metrics chan<- prometheus.Metric) {
	m.duration.Collect(metrics)
	m.errors.Collect(metrics)
}
