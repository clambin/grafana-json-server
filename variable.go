package grafana_json_server

import (
	"encoding/json"
	"time"
)

// VariableFunc is the function signature of function provided to WithVariable.
// It returns a list of possible values for a dashboard variable.
type VariableFunc func(VariableRequest) ([]Variable, error)

// VariableRequest is the request sent to VariableFunc. Target is the name of the variable, as provided to WithVariable.
type VariableRequest struct {
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

// VariableTarget is the name of the dashboard variable, as provided to WithVariable.
type VariableTarget string

// UnmarshalJSON unmarshals a VariableRequest's Target to a string.
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

// Variable is one possible value for a dashboard value.
// Text is the name to be displayed on the screen. Value will be used in API calls.
type Variable struct {
	Text  string `json:"__text"`
	Value string `json:"__value"`
}
