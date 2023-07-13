package data

import "github.com/grafana/grafana-plugin-sdk-go/data"

// Accumulate creates a new Table where the number values are accumulated over subsequent rows
func (t Table) Accumulate() *Table {
	output := t.Frame.EmptyCopy()

	for idx, f := range t.Frame.Fields {
		switch f.Type() {
		case data.FieldTypeFloat64:
			// accumulate values
			var total float64
			output.Fields[idx].Extend(f.Len())
			for i := 0; i < f.Len(); i++ {
				total += f.At(i).(float64)
				output.Fields[idx].Set(i, total)
			}
		default:
			// copy values
			output.Fields[idx].Extend(f.Len())
			for i := 0; i < f.Len(); i++ {
				output.Fields[idx].Set(i, f.At(i))
			}
		}
	}

	return &Table{Frame: output}
}
