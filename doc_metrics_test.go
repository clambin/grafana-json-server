package grafana_json_server_test

import (
	"context"
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"net/http"
	"time"
)

func Example_metricOptions() {
	metric := grafanaJSONServer.Metric{
		Label: "my advanced metric",
		Value: "metric1",
		Payloads: []grafanaJSONServer.MetricPayload{
			{
				Label: "Option",
				Name:  "option",
				Type:  "multi-select",
				Width: 40,
				Options: []grafanaJSONServer.MetricPayloadOption{
					{Label: "Option 1", Value: "option1"},
					{Label: "Option 2", Value: "option2"},
				},
			},
		},
	}

	s := grafanaJSONServer.NewServer(grafanaJSONServer.WithMetric(metric, metricOptionsQueryFunc, nil))
	_ = http.ListenAndServe(":8080", s)
}

func metricOptionsQueryFunc(_ context.Context, target string, req grafanaJSONServer.QueryRequest) (grafanaJSONServer.QueryResponse, error) {
	var payload struct {
		Option []string
	}
	if err := req.GetPayload(target, &payload); err != nil {
		return nil, err
	}
	// payload Option will now contain all selected options, i.e. option1, option2.  If no options are selected, Option will be an empty slice.
	return grafanaJSONServer.TimeSeriesResponse{
		Target: target,
		DataPoints: []grafanaJSONServer.DataPoint{
			{Timestamp: time.Now(), Value: 1.0},
		},
	}, nil
}
