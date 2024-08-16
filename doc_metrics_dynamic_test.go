package grafana_json_server_test

import (
	"context"
	gjson "github.com/clambin/grafana-json-server"
	"net/http"
	"time"
)

func Example_metricOptionsDynamic() {
	metric := gjson.Metric{
		Label: "my advanced metric",
		Value: "metric1",
		Payloads: []gjson.MetricPayload{
			{Label: "Option 1", Name: "option1", Type: "select", Width: 40, Options: []gjson.MetricPayloadOption{
				{Label: "Mode 1", Value: "mode1"},
				{Label: "Mode 2", Value: "mode2"},
			}},
			{Label: "Option 2", Name: "option2", Type: "select", Width: 40},
		},
	}

	s := gjson.NewServer(gjson.WithMetric(metric, gjson.HandlerFunc(metricOptionsDynamicQueryFunc), metricPayloadOptionsFunc))
	_ = http.ListenAndServe(":8080", s)
}

func metricPayloadOptionsFunc(req gjson.MetricPayloadOptionsRequest) ([]gjson.MetricPayloadOption, error) {
	var payload struct {
		Option1 string
		Option2 string
	}
	if err := req.GetPayload(&payload); err != nil {
		return nil, err
	}
	// payload will now contain all selected options across all metric's payloads

	// req.Name tells us for metric payload the function was called

	return []gjson.MetricPayloadOption{
		{Label: "Value 1", Value: "value1"},
		{Label: "Value 2", Value: "value2"},
	}, nil
}

func metricOptionsDynamicQueryFunc(_ context.Context, target string, req gjson.QueryRequest) (gjson.QueryResponse, error) {
	var payload struct {
		Option1 string
		Option2 string
	}
	if err := req.GetPayload(target, &payload); err != nil {
		return nil, err
	}
	// payload will now contain all selected options across all metric's payloads

	return gjson.TimeSeriesResponse{
		Target: target,
		DataPoints: []gjson.DataPoint{
			{Timestamp: time.Now(), Value: 1.0},
		},
	}, nil
}
