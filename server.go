package grafana_json_server

import (
	"encoding/json"
	"github.com/clambin/go-common/httpserver/middleware"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"golang.org/x/exp/slog"
	"net/http"
)

// The Server structure implements a JSON API server compatible with the JSON API Grafana datasource.
type Server struct {
	handlers  map[string]handler
	variables map[string]VariableFunc
	logger    *slog.Logger
}

type handler struct {
	metric              Metric
	metricPayloadOption MetricPayloadOptionFunc
	queryHandler        QueryFunc
}

// NewServer returns a new JSON API server, configured as per the provided Option items.
func NewServer(options ...Option) http.Handler {
	s := &Server{
		handlers:  make(map[string]handler),
		variables: make(map[string]VariableFunc),
		logger:    slog.Default(),
	}

	for _, option := range options {
		option(s)
	}

	return createRouter(s)
}

func createRouter(s *Server) http.Handler {
	r := chi.NewMux()
	r.Use(chiMiddleware.Heartbeat("/"))
	r.Use(middleware.Logger(s.logger))
	//r.Use(requestLogger(s.logger))
	r.Post("/metrics", s.metrics)
	r.Post("/metric-payload-options", s.metricsPayloadOptions)
	r.Post("/variable", s.variable)
	r.Post("/tag-keys", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotImplemented) })
	r.Post("/tag-values", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotImplemented) })
	r.Post("/query", s.query)
	return r
}

func (s Server) metrics(w http.ResponseWriter, r *http.Request) {
	var queryRequest metricRequest
	err := json.NewDecoder(r.Body).Decode(&queryRequest)
	if err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	metrics := make([]Metric, 0)
	for _, h := range s.handlers {
		if queryRequest.Metric == "" || queryRequest.Metric == h.metric.Value {
			metrics = append(metrics, h.metric)
		}
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(metrics)
}

func (s Server) metricsPayloadOptions(w http.ResponseWriter, r *http.Request) {
	var req MetricPayloadOptionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	h, ok := s.handlers[req.Metric]
	if !ok {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]\n"))
		return
	}

	options, err := h.metricPayloadOption(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	_ = json.NewEncoder(w).Encode(options)
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

	_ = json.NewEncoder(w).Encode(variables)
}

func (s Server) query(w http.ResponseWriter, req *http.Request) {
	var queryRequest QueryRequest
	if err := json.NewDecoder(req.Body).Decode(&queryRequest); err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	responses := make([]QueryResponse, 0)
	for _, t := range queryRequest.Targets {
		h, ok := s.handlers[t.Target]
		if !ok {
			s.logger.Warn("invalid query target", "target", t)
			continue
		}
		resp, err := h.queryHandler(req.Context(), t.Target, queryRequest)
		if err != nil {
			s.logger.Error("query failed", "err", err)
			continue
		}
		responses = append(responses, resp)
	}
	_ = json.NewEncoder(w).Encode(responses)
}

/*
func requestLogger(logger *slog.Logger) func(next http.handler) http.handler {
	return func(next http.handler) http.handler {
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
