package grafana_json_server_test

import (
	"context"
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"net/http"
	"time"
)

func Example_basicTableQuery() {
	metric := grafanaJSONServer.Metric{
		Label: "My first metric",
		Value: "metric1",
	}

	s := grafanaJSONServer.NewServer(
		grafanaJSONServer.WithMetric(metric, basicTableQueryFunc, nil),
	)

	_ = http.ListenAndServe(":8080", s)
}

func basicTableQueryFunc(_ context.Context, _ string, _ grafanaJSONServer.QueryRequest) (grafanaJSONServer.QueryResponse, error) {
	return grafanaJSONServer.TableResponse{
		Columns: []grafanaJSONServer.Column{
			{Text: "Time", Data: grafanaJSONServer.TimeColumn{
				time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2020, 1, 1, 0, 1, 0, 0, time.UTC)},
			},
			{Text: "Label", Data: grafanaJSONServer.StringColumn{"foo", "bar"}},
			{Text: "Series A", Data: grafanaJSONServer.NumberColumn{42, 43}},
			{Text: "Series B", Data: grafanaJSONServer.NumberColumn{64.5, 100.0}},
		},
	}, nil
}
