package grafana_json_server_test

import (
	"encoding/json"
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMetricPayloadOptionsRequest_GetPayload(t *testing.T) {
	req := grafanaJSONServer.MetricPayloadOptionsRequest{
		Metric:  "foo",
		Name:    "Foo",
		Payload: json.RawMessage(`{ "bar": "snafu" }`),
	}

	var payload struct {
		Bar string
	}
	assert.NoError(t, req.GetPayload(&payload))
	assert.Equal(t, "snafu", payload.Bar)
}
