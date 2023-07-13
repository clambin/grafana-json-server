package grafana_json_server

import "golang.org/x/exp/slog"

type Option func(*Server)

func WithLogger(l *slog.Logger) Option {
	return func(s *Server) {
		s.logger = l
	}
}

func WithMetric(m Metric, handler QueryHandlerFunc, payloadOption MetricPayloadOptionFunc) Option {
	return func(s *Server) {
		s.handlers[m.Value] = Handler{
			Metric:              m,
			MetricPayloadOption: payloadOption,
			QueryHandler:        handler,
		}
	}
}

func WithVariable(name string, v []Variable) Option {
	return func(s *Server) {
		s.variables[name] = v
	}
}
