package grafana_json_server_test

import (
	"bytes"
	"errors"
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServer_WithVariable(t *testing.T) {
	h := grafanaJSONServer.NewServer(
		grafanaJSONServer.WithVariable("foo", func(_ grafanaJSONServer.VariableRequest) ([]grafanaJSONServer.Variable, error) {
			return []grafanaJSONServer.Variable{{Text: "Foo", Value: "foo"}, {Text: "Bar", Value: "bar"}}, nil
		}),
		grafanaJSONServer.WithVariable("fubar", func(_ grafanaJSONServer.VariableRequest) ([]grafanaJSONServer.Variable, error) {
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
