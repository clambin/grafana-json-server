package grafana_json_server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// The Server structure implements a JSON API server compatible with the JSON API Grafana datasource.
type Server struct {
	metricConfigs     map[string]metric
	variables         map[string]VariableFunc
	logger            *slog.Logger
	prometheusMetrics PrometheusQueryMetrics
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
		metricConfigs:     make(map[string]metric),
		variables:         make(map[string]VariableFunc),
		Router:            chi.NewRouter(),
		prometheusMetrics: NewDefaultPrometheusQueryMetrics("", "", "grafana-json-server"),
		logger:            slog.Default(),
	}

	s.Router.Use(chiMiddleware.Heartbeat("/"))

	for _, option := range options {
		option(&s)
	}

	s.Router.Post("/metrics", s.metrics)
	s.Router.Post("/metric-payload-options", s.metricsPayloadOptions)
	s.Router.Post("/variable", s.variable)
	s.Router.Post("/tag-keys", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotImplemented) })
	s.Router.Post("/tag-values", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotImplemented) })
	s.Router.Post("/query", s.query)

	return &s
}

func (s Server) metrics(w http.ResponseWriter, r *http.Request) {
	type metricRequest struct {
		Metric  string `json:"metric"`
		Payload struct {
		} `json:"payload"`
	}

	queryRequest, err := parseRequest[metricRequest](w, r)
	if err != nil {
		s.logger.Error("invalid request", "err", err)
		return
	}

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
	req, err := parseRequest[MetricPayloadOptionsRequest](w, r)
	if err != nil {
		s.logger.Error("invalid request", "err", err)
	}

	dataSource, ok := s.metricConfigs[req.Metric]
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]\n"))
		return
	}

	if dataSource.MetricPayloadOptionFunc == nil {
		w.Header().Set("Content-Type", "plain/text")
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

func (s Server) query(w http.ResponseWriter, r *http.Request) {
	queryRequest, err := parseRequest[QueryRequest](w, r)
	if err != nil {
		s.logger.Error("invalid request", "err", err)
		return
	}

	targetRefIDs := make(map[string][]QueryRequestTarget)
	for _, target := range queryRequest.Targets {
		targetRefIDs[target.RefID] = append(targetRefIDs[target.RefID], target)
	}
	responses := make([]QueryResponse, 0, len(queryRequest.Targets))
	for _, t := range queryRequest.Targets {
		queryRequest.Targets = targetRefIDs[t.RefID]
		resp, err := s.queryTarget(r.Context(), t.Target, queryRequest)
		if err != nil {
			s.logger.Error("query failed", "err", err)
			continue
		}
		responses = append(responses, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(responses); err != nil {
		http.Error(w, "query: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s Server) queryTarget(ctx context.Context, target string, req QueryRequest) (resp QueryResponse, err error) {
	start := time.Now()

	if datasource, ok := s.metricConfigs[target]; ok {
		resp, err = datasource.Handler.Query(ctx, target, req)
	} else {
		err = fmt.Errorf("invalid target: %s", target)
	}
	s.prometheusMetrics.Measure(target, time.Since(start), err)
	return resp, err
}

func (s Server) variable(w http.ResponseWriter, r *http.Request) {
	request, err := parseRequest[VariableRequest](w, r)
	if err != nil {
		s.logger.Error("invalid request", "err", err)
		return
	}

	variableFunc, ok := s.variables[request.Target]
	if !ok {
		s.logger.Error("no variable handler found", "err", err)
		w.Header().Set("Content-Type", "plain/text")
		http.Error(w, "no variable handler found for '"+request.Target+"'", http.StatusBadRequest)
		return
	}

	variables, err := variableFunc(request)
	if err != nil {
		s.logger.Error("variable handler failed", "err", err, "target", request.Target)
		w.Header().Set("Content-Type", "plain/text")
		http.Error(w, "variables: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(variables)
}

func parseRequest[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var request T
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		w.Header().Set("Content-Type", "plain/text")
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
	}
	return request, err
}
