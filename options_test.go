package grafana_json_server_test

import (
	"context"
	"encoding/json"
	"errors"
	gjson "github.com/clambin/grafana-json-server"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWithHandlerFunc(t *testing.T) {
	h := gjson.NewServer(gjson.WithHTTPHandler(http.MethodGet, "/extra", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "http://localhost/extra", nil)
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWithPrometheusQueryMetrics(t *testing.T) {
	metrics := gjson.NewDefaultPrometheusQueryMetrics("namespace", "subsystem", "test")
	handler := gjson.NewServer(
		gjson.WithPrometheusQueryMetrics(metrics),
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
	req, _ := http.NewRequest(http.MethodPost, "http://localhost/query", io.NopCloser(strings.NewReader(`{ "targets": [ { "target": "foo" } ] }`)))
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, 1, testutil.CollectAndCount(metrics))

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "http://localhost/query", io.NopCloser(strings.NewReader(`{ "targets": [ { "target": "missing" } ] }`)))
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	assert.NoError(t, testutil.CollectAndCompare(metrics, strings.NewReader(`
# HELP namespace_subsystem_json_query_error_count Grafana JSON Data server count of failed requests
# TYPE namespace_subsystem_json_query_error_count counter
namespace_subsystem_json_query_error_count{application="test",target="missing"} 1
`), `namespace_subsystem_json_query_error_count`))

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "http://localhost/query", io.NopCloser(strings.NewReader(`{ "targets": [ { "target": "fubar" } ] }`)))
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	assert.NoError(t, testutil.CollectAndCompare(metrics, strings.NewReader(`
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
		gjson.WithVariable("", func(request gjson.VariableRequest) ([]gjson.Variable, error) {
			var p map[string]any
			var vars []gjson.Variable
			if err := json.Unmarshal(request.Payload, &p); err == nil {
				for k, v := range p {
					vars = append(vars,
						gjson.Variable{Text: strings.ToTitle(k), Value: k},
						gjson.Variable{Text: strings.ToTitle(v.(string)), Value: v.(string)},
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
			req, _ := http.NewRequest(http.MethodPost, "http://localhost/variable", io.NopCloser(strings.NewReader(tt.request)))
			h.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if w.Code == http.StatusOK {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				assert.Equal(t, tt.want, w.Body.String())
			}
		})
	}
}
