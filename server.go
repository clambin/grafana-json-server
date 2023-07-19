package grafana_json_server

import (
	"encoding/json"
	"github.com/clambin/go-common/httpserver/middleware"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/slog"
	"net/http"
)

// The Server structure implements a JSON API server compatible with the JSON API Grafana datasource.
type Server struct {
	dataSources       map[string]DataSource
	variables         map[string]VariableFunc
	logger            *slog.Logger
	prometheusMetrics *prometheusMetrics
	chi.Router
}

// NewServer returns a new JSON API server, configured as per the provided Option items.
func NewServer(options ...Option) *Server {
	s := &Server{
		dataSources: make(map[string]DataSource),
		variables:   make(map[string]VariableFunc),
		logger:      slog.Default(),
		Router:      chi.NewRouter(),
	}

	s.Router.Use(chiMiddleware.Heartbeat("/"))

	for _, option := range options {
		option(s)
	}

	s.Router.Group(func(r chi.Router) {
		r.Use(middleware.Logger(s.logger))
		r.Post("/metrics", s.metrics)
		r.Post("/metric-payload-options", s.metricsPayloadOptions)
		r.Post("/variable", s.variable)
		r.Post("/tag-keys", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotImplemented) })
		r.Post("/tag-values", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotImplemented) })
		r.Post("/query", s.query)
	})

	return s
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
	metrics := make([]Metric, 0)
	for _, dataSource := range s.dataSources {
		if queryRequest.Metric == "" || queryRequest.Metric == dataSource.Metric.Value {
			metrics = append(metrics, dataSource.Metric)
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

	dataSource, ok := s.dataSources[req.Metric]
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

	responses := make([]QueryResponse, 0)
	for _, t := range queryRequest.Targets {
		dataSource, ok := s.dataSources[t.Target]
		if !ok {
			s.logger.Warn("invalid query target", "target", t)
			if s.prometheusMetrics != nil {
				s.prometheusMetrics.errors.WithLabelValues(t.Target).Add(1)
			}
			continue
		}

		var timer *prometheus.Timer
		if s.prometheusMetrics != nil {
			timer = prometheus.NewTimer(s.prometheusMetrics.duration.WithLabelValues(t.Target))
		}

		resp, err := dataSource.Query.Query(req.Context(), t.Target, queryRequest)

		if timer != nil {
			timer.ObserveDuration()
		}

		if err != nil {
			s.logger.Error("query failed", "err", err)
			if s.prometheusMetrics != nil {
				s.prometheusMetrics.errors.WithLabelValues(t.Target).Add(1)
			}
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

func (s Server) variable(w http.ResponseWriter, r *http.Request) {
	var request VariableRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	variables := make([]Variable, 0)
	variableFunc, ok := s.variables[string(request.Target)]
	if ok && variableFunc != nil {
		var err error
		if variables, err = variableFunc(request); err != nil {
			http.Error(w, "variables: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(variables)
}

func (s Server) Describe(descs chan<- *prometheus.Desc) {
	if s.prometheusMetrics != nil {
		s.prometheusMetrics.Describe(descs)
	}
}

func (s Server) Collect(metrics chan<- prometheus.Metric) {
	if s.prometheusMetrics != nil {
		s.prometheusMetrics.Collect(metrics)
	}
}

/*
func requestLogger(logger *slog.Logger) func(next http.DataSource) http.DataSource {
	return func(next http.DataSource) http.DataSource {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var reqBody bytes.Buffer
			r2 := io.TeeReader(r.Body, &reqBody)
			body, _ := io.ReadAll(r2)
			logger.Debug("rcvd", "path", r.URL.Path, "body", string(body))
			r.Body = io.NopCloser(&reqBody)

			lrw := loggingResponseWriter{ResponseWriter: w}
			next.ServeHTTP(&lrw, r)

			logger.Debug("sent", "path", r.URL.Path, "statusCode", lrw.statusCode, "body", lrw.body.String())
		})
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
	statusCode  int
	body        bytes.Buffer
}

// WriteHeader implements the http.ResponseWriter interface.
func (w *loggingResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
	w.statusCode = code
	w.wroteHeader = true
}

// Write implements the http.ResponseWriter interface.
func (w *loggingResponseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	w.body.Write(body)
	return w.ResponseWriter.Write(body)
}


*/
