package grafana_json_server

import (
	"golang.org/x/exp/slog"
	"net/http"
)

// Option configures a Server.
type Option func(*Server)

// WithLogger logs all requests to the server.  Only slog is supported as a logger.
func WithLogger(l *slog.Logger) Option {
	return func(s *Server) {
		s.logger = l
	}
}

// WithDatasource adds a new data source to the server.
//
// Deprecated: use either WithMetric or WithQuery.
func WithDatasource(dataSource DataSource) Option {
	return func(s *Server) {
		s.dataSources[dataSource.Metric.Value] = dataSource
	}
}

// WithMetric adds a new metric to the server. See Metric for more configuration options for a metric.
func WithMetric(m Metric, query QueryFunc, payloadOption MetricPayloadOptionFunc) Option {
	return WithDatasource(DataSource{
		Metric:                  m,
		MetricPayloadOptionFunc: payloadOption,
		Query:                   query,
	})
}

// WithHandlerFunc adds a http.Handler to its http router.
func WithHandlerFunc(method, path string, handler http.HandlerFunc) Option {
	return func(s *Server) {
		s.Router.MethodFunc(method, path, handler)
	}
}

// WithPrometheusQueryMetrics adds Prometheus metrics to the server's Queries. The metrics can then be collected
// by registering the server with a Prometheus registry.
//
// Calling WithPrometheusMetrics creates two Prometheus metrics:
//   - json_query_duration_seconds records the duration of each query
//   - json_query_error_count counts the total number of errors executing a query
//
// If namespace and/or subsystem are not blank, they are prepended to the metric name.
// Application is added as a label "application".
// The query target is added as a label "target".
func WithPrometheusQueryMetrics(namespace, subsystem, application string) Option {
	return func(s *Server) {
		s.prometheusMetrics = createPrometheusMetrics(namespace, subsystem, application)
	}
}
