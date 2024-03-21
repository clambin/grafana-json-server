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
		s.Handler.(*http.ServeMux).Handle(method+" "+path, handler)
	}
}

// WithPrometheusQueryMetrics adds Prometheus metrics to the server's Queries. The caller must register the metrics
// with the Prometheus registry.
//
// See [NewDefaultPrometheusQueryMetrics] for the default implementation of Prometheus metrics.
func WithPrometheusQueryMetrics(metrics PrometheusQueryMetrics) Option {
	return func(s *Server) {
		s.prometheusMetrics = metrics
	}
}

// WithVariable adds a new dashboard variable to the server.
func WithVariable(name string, v VariableFunc) Option {
	return func(s *Server) {
		s.variables[name] = v
	}
}
