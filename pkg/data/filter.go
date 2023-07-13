package data

import (
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"time"
)

// Filter returns a Dataset filters on a Range. Only the first time column is taken into consideration.
func (t Table) Filter(args grafanaJSONServer.Range) (filtered *Table) {
	index, found := t.getFirstTimestampColumn()
	if !found {
		return &Table{Frame: t.Frame.EmptyCopy()}
	}

	f, _ := t.Frame.FilterRowsByField(index, func(i interface{}) (bool, error) {
		if !args.From.IsZero() && i.(time.Time).Before(args.From) {
			return false, nil
		}
		if !args.To.IsZero() && i.(time.Time).After(args.To) {
			return false, nil
		}
		return true, nil
	})

	return &Table{Frame: f}
}
