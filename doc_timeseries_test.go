package grafana_json_server_test

import (
	"context"
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"net/http"
	"time"
)

func Example_basicTimeSeriesQuery() {
	s := grafanaJSONServer.NewServer(grafanaJSONServer.WithHandler("metric1", grafanaJSONServer.HandlerFunc(basicTimeSeriesQueryFunc)))
	_ = http.ListenAndServe(":8080", s)
}

func basicTimeSeriesQueryFunc(_ context.Context, target string, _ grafanaJSONServer.QueryRequest) (grafanaJSONServer.QueryResponse, error) {
	return grafanaJSONServer.TimeSeriesResponse{
		Target: target,
		DataPoints: []grafanaJSONServer.DataPoint{
			{Timestamp: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), Value: 100},
			{Timestamp: time.Date(2020, 1, 1, 0, 1, 0, 0, time.UTC), Value: 101},
			{Timestamp: time.Date(2020, 1, 1, 0, 2, 0, 0, time.UTC), Value: 103},
		},
	}, nil
}
