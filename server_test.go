package grafana_json_server_test

import (
	"bytes"
	"context"
	"errors"
	gjson "github.com/clambin/grafana-json-server"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServer_Heartbeat(t *testing.T) {
	s := gjson.NewServer()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	s.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServer_Metrics(t *testing.T) {
	h := gjson.NewServer(
		gjson.WithMetric(
			gjson.Metric{Value: "foo"},
			nil,
			func(_ gjson.MetricPayloadOptionsRequest) ([]gjson.MetricPayloadOption, error) {
				return []gjson.MetricPayloadOption{
					{Label: "Foo", Value: "foo"},
					{Label: "Bar", Value: "bar"},
				}, nil
			},
		))

	testCases := []struct {
		name           string
		request        string
		wantStatusCode int
		wantResponse   string
	}{
		{
			name:           "valid",
			request:        `{ "metric": "foo" }`,
			wantStatusCode: http.StatusOK,
			wantResponse: `[{"value":"foo","payloads":null}]
`,
		},
		{
			name:           "missing",
			request:        `{ "metric": "bar" }`,
			wantStatusCode: http.StatusOK,
			wantResponse: `[]
`,
		},
		{
			name:           "invalid",
			request:        `definitely not a json object`,
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "http://localhost/metrics", io.NopCloser(bytes.NewBuffer([]byte(tt.request))))
			h.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if w.Code == http.StatusOK {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				assert.Equal(t, tt.wantResponse, w.Body.String())
			}

		})
	}
}

func TestServer_MetricPayloadOptions(t *testing.T) {
	h := gjson.NewServer(
		gjson.WithMetric(
			gjson.Metric{Value: "foo"},
			nil,
			func(_ gjson.MetricPayloadOptionsRequest) ([]gjson.MetricPayloadOption, error) {
				return []gjson.MetricPayloadOption{
					{Label: "Foo", Value: "foo"},
					{Label: "Bar", Value: "bar"},
				}, nil
			},
		),
		gjson.WithMetric(
			gjson.Metric{Value: "fubar"},
			nil,
			func(_ gjson.MetricPayloadOptionsRequest) ([]gjson.MetricPayloadOption, error) {
				return nil, errors.New("failing")
			},
		),
		gjson.WithMetric(
			gjson.Metric{Value: "fubar2"},
			nil,
			nil,
		),
	)

	testCases := []struct {
		name           string
		request        string
		wantStatusCode int
		want           string
	}{
		{
			name:           "valid",
			request:        `{ "metric": "foo" }`,
			wantStatusCode: http.StatusOK,
			want: `[{"label":"Foo","value":"foo"},{"label":"Bar","value":"bar"}]
`,
		},
		{
			name:           "missing",
			request:        `{ "metric": "bar" }`,
			wantStatusCode: http.StatusOK,
			want:           "[]\n",
		},
		{
			name:           "failing",
			request:        `{ "metric": "fubar" }`,
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:           "failing 2",
			request:        `{ "metric": "fubar2" }`,
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
			req, _ := http.NewRequest(http.MethodPost, "http://localhost/metric-payload-options", io.NopCloser(bytes.NewBuffer([]byte(tt.request))))
			h.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if w.Code == http.StatusOK {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				assert.Equal(t, tt.want, w.Body.String())
			}
		})
	}
}

func TestServer_WithQuery(t *testing.T) {
	h := gjson.NewServer(
		gjson.WithHandler("foo", gjson.HandlerFunc(func(_ context.Context, target string, _ gjson.QueryRequest) (gjson.QueryResponse, error) {
			return gjson.TimeSeriesResponse{
				Target: "foo",
				DataPoints: []gjson.DataPoint{
					{Timestamp: time.Date(2023, time.July, 15, 0, 0, 0, 0, time.UTC), Value: 10},
				},
			}, nil
		})),
		gjson.WithHandler("bar", gjson.HandlerFunc(func(_ context.Context, target string, _ gjson.QueryRequest) (gjson.QueryResponse, error) {
			return gjson.TableResponse{Columns: []gjson.Column{
				{Text: "time", Data: gjson.TimeColumn([]time.Time{time.Date(2023, time.July, 15, 0, 0, 0, 0, time.UTC)})},
				{Text: "value", Data: gjson.NumberColumn([]float64{10})},
			}}, nil
		})),
		gjson.WithHandler("fubar", gjson.HandlerFunc(func(_ context.Context, target string, _ gjson.QueryRequest) (gjson.QueryResponse, error) {
			return nil, errors.New("fubar")
		})),
		gjson.WithHandler("fubar2", gjson.HandlerFunc(func(_ context.Context, target string, _ gjson.QueryRequest) (gjson.QueryResponse, error) {
			return gjson.TableResponse{Columns: []gjson.Column{
				{Text: "time", Data: gjson.TimeColumn{time.Now()}},
				{Text: "value", Data: gjson.NumberColumn{1, 2, 3}},
			}}, nil
		})),
		gjson.WithHandler("multiple-targets", gjson.HandlerFunc(func(_ context.Context, target string, req gjson.QueryRequest) (gjson.QueryResponse, error) {
			var responsesByTargetPayload map[string]int = map[string]int{
				"first":  1,
				"second": 2,
			}
			var payload struct {
				TargetSeq string
			}
			if err := req.GetPayload(target, &payload); err != nil {
				return nil, err
			}
			return gjson.TimeSeriesResponse{
				Target: "multiple-targets",
				DataPoints: []gjson.DataPoint{
					{Timestamp: time.Date(2023, time.July, 15, 0, 0, 0, 0, time.UTC), Value: float64(responsesByTargetPayload[payload.TargetSeq])},
				},
			}, nil
		})),
	)

	testCases := []struct {
		name           string
		queryRequest   string
		wantStatusCode int
		want           string
	}{
		{
			name:           "timeseries",
			queryRequest:   `{ "targets": [ { "target": "foo" } ] }`,
			wantStatusCode: http.StatusOK,
			want: `[{"target":"foo","datapoints":[[10,1689379200000]]}]
`,
		},
		{
			name:           "timeseries",
			queryRequest:   `{ "targets": [ { "target": "multiple-targets", "refId": "A", "payload": {"targetSeq": "first"} }, { "target": "multiple-targets", "refId": "B", "payload": {"targetSeq": "second"} }]}`,
			wantStatusCode: http.StatusOK,
			want: `[{"target":"multiple-targets","datapoints":[[1,1689379200000]]},{"target":"multiple-targets","datapoints":[[2,1689379200000]]}]
`,
		},
		{
			name:           "table",
			queryRequest:   `{ "targets": [ { "target": "bar" } ] }`,
			wantStatusCode: http.StatusOK,
			want: `[{"type":"table","columns":[{"text":"time","type":"time"},{"text":"value","type":"number"}],"rows":[["2023-07-15T00:00:00Z",10]]}]
`,
		},
		{
			name:           "missing",
			queryRequest:   `{ "targets": [ { "target": "not-a-target" } ] }`,
			wantStatusCode: http.StatusOK,
			want: `[]
`,
		},
		{
			// TODO: should this result in an error (http statuscode?)?
			name:           "failing",
			queryRequest:   `{ "targets": [ { "target": "fubar" } ] }`,
			wantStatusCode: http.StatusOK,
			want: `[]
`,
		},
		{
			name:           "invalid response",
			queryRequest:   `{ "targets": [ { "target": "fubar2" } ] }`,
			wantStatusCode: http.StatusInternalServerError,
			want: `query: json: error calling MarshalJSON for type grafana_json_server.QueryResponse: error building table query output: all columns must have the same number of rows
`,
		},
		{
			name:           "invalid",
			queryRequest:   `not a json object`,
			wantStatusCode: http.StatusBadRequest,
			want: `invalid request: invalid character 'o' in literal null (expecting 'u')
`,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "http://localhost/query", io.NopCloser(bytes.NewBuffer([]byte(tt.queryRequest))))
			h.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			assert.Equal(t, tt.want, w.Body.String())

			if w.Code == http.StatusOK {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestServer_Tags(t *testing.T) {
	s := gjson.NewServer()

	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodPost, "http://localhost/tag-keys", nil)

	s.ServeHTTP(w, r)
	assert.Equal(t, http.StatusNotImplemented, w.Code)

	w = httptest.NewRecorder()
	r, _ = http.NewRequest(http.MethodPost, "http://localhost/tag-values", nil)

	s.ServeHTTP(w, r)
	assert.Equal(t, http.StatusNotImplemented, w.Code)
}
