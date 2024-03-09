package grafana_json_server_test

import (
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"net/http"
)

func Example_variable() {
	s := grafanaJSONServer.NewServer(
		grafanaJSONServer.WithVariable("query0", variableFunc), // this will be called if the payload contains "target": "query0"
		grafanaJSONServer.WithVariable("", variableFunc),       // this will be called if the payload contains no "target"
	)

	_ = http.ListenAndServe(":8080", s)
}

func variableFunc(_ grafanaJSONServer.VariableRequest) ([]grafanaJSONServer.Variable, error) {
	return []grafanaJSONServer.Variable{
		{Text: "Value 1", Value: "value1"},
		{Text: "Value 2", Value: "value2"},
	}, nil
}
