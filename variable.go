package grafana_json_server

import (
	"encoding/json"
)

// VariableFunc is the function signature of function provided to WithVariable.
// It returns a list of possible values for a dashboard variable.
type VariableFunc func(VariableRequest) ([]Variable, error)

// VariableRequest is the request sent to VariableFunc.
//
// Payload and Target are determined by the Grafana definition of the variable:
//   - if Raw JSON is off, Payload contains a JSON object with a single field "target", as set in the Variable's Query field.
//   - if Raw JSON is on, Payload contains the JSON object set in the Variable's Query field.
//
// In both cases, if the Payload contains a field "target", its value is stored in Target. If no "target" exists, Target is blank. No error is raised.
type VariableRequest struct {
	Payload json.RawMessage `json:"payload"`
	Range   Range           `json:"range"`
	Target  string
}

func (v *VariableRequest) UnmarshalJSON(bytes []byte) error {
	type v2 VariableRequest
	var req2 v2
	if err := json.Unmarshal(bytes, &req2); err != nil {
		return err
	}
	*v = VariableRequest(req2)
	var payload struct {
		Target string `json:"target"`
	}
	err := json.Unmarshal(v.Payload, &payload)
	if err == nil {
		v.Target = payload.Target
	}
	return err
}

// Variable is one possible value for a dashboard value.
// Text is the name to be displayed on the screen. Value will be used in API calls.
type Variable struct {
	Text  string `json:"__text"`
	Value string `json:"__value"`
}
