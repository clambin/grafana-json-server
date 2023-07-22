package grafana_json_server_test

import (
	"bytes"
	"context"
	"errors"
	"github.com/clambin/go-common/httpserver/middleware"
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
	"io"
	"net/http"
	"net/http/httptest"
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
