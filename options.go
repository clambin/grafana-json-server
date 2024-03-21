package grafana_json_server

import (
	"log/slog"
	"net/http"
)

// Option configures a Server.
type Option func(*Server)

// WithLogger sets the slog Logger. The default is slog.Default().
func WithLogger(l *slog.Logger) Option {
	return func(s *Server) {
		s.logger = l
	}
}

// WithMetric adds a new metric to the server. See Metric for more configuration options for a metric.
func WithMetric(m Metric, handler Handler, payloadOption MetricPayloadOptionFunc) Option {
	return func(s *Server) {
		s.metricConfigs[m.Value] = metric{
			Metric:                  m,
			MetricPayloadOptionFunc: payloadOption,
			Handler:                 handler,
		}
	}
}

// WithHandler is a convenience function to create a simple metric (i.e. one without any payload options).
func WithHandler(target string, handler Handler) Option {
	return WithMetric(Metric{Value: target}, handler, nil)
}

// WithHTTPHandler adds a http.Handler to its http router.
func WithHTTPHandler(method, path string, handler http.Handler) Option {
	return func(s *Server) {
		s.Router.Method(method, path, handler)
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

// WithVariable adds a new dashboard variable to the server.
func WithVariable(name string, v VariableFunc) Option {
	return func(s *Server) {
		s.variables[name] = v
	}
}
