package grafana_json_server_test

import (
	"bytes"
	"context"
	"errors"
	gjson "github.com/clambin/grafana-json-server"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWithHandlerFunc(t *testing.T) {
	h := gjson.NewServer(
		gjson.WithLogger(slog.Default()),
		gjson.WithHTTPHandler(http.MethodGet, "/extra", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "http://localhost/extra", nil)
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWithPrometheusQueryMetrics(t *testing.T) {
	h := gjson.NewServer(
		gjson.WithPrometheusQueryMetrics("namespace", "subsystem", "test"),
		gjson.WithHandler("foo", gjson.HandlerFunc(func(_ context.Context, target string, _ gjson.QueryRequest) (gjson.QueryResponse, error) {
			return gjson.TimeSeriesResponse{
				Target: target,
				DataPoints: []gjson.DataPoint{
					{Timestamp: time.Date(2023, time.July, 15, 0, 0, 0, 0, time.UTC), Value: 10},
				},
			}, nil
		})),
		gjson.WithHandler("fubar", gjson.HandlerFunc(func(_ context.Context, target string, _ gjson.QueryRequest) (gjson.QueryResponse, error) {
			return nil, errors.New("failed")
		})),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "http://localhost/query", io.NopCloser(bytes.NewBufferString(`{ "targets": [ { "target": "foo" } ] }`)))
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

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
	h := gjson.NewServer(
		gjson.WithVariable("foo", func(_ gjson.VariableRequest) ([]gjson.Variable, error) {
			return []gjson.Variable{{Text: "Foo", Value: "foo"}, {Text: "Bar", Value: "bar"}}, nil
		}),
		gjson.WithVariable("fubar", func(_ gjson.VariableRequest) ([]gjson.Variable, error) {
			return nil, errors.New("failed")
		}),
	)

	testCases := []struct {
		name           string
		request        string
		wantStatusCode int
		want           string
	}{
		{
			name:           "valid",
			request:        `{ "payload": { "target": "foo" } }`,
			wantStatusCode: http.StatusOK,
			want: `[{"__text":"Foo","__value":"foo"},{"__text":"Bar","__value":"bar"}]
`,
		},
		{
			name:           "missing",
			request:        `{ "payload": { "target": "bar" } }`,
			wantStatusCode: http.StatusOK,
			want:           "[]\n",
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

	for _, tt := range testCases {
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
