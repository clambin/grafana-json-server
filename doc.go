/*
Package grafana_json_server provides a Go implementation of the [JSON API Grafana Datasource], which provides
a way of sending JSON-formatted data into Grafana dataframes.

# Creating a JSON API server for a metric

A metric is the source of data to be sent to Grafana.  In its simplest form, running a JSON API server for a metric
requires creating a server for that metric and starting an HTTP listener:

	s := grafana_json_server.NewServer(
		grafana_json_server.WithQuery("metric1", queryFunc),
	)
	_ = http.ListenAndServe(":8080", s)

This starts a JSON API server for a single metric, called 'metric1' and a query function queryFunc, which will generate
the data for the metric.

To provide more configuration options for the metric, use WithMetric instead:

	metric := grafana_json_server.Metric{
		Label: "My first metric",
		Value: "metric1",
	}

	s := grafana_json_server.NewServer(
		grafana_json_server.WithMetric(metric, queryFunc, nil),
	)
	_ = http.ListenAndServe(":8080", s)

# Writing query functions

The query function produces the data to be sent to Grafana. Queries can be of one of two types:

  - time series queries return values as a list of timestamp/value tuples.
  - table queries return data organized in columns and rows.  Each column needs to have the same number of rows

Time series queries can therefore only return a single set of values.  If your query involves returning multiple sets of
data, use table queries instead.

# Writing time series queries

A time series query returns a TimeSeriesResponse:

	func timeSeriesFunc(_ context.Context, target string, req grafanaJSONServer.QueryRequest) (grafanaJSONServer.QueryResponse, error) {
		return grafanaJSONServer.TimeSeriesResponse{
			Target: target,
			DataPoints: []grafanaJSONServer.DataPoint{
				{Timestamp: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), Value: 100},
				{Timestamp: time.Date(2020, 1, 1, 0, 1, 0, 0, time.UTC), Value: 101},
				{Timestamp: time.Date(2020, 1, 1, 0, 2, 0, 0, time.UTC), Value: 103},
			},
		}, nil
	}

# Writing table queries

A table query returns a TableResponse:

	func tableFunc(_ context.Context, _ string, _ grafanaJSONServer.QueryRequest) (grafanaJSONServer.QueryResponse, error) {
		return grafanaJSONServer.TableResponse{
			Columns: []grafanaJSONServer.Column{
				{Text: "Time", Data: grafanaJSONServer.TimeColumn{
					time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					time.Date(2020, 1, 1, 0, 1, 0, 0, time.UTC)}},
				{Text: "Label", Data: grafanaJSONServer.StringColumn{"foo", "bar"}},
				{Text: "Series A", Data: grafanaJSONServer.NumberColumn{42, 43}},
				{Text: "Series B", Data: grafanaJSONServer.NumberColumn{64.5, 100.0}},
			},
		}, nil
	}

Note that the table must be 'complete', i.e. each column should have the same number of entries.

# Metric Payload Options

The JSON API Grafana Datasource allows each metric to have a number of user-selectable options. In the Grafana Edit panel,
these are shown in the Payload section of the metric.

To add payload options to the metric, configure these when creating the metric:

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

The above example configures a new metric, with one payload option, called "option". The option allows multiple values to be selected.
Two possible options are configured: option1 and option2.

When performing the query, the query function can check which options are selected by reading the request's payload:

	var payload struct {
		Option []string
	}
	_ = req.GetPayload(target, &payload)

The payload Option field will contain all selected options, i.e. option1, option2.  If no options are selected, Option will be an empty slice.

Note: the payload structure must match the metric's payload definition. Otherwise GetPayload returns an error.
With the above metric definition, the following will fail:

	var payload struct {
		Option string
	}
	_ = req.GetPayload(target, &payload)

Since Option is a multi-select option, a slice is expected.

# Dynamic Metric Payload Options

In the previous section, we configured a metric with hard-coded options. If we want to have dynamic options, we use
the MetricPayloadOption function when creating the metric.  Possible use cases for this could be if runtime conditions
determine which options you want to present, or if you want to determine valid options based on what other options are selected.

	metric := grafanaJSONServer.Metric{
		Label: "my advanced metric",
		Value: "metric1",
		Payloads: []grafanaJSONServer.MetricPayload{
			{Label: "Option 1", Name: "option1", Type: "select", Width: 40, Options: []grafanaJSONServer.MetricPayloadOption{
				{Label: "Mode 1", Value: "mode1"},
				{Label: "Mode 2", Value: "mode2"},
			}},
			{Label: "Option 2", Name: "option2", Type: "select", Width: 40},
		},
	}

	s := grafanaJSONServer.NewServer(grafanaJSONServer.WithMetric(metric, metricOptionsDynamicQueryFunc, metricPayloadOptionsFunc))

This creates a single metric, metric1.  The metric has two payload options, option1 and option2. The former has two hardcoded options.
The latter has no Options configured.  This will call JSON API DataSource to call metricPayloadOptionsFunc to determine which
options to present.

The following is a basic example of such a MetricPayloadOption function:

		func metricPayloadOptionsFunc(req grafanaJSONServer.MetricPayloadOptionsRequest) ([]grafanaJSONServer.MetricPayloadOption, error) {
		var payload struct {
			Option1 string
			Option2 string
		}
		if err := req.GetPayload(&payload); err != nil {
			return nil, err
		}

		// payload will now contain all selected options across all metric's payloads
		// req.Name tells us for which metric the function was called

		return []grafanaJSONServer.MetricPayloadOption{
			{Label: "Value 1", Value: "value1"},
			{Label: "Value 2", Value: "value2"},
		}, nil
	}

# Variables

The JSON API datasource supports dashboard variable values to be retrieved from an JSON API server. To configure this,
add a dashboard variable with the variable type set to "Query" and the data source to your JSON API server.

In the server, create the server with the WithVariable option:

	s := grafanaJSONServer.NewServer(
		grafanaJSONServer.WithVariable("query0", variableFunc),
	)
	_ = http.ListenAndServe(":8080", s)

In the example, "query0" is the name of the dashboard variable.

This causes Grafana to call the variableFunc whenever the variable is refreshed.  This function returns all possible
values for the variable:

	func variableFunc(_ grafanaJSONServer.VariableRequest) ([]grafanaJSONServer.Variable, error) {
		return []grafanaJSONServer.Variable{
			{Text: "Value 1", Value: "value1"},
			{Text: "Value 2", Value: "value2"},
		}, nil
	}

A Query function can read the value of each variables by examining the ScopedVars in the QueryRequest:

	var req grafanaJSONServer.QueryRequest
	var scopedVars struct {
		Var1   grafanaJSONServer.ScopedVar[string]
		Query0 grafanaJSONServer.ScopedVar[[]string]
	}
	_ = req.GetScopedVars(&scopedVars))

[JSON API Grafana Datasource]: https://github.com/simPod/GrafanaJsonDatasource
*/
package grafana_json_server
