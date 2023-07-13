package grafana_json_server

import (
	"encoding/json"
	"time"
)

type variableRequest struct {
	Target VariableTarget `json:"payload"`
	Range  struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
		Raw  struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"raw"`
	} `json:"range"`
}

type VariableTarget string

func (t *VariableTarget) UnmarshalJSON(body []byte) error {
	var payload struct {
		Target string `json:"target"`
	}
	err := json.Unmarshal(body, &payload)
	if err == nil {
		*t = VariableTarget(payload.Target)
	}
	return err
}

type Variable struct {
	Text  string `json:"__text"`
	Value string `json:"__value"`
}
