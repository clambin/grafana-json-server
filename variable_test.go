package grafana_json_server_test

import (
	"encoding/json"
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVariableRequest_UnmarshalJSON(t *testing.T) {
	type want struct {
		err     assert.ErrorAssertionFunc
		payload string
		target  string
	}
	tests := []struct {
		name  string
		input []byte
		want  want
	}{
		{
			name:  "empty",
			input: []byte(``),
			want:  want{err: assert.Error},
		},
		{
			name:  "invalid json",
			input: []byte(`{ "range": { "raw": { "from": 1 } } }`),
			want:  want{err: assert.Error},
		},
		{
			name:  "standard",
			input: []byte(`{ "payload": { "target": "foo" } }`),
			want: want{
				err:     assert.NoError,
				payload: `{ "target": "foo" }`,
				target:  "foo",
			},
		},
		{
			name:  "raw json",
			input: []byte(`{ "payload": { "target": "foo", "args": "bar" } }`),
			want: want{
				err:     assert.NoError,
				payload: `{ "target": "foo", "args": "bar" }`,
				target:  "foo",
			},
		},
		{
			name:  "raw json - no target",
			input: []byte(`{ "payload": { "args": "bar" } }`),
			want: want{
				err:     assert.NoError,
				payload: `{ "args": "bar" }`,
				target:  "",
			},
		},
		{
			name:  "raw json - invalid payload",
			input: []byte(`{ "payload": { args: "bar" } }`),
			want: want{
				err: assert.Error,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v grafanaJSONServer.VariableRequest
			err := json.Unmarshal(tt.input, &v)
			tt.want.err(t, err)
			assert.Equal(t, tt.want.payload, string(v.Payload))
			assert.Equal(t, tt.want.target, v.Target)
		})
	}
}
