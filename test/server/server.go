package main

import (
	"context"
	"errors"
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"golang.org/x/exp/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})))

	m1 := grafanaJSONServer.Metric{
		Label: "Metric 1",
		Value: "foo",
		Payloads: []grafanaJSONServer.MetricPayload{
			{
				Label:        "Option 1",
				Name:         "option1",
				Type:         "select",
				Placeholder:  "",
				ReloadMetric: false,
				Width:        40,
				Options: []grafanaJSONServer.MetricPayloadOption{
					{
						Label: "Value 1",
						Value: "value 1",
					},
					{
						Label: "Value 2",
						Value: "value 2",
					},
				},
			},
			{
				Label: "Option 2",
				Name:  "option2",
				Type:  "multi-select",
				Width: 40,
			},
		},
	}
	m2 := grafanaJSONServer.Metric{
		Label: "Metric 2",
		Value: "bar",
	}

	r := grafanaJSONServer.NewServer(
		grafanaJSONServer.WithLogger(slog.Default()),
		grafanaJSONServer.WithMetric(m1, timeSeriesQuery, getMetricPayloadOptions),
		grafanaJSONServer.WithMetric(m2, tableQuery, nil),
		grafanaJSONServer.WithVariable("query0", func(_ grafanaJSONServer.VariableRequest) ([]grafanaJSONServer.Variable, error) {
			return []grafanaJSONServer.Variable{
				{Text: "Label 1", Value: "Value1"},
				{Text: "Label 2", Value: "Value2"},
				{Text: "Label 3", Value: "Value3"},
			}, nil
		}),
	)

	if err := http.ListenAndServe(":8080", r); !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

func getMetricPayloadOptions(req grafanaJSONServer.MetricPayloadOptionsRequest) ([]grafanaJSONServer.MetricPayloadOption, error) {
	var payload struct {
		Option1 string
		Option2 []string
	}
	slog.Info("getMetricPayloadOptions called", "metric", req.Metric, "name", req.Name)
	if err := req.GetPayload(&payload); err != nil {
		slog.Error("failed", "err", err)
		return nil, err
	}
	return []grafanaJSONServer.MetricPayloadOption{{Value: "one"}, {Value: "two"}}, nil
}

func timeSeriesQuery(_ context.Context, target string, req grafanaJSONServer.QueryRequest) (grafanaJSONServer.QueryResponse, error) {
	var payload struct {
		Option1 string
		Option2 []string
	}
	_ = req.GetPayload(target, &payload)

	resp := grafanaJSONServer.TimeSeriesResponse{Target: target}
	period := req.MaxDataPoints / 10
	timestamp := req.Range.From
	c := 0
	if target == "bar" {
		c = period / 2
	}
	for timestamp.Before(req.Range.To) {
		resp.DataPoints = append(resp.DataPoints, grafanaJSONServer.DataPoint{
			Timestamp: timestamp,
			Value:     float64(c % period),
		})
		c++
		timestamp = timestamp.Add(time.Duration(req.IntervalMs) * time.Millisecond)
	}

	return resp, nil
}

func tableQuery(_ context.Context, target string, req grafanaJSONServer.QueryRequest) (grafanaJSONServer.QueryResponse, error) {
	var timestamps grafanaJSONServer.TimeColumn
	var values grafanaJSONServer.NumberColumn

	period := req.MaxDataPoints / 10
	timestamp := req.Range.From
	c := 0
	if target == "bar" {
		c = period / 2
	}
	for timestamp.Before(req.Range.To) {
		timestamps = append(timestamps, timestamp)
		values = append(values, float64(c%period))
		c++
		timestamp = timestamp.Add(time.Duration(req.IntervalMs) * time.Millisecond)
	}

	return grafanaJSONServer.TableResponse{
		Columns: []grafanaJSONServer.Column{
			{Text: "time", Data: timestamps},
			{Text: "bar", Data: values},
		},
	}, nil
}
