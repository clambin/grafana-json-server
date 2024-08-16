package grafana_json_server_test

import (
	"context"
	gjson "github.com/clambin/grafana-json-server"
	"net/http"
	"time"
)

func Example_basicTimeSeriesQuery() {
	s := gjson.NewServer(gjson.WithHandler("metric1", gjson.HandlerFunc(basicTimeSeriesQueryFunc)))
	_ = http.ListenAndServe(":8080", s)
}

func basicTimeSeriesQueryFunc(_ context.Context, target string, _ gjson.QueryRequest) (gjson.QueryResponse, error) {
	return gjson.TimeSeriesResponse{
		Target: target,
		DataPoints: []gjson.DataPoint{
			{Timestamp: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), Value: 100},
			{Timestamp: time.Date(2020, 1, 1, 0, 1, 0, 0, time.UTC), Value: 101},
			{Timestamp: time.Date(2020, 1, 1, 0, 2, 0, 0, time.UTC), Value: 103},
		},
	}, nil
}
