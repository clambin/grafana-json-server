package grafana_json_server

import (
	"encoding/json"
	"io"
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
	Target  string
	Payload json.RawMessage `json:"payload"`
	Range   Range           `json:"range"`
}

// Variable is one possible value for a dashboard value.
// Text is the name to be displayed on the screen. Value will be used in API calls.
type Variable struct {
	Text  string `json:"__text"`
	Value string `json:"__value"`
}

func parseVariableRequest(r io.Reader) (VariableRequest, error) {
	var req VariableRequest
	err := json.NewDecoder(r).Decode(&req)
	if err != nil {
		return req, err
	}

	var payload struct {
		Target string `json:"target"`
	}
	err = json.Unmarshal(req.Payload, &payload)
	if err == nil {
		req.Target = payload.Target
	}
	return req, err
}
