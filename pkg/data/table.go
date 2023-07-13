package data

import (
	"github.com/clambin/go-common/set"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"time"
)

// Table is a convenience structure to create tables to response to SimpleJSON Table Queries
type Table struct {
	Frame *data.Frame
}

// Column is used by New to specify the columns to create
type Column struct {
	//  Name of the column
	Name string
	// Values for the column
	Values interface{}
}

// New creates a new Table with the specified Column fields
func New(columns ...Column) (table *Table) {
	var fields data.Fields
	for _, column := range columns {
		fields = append(fields, data.NewField(column.Name, nil, column.Values))
	}

	return &Table{Frame: data.NewFrame("frame", fields...)}
}

// GetTimestamps returns the dataset's timestamps
func (t Table) GetTimestamps() (timestamps []time.Time) {
	if index, found := t.getFirstTimestampColumn(); found {
		for i := 0; i < t.Frame.Fields[index].Len(); i++ {
			timestamps = append(timestamps, t.Frame.Fields[0].At(i).(time.Time))
		}
	}
	return
}

func (t Table) getFirstTimestampColumn() (index int, found bool) {
	for i, f := range t.Frame.Fields {
		if f.Len() > 0 {
			if _, ok := f.At(0).(time.Time); ok {
				return i, true
			}
		}
	}
	return
}

// GetColumns returns the dataset's column names
func (t Table) GetColumns() (columns []string) {
	for _, field := range t.Frame.Fields {
		columns = append(columns, field.Name)
	}
	return
}

// GetValues returns the values for the specified column name. If the column does not exist, found will be false
func (t Table) GetValues(column string) (values []interface{}, found bool) {
	if f, n := t.Frame.FieldByName(column); n != -1 {
		found = true
		for i := 0; i < f.Len(); i++ {
			values = append(values, f.At(i))
		}
	}
	return
}

// GetTimeValues returns the time values for the specified column name. If the column does not exist, found will be false.
// Will panic if the data in the provided column is of the wrong type.
func (t Table) GetTimeValues(column string) (values []time.Time, found bool) {
	f, n := t.Frame.FieldByName(column)
	if n == -1 {
		return nil, false
	}
	return getFieldValues[time.Time](f), true
}

// GetFloatValues returns the float64 values for the specified column name. If the column does not exist, found will be false.
// Will panic if the data in the provided column is of the wrong type.
func (t Table) GetFloatValues(column string) (values []float64, found bool) {
	f, n := t.Frame.FieldByName(column)
	if n == -1 {
		return nil, false
	}
	return getFieldValues[float64](f), true
}

// GetStringValues returns the string values for the specified column name. If the column does not exist, found will be false
// Will panic if the data in the provided column is of the wrong type.
func (t Table) GetStringValues(column string) (values []string, found bool) {
	f, n := t.Frame.FieldByName(column)
	if n == -1 {
		return nil, false
	}
	return getFieldValues[string](f), true
}

func getFieldValues[T any](f *data.Field) (values []T) {
	values = make([]T, f.Len())
	for i := 0; i < f.Len(); i++ {
		values[i] = f.At(i).(T)
	}
	return
}

// DeleteColumn returns a table with the listed columns removed
func (t Table) DeleteColumn(columns ...string) *Table {
	fields := make([]*data.Field, 0, len(t.Frame.Fields))
	s := set.Create(columns...)
	for _, field := range t.Frame.Fields {
		if !s.Contains(field.Name) {
			fields = append(fields, field)
		}
	}
	return &Table{Frame: data.NewFrame(t.Frame.Name, fields...)}
}
