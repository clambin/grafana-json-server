package grafana_json_server_test

import (
	gjson "github.com/clambin/grafana-json-server"
	"net/http"
)

func Example_variable() {
	s := gjson.NewServer(
		gjson.WithVariable("query0", variableFunc), // this will be called if the payload contains "target": "query0"
		gjson.WithVariable("", variableFunc),       // this will be called if the payload contains no "target"
	)

	_ = http.ListenAndServe(":8080", s)
}

func variableFunc(_ gjson.VariableRequest) ([]gjson.Variable, error) {
	return []gjson.Variable{
		{Text: "Value 1", Value: "value1"},
		{Text: "Value 2", Value: "value2"},
	}, nil
}
