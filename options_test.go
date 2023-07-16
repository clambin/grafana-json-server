package grafana_json_server_test

import (
	"bytes"
	"context"
	"errors"
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

func TestWithLogger(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{}))
	h := grafanaJSONServer.NewServer(
		grafanaJSONServer.WithLogger(l),
		grafanaJSONServer.WithMetric(
			grafanaJSONServer.Metric{Value: "foo"},
			nil,
			nil,
		))

	const metricsRequest = `{ "metric": "foo" }`
	const metricResponse = `[{"value":"foo","payloads":null}]
`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "http://localhost/metrics", io.NopCloser(bytes.NewBuffer([]byte(metricsRequest))))
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, metricResponse, w.Body.String())
	assert.Contains(t, buf.String(), `level=INFO msg="request processed" request.from="" request.path=/metrics request.method=POST request.status=200 request.elapsed=`)
}

func TestWithHandlerFunc(t *testing.T) {
	h := grafanaJSONServer.NewServer(grafanaJSONServer.WithHandlerFunc(http.MethodGet, "/extra", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "http://localhost/extra", nil)
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

}

func TestWithPrometheusQueryMetrics(t *testing.T) {
	h := grafanaJSONServer.NewServer(
		grafanaJSONServer.WithPrometheusQueryMetrics("namespace", "subsystem", "test"),
		grafanaJSONServer.WithMetric(
			grafanaJSONServer.Metric{Value: "foo"},
			func(_ context.Context, target string, _ grafanaJSONServer.QueryRequest) (grafanaJSONServer.QueryResponse, error) {
				return grafanaJSONServer.TimeSeriesResponse{
					Target: target,
					DataPoints: []grafanaJSONServer.DataPoint{
						{Timestamp: time.Date(2023, time.July, 15, 0, 0, 0, 0, time.UTC), Value: 10},
					},
				}, nil
			},
			nil,
		),
		grafanaJSONServer.WithMetric(
			grafanaJSONServer.Metric{Value: "fubar"},
			func(_ context.Context, target string, _ grafanaJSONServer.QueryRequest) (grafanaJSONServer.QueryResponse, error) {
				return nil, errors.New("failed")
			},
			nil,
		),
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
