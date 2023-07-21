package main

import (
	"context"
	"errors"
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"golang.org/x/exp/slog"
	"math"
	"net/http"
	"os"
	"strconv"
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
		grafanaJSONServer.WithMetric(m1, grafanaJSONServer.HandlerFunc(timeSeriesQuery), getMetricPayloadOptions),
		grafanaJSONServer.WithMetric(m2, grafanaJSONServer.HandlerFunc(tableQuery), nil),
		grafanaJSONServer.WithVariable("query0", func(_ grafanaJSONServer.VariableRequest) ([]grafanaJSONServer.Variable, error) {
			return []grafanaJSONServer.Variable{
				{Text: "1", Value: "1"},
				{Text: "5", Value: "5"},
				{Text: "10", Value: "10"},
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
	scale, _ := getScale(req)
	resp := grafanaJSONServer.TimeSeriesResponse{Target: target}
	period := float64(req.MaxDataPoints) / scale
	timestamp := req.Range.From
	var c float64
	for timestamp.Before(req.Range.To) {
		resp.DataPoints = append(resp.DataPoints, grafanaJSONServer.DataPoint{
			Timestamp: timestamp,
			Value:     100 * math.Cos(c*2*math.Pi/period),
		})
		c++
		timestamp = timestamp.Add(time.Duration(req.IntervalMs) * time.Millisecond)
	}

	return resp, nil
}

func tableQuery(_ context.Context, _ string, req grafanaJSONServer.QueryRequest) (grafanaJSONServer.QueryResponse, error) {
	var timestamps grafanaJSONServer.TimeColumn
	var values grafanaJSONServer.NumberColumn

	scale, _ := getScale(req)
	period := float64(req.MaxDataPoints) / scale
	timestamp := req.Range.From
	var c float64
	for timestamp.Before(req.Range.To) {
		timestamps = append(timestamps, timestamp)
		values = append(values, 100*math.Sin(c*2*math.Pi/period))
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

func getScale(req grafanaJSONServer.QueryRequest) (float64, error) {
	var scopedVars struct {
		Query0 grafanaJSONServer.ScopedVar[string]
	}
	if err := req.GetScopedVars(&scopedVars); err != nil {
		return 0, err
	}

	scale, err := strconv.Atoi(scopedVars.Query0.Value)
	if err != nil {
		return 0, err
	}
	return float64(scale), nil
}
