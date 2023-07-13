package data

import (
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"time"
)

// CreateTableResponse creates a simplejson TableResponse from a Dataset
func (t Table) CreateTableResponse() *grafanaJSONServer.TableResponse {
	columns := make([]grafanaJSONServer.Column, len(t.Frame.Fields))

	for i, f := range t.Frame.Fields {
		columns[i] = makeColumn(f)
	}

	return &grafanaJSONServer.TableResponse{Columns: columns}
}

func makeColumn(f *data.Field) (column grafanaJSONServer.Column) {
	name := f.Name
	if name == "" {
		name = "(unknown)"
	}

	var values interface{}
	if f.Len() > 0 {
		switch f.At(0).(type) {
		case time.Time:
			values = grafanaJSONServer.TimeColumn(getFieldValues[time.Time](f))
		case string:
			values = grafanaJSONServer.StringColumn(getFieldValues[string](f))
		case float64:
			values = grafanaJSONServer.NumberColumn(getFieldValues[float64](f))
		}
	}
	return grafanaJSONServer.Column{
		Text: name,
		Data: values,
	}
}
