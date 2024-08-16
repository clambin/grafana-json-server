package grafana_json_server_test

import (
	"context"
	gjson "github.com/clambin/grafana-json-server"
	"net/http"
	"time"
)

func Example_basicTableQuery() {
	s := gjson.NewServer(gjson.WithHandler("metric1", gjson.HandlerFunc(basicTableQueryFunc)))
	_ = http.ListenAndServe(":8080", s)
}

func basicTableQueryFunc(_ context.Context, _ string, _ gjson.QueryRequest) (gjson.QueryResponse, error) {
	return gjson.TableResponse{
		Columns: []gjson.Column{
			{Text: "Time", Data: gjson.TimeColumn{
				time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2020, 1, 1, 0, 1, 0, 0, time.UTC)},
			},
			{Text: "Label", Data: gjson.StringColumn{"foo", "bar"}},
			{Text: "Series A", Data: gjson.NumberColumn{42, 43}},
			{Text: "Series B", Data: gjson.NumberColumn{64.5, 100.0}},
		},
	}, nil
}
