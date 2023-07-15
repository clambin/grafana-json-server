package grafana_json_server

import "golang.org/x/exp/slog"

// Option configures a Server.
type Option func(*Server)

// WithLogger logs all requests to the server.  Only slog is supported as a logger.
func WithLogger(l *slog.Logger) Option {
	return func(s *Server) {
		s.logger = l
	}
}

// WithMetric adds a new metric to the server. See Metric for more configuration options for a metric.
func WithMetric(m Metric, query QueryFunc, payloadOption MetricPayloadOptionFunc) Option {
	return func(s *Server) {
		s.handlers[m.Value] = handler{
			metric:              m,
			metricPayloadOption: payloadOption,
			queryHandler:        query,
		}
	}
}

// WithVariable adds a new dashboard variable to the server.
func WithVariable(name string, v VariableFunc) Option {
	return func(s *Server) {
		s.variables[name] = v
	}
}
