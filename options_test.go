package grafana_json_server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/clambin/go-common/httpserver/middleware"
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWithRequestLogger(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug, ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
		// Remove time from the output for predictable test output.
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}}))
	h := grafanaJSONServer.NewServer(
		grafanaJSONServer.WithLogger(l),
		grafanaJSONServer.WithRequestLogger(slog.LevelDebug, middleware.RequestLogFormatterFunc(func(r *http.Request, code int, _ time.Duration) []slog.Attr {
			return []slog.Attr{
				slog.String("path", r.URL.Path),
				slog.String("method", r.Method),
				slog.Int("code", code),
			}
		})),
		grafanaJSONServer.WithHandler("foo", nil),
	)

	const metricsRequest = `{ "metric": "foo" }`
	const metricResponse = `[{"value":"foo","payloads":null}]
`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "http://localhost/metrics", io.NopCloser(bytes.NewBuffer([]byte(metricsRequest))))
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, metricResponse, w.Body.String())
	assert.Equal(t, `level=DEBUG msg="http request" path=/metrics method=POST code=200
`, buf.String())
}

func TestWithHandlerFunc(t *testing.T) {
	h := grafanaJSONServer.NewServer(grafanaJSONServer.WithHTTPHandler(http.MethodGet, "/extra", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "http://localhost/extra", nil)
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWithPrometheusQueryMetrics(t *testing.T) {
	h := grafanaJSONServer.NewServer(
		grafanaJSONServer.WithPrometheusQueryMetrics("namespace", "subsystem", "test"),
		grafanaJSONServer.WithHandler("foo", grafanaJSONServer.HandlerFunc(func(_ context.Context, target string, _ grafanaJSONServer.QueryRequest) (grafanaJSONServer.QueryResponse, error) {
			return grafanaJSONServer.TimeSeriesResponse{
				Target: target,
				DataPoints: []grafanaJSONServer.DataPoint{
					{Timestamp: time.Date(2023, time.July, 15, 0, 0, 0, 0, time.UTC), Value: 10},
				},
			}, nil
		})),
		grafanaJSONServer.WithHandler("fubar", grafanaJSONServer.HandlerFunc(func(_ context.Context, target string, _ grafanaJSONServer.QueryRequest) (grafanaJSONServer.QueryResponse, error) {
			return nil, errors.New("failed")
		})),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "http://localhost/query", io.NopCloser(bytes.NewBufferString(`{ "targets": [ { "target": "foo" } ] }`)))
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, 1, testutil.CollectAndCount(h))

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "http://localhost/query", io.NopCloser(bytes.NewBufferString(`{ "targets": [ { "target": "missing" } ] }`)))
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	assert.NoError(t, testutil.CollectAndCompare(h, bytes.NewBufferString(`
# HELP namespace_subsystem_json_query_error_count Grafana JSON Data server count of failed requests
# TYPE namespace_subsystem_json_query_error_count counter
namespace_subsystem_json_query_error_count{application="test",target="missing"} 1
`), `namespace_subsystem_json_query_error_count`))

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "http://localhost/query", io.NopCloser(bytes.NewBufferString(`{ "targets": [ { "target": "fubar" } ] }`)))
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	assert.NoError(t, testutil.CollectAndCompare(h, bytes.NewBufferString(`
# HELP namespace_subsystem_json_query_error_count Grafana JSON Data server count of failed requests
# TYPE namespace_subsystem_json_query_error_count counter
namespace_subsystem_json_query_error_count{application="test",target="fubar"} 1
namespace_subsystem_json_query_error_count{application="test",target="missing"} 1
`), `namespace_subsystem_json_query_error_count`))
}

func TestServer_WithVariable(t *testing.T) {
	h := grafanaJSONServer.NewServer(
		grafanaJSONServer.WithVariable("foo", func(_ grafanaJSONServer.VariableRequest) ([]grafanaJSONServer.Variable, error) {
			return []grafanaJSONServer.Variable{{Text: "Foo", Value: "foo"}, {Text: "Bar", Value: "bar"}}, nil
		}),
		grafanaJSONServer.WithVariable("fubar", func(_ grafanaJSONServer.VariableRequest) ([]grafanaJSONServer.Variable, error) {
			return nil, errors.New("failed")
		}),
		grafanaJSONServer.WithVariable("", func(request grafanaJSONServer.VariableRequest) ([]grafanaJSONServer.Variable, error) {
			var p map[string]any
			var vars []grafanaJSONServer.Variable
			if err := json.Unmarshal(request.Payload, &p); err == nil {
				for k, v := range p {
					vars = append(vars,
						grafanaJSONServer.Variable{Text: strings.ToTitle(k), Value: k},
						grafanaJSONServer.Variable{Text: strings.ToTitle(v.(string)), Value: v.(string)},
					)
				}
			}
			return vars, nil
		}),
	)

	tests := []struct {
		name           string
		request        string
		wantStatusCode int
		want           string
	}{
		{
			name:           "named",
			request:        `{ "payload": { "target": "foo" } }`,
			wantStatusCode: http.StatusOK,
			want: `[{"__text":"Foo","__value":"foo"},{"__text":"Bar","__value":"bar"}]
`,
		},
		{
			name:           "unnamed",
			request:        `{ "payload": { "foo": "bar"} }`,
			wantStatusCode: http.StatusOK,
			want: `[{"__text":"FOO","__value":"foo"},{"__text":"BAR","__value":"bar"}]
`,
		},
		{
			name:           "missing",
			request:        `{ "payload": { "target": "bar" } }`,
			wantStatusCode: http.StatusBadRequest,
			want: `[]
`,
		},
		{
			name:           "failure",
			request:        `{ "payload": { "target": "fubar" } }`,
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:           "invalid",
			request:        `not a json object`,
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "http://localhost/variable", io.NopCloser(bytes.NewBuffer([]byte(tt.request))))
			h.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if w.Code == http.StatusOK {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				assert.Equal(t, tt.want, w.Body.String())
			}
		})
	}
}
