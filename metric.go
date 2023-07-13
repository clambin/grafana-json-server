package grafana_json_server

import (
	"encoding/json"
)

type MetricRequest struct {
	Metric  string `json:"metric"`
	Payload struct {
	} `json:"payload"`
}

type Metric struct {
	Label    string          `json:"label,omitempty"`
	Value    string          `json:"value"`
	Payloads []MetricPayload `json:"payloads"`
}

type MetricPayload struct {
	Label        string                `json:"label,omitempty"`
	Name         string                `json:"name"`
	Type         string                `json:"type"`
	Placeholder  string                `json:"placeholder,omitempty"`
	ReloadMetric bool                  `json:"reloadMetric,omitempty"`
	Width        int                   `json:"width,omitempty"`
	Options      []MetricPayloadOption `json:"options,omitempty"`
}

type MetricPayloadOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type MetricPayloadOptionsRequest struct {
	Metric  string          `json:"metric"`
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"payload"`
}

func (r MetricPayloadOptionsRequest) GetPayload(payload any) error {
	return json.Unmarshal(r.Payload, payload)
}
