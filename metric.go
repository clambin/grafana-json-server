package grafana_json_server

import (
	"encoding/json"
)

type metricRequest struct {
	Metric  string `json:"metric"`
	Payload struct {
	} `json:"payload"`
}

// A Metric represents one data source offered by a JSON API server.
type Metric struct {
	// Label is the name of the metric, as shown on the screen.
	Label string `json:"label,omitempty"`
	// Value is the internal name of the metric, used in API calls.
	Value string `json:"value"`
	// Payloads configures one or more payload options for the metric.
	Payloads []MetricPayload `json:"payloads"`
}

// A MetricPayload configures a payload options for a metric.
type MetricPayload struct {
	// Label is the name of the option, as shown on the screen.
	Label string `json:"label,omitempty"`
	// Name is the internal name of the metric, used in API calls.
	Name string `json:"name"`
	// Type specifies what kind of option should be provided:
	// 		If the value is select, the UI of the payload is a radio box.
	// 		If the value is multi-select, the UI of the payload is a multi selection box.
	// 		If the value is input, the UI of the payload is an input box.
	// 		If the value is textarea, the UI of the payload is a multiline input box.
	// The default is input.
	Type string `json:"type"`
	// Placeholder specifies the input box / selection box prompt information.
	Placeholder string `json:"placeholder,omitempty"`
	// ReloadMetric specifies whether to overload the metrics API after modifying the value of the payload.
	ReloadMetric bool `json:"reloadMetric,omitempty"`
	// Width specifies the width of the input / selection box width to a multiple of 8px.
	Width int `json:"width,omitempty"`
	// Options lists of the configuration of the options list, if the payload type is select / multi-select.
	// If Options is nil, and the type is select / multi-select, Grafana JSON API datasource will call
	// the MetricPayloadOptionFunc provided to WithMetric to populate the options list.
	Options []MetricPayloadOption `json:"options,omitempty"`
}

// MetricPayloadOption contains one option of a MetricPayload.  Label is the value to display on the screen, while
// Value is the internal name used in API calls.
type MetricPayloadOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// MetricPayloadOptionFunc is the function signature of the metric payload option function, provided to WithMetric.
// It is called when a metric has a payload with a nil Options field and returns the possible options for the requested Metric payload.
type MetricPayloadOptionFunc func(MetricPayloadOptionsRequest) ([]MetricPayloadOption, error)

// MetricPayloadOptionsRequest is the request provided to MetricPayloadOptionFunc.
type MetricPayloadOptionsRequest struct {
	Metric  string          `json:"metric"`
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"payload"`
}

// GetPayload unmarshals the json object from the MetricPayloadOptionsRequest in the provided payload.
func (r MetricPayloadOptionsRequest) GetPayload(payload any) error {
	return json.Unmarshal(r.Payload, payload)
}
