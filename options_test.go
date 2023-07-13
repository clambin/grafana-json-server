package grafana_json_server_test

import (
	"bytes"
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
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
