package grafana_json_server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/clambin/go-common/httpserver/middleware"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"net/http"
)

// The Server structure implements a JSON API server compatible with the JSON API Grafana datasource.
type Server struct {
	metricConfigs       map[string]metric
	variables           map[string]VariableFunc
	logger              *slog.Logger
	requestLogLevel     slog.Level
	requestLogFormatter middleware.RequestLogFormatter
	prometheusMetrics   *prometheusMetrics
	chi.Router
}

type metric struct {
	Metric
	MetricPayloadOptionFunc
	Handler
}

// NewServer returns a new JSON API server, configured as per the provided Option items.
func NewServer(options ...Option) *Server {
	s := Server{
		metricConfigs:       make(map[string]metric),
		variables:           make(map[string]VariableFunc),
		Router:              chi.NewRouter(),
		logger:              slog.Default(),
		requestLogLevel:     slog.LevelDebug,
		requestLogFormatter: middleware.DefaultRequestLogFormatter,
	}

	s.Router.Use(chiMiddleware.Heartbeat("/"))

	for _, option := range options {
		option(&s)
	}

	s.Router.Group(func(r chi.Router) {
		r.Use(middleware.RequestLogger(s.logger, s.requestLogLevel, s.requestLogFormatter))
		r.Post("/metrics", s.metrics)
		r.Post("/metric-payload-options", s.metricsPayloadOptions)
		r.Post("/variable", s.variable)
		r.Post("/tag-keys", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotImplemented) })
		r.Post("/tag-values", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotImplemented) })
		r.Post("/query", s.query)
	})

	return &s
}

func (s Server) metrics(w http.ResponseWriter, r *http.Request) {
	type metricRequest struct {
		Metric  string `json:"metric"`
		Payload struct {
		} `json:"payload"`
	}

	var queryRequest metricRequest
	err := json.NewDecoder(r.Body).Decode(&queryRequest)
	if err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: we could cache metrics so we only need to build it once
	metrics := make([]Metric, 0, len(s.metricConfigs))
	for _, config := range s.metricConfigs {
		if queryRequest.Metric == "" || queryRequest.Metric == config.Metric.Value {
			metrics = append(metrics, config.Metric)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metrics)
}

func (s Server) metricsPayloadOptions(w http.ResponseWriter, r *http.Request) {
	var req MetricPayloadOptionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	dataSource, ok := s.metricConfigs[req.Metric]
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]\n"))
		return
	}

	if dataSource.MetricPayloadOptionFunc == nil {
		http.Error(w, "invalid request: target does not have a metric payload option function", http.StatusInternalServerError)
		return
	}

	options, err := dataSource.MetricPayloadOptionFunc(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(options)
}

func (s Server) query(w http.ResponseWriter, req *http.Request) {
	var queryRequest QueryRequest
	if err := json.NewDecoder(req.Body).Decode(&queryRequest); err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	responses := make([]QueryResponse, 0, len(queryRequest.Targets))
	for _, t := range queryRequest.Targets {
		resp, err := s.queryTarget(req.Context(), t.Target, queryRequest)
		if err != nil {
			s.logger.Error("query failed", "err", err)
			continue
		}
		responses = append(responses, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(responses)
	if err != nil {
		http.Error(w, "query: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s Server) queryTarget(ctx context.Context, target string, req QueryRequest) (QueryResponse, error) {
	datasource, ok := s.metricConfigs[target]
	if !ok {
		if s.prometheusMetrics != nil {
			s.prometheusMetrics.errors.WithLabelValues(target).Add(1)
		}
		return nil, fmt.Errorf("invalid query target: %s", target)
	}

	var timer *prometheus.Timer
	if s.prometheusMetrics != nil {
		timer = prometheus.NewTimer(s.prometheusMetrics.duration.WithLabelValues(target))
	}

	resp, err := datasource.Handler.Query(ctx, target, req)

	if timer != nil {
		timer.ObserveDuration()
	}

	if s.prometheusMetrics != nil && err != nil {
		s.prometheusMetrics.errors.WithLabelValues(target).Add(1)
	}
	return resp, err
}

func (s Server) variable(w http.ResponseWriter, r *http.Request) {
	request, err := parseVariableRequest(r.Body)
	if err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	variableFunc, ok := s.variables[request.Target]
	if !ok {
		http.Error(w, "no variable handler found", http.StatusBadRequest)
		return
	}

	variables, err := variableFunc(request)
	if err != nil {
		http.Error(w, "variables: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(variables)
}

// Describe implements the prometheus.Collector interface. It describes the prometheus metrics, if present.
func (s Server) Describe(ch chan<- *prometheus.Desc) {
	if s.prometheusMetrics != nil {
		s.prometheusMetrics.Describe(ch)
	}
}

// Collect implements the prometheus.Collector interface. It describes the prometheus metrics, if present.
func (s Server) Collect(ch chan<- prometheus.Metric) {
	if s.prometheusMetrics != nil {
		s.prometheusMetrics.Collect(ch)
	}
}
