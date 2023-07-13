package data_test

import (
	"github.com/clambin/grafana-json-server/pkg/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
	"time"
)

func TestTable_New(t *testing.T) {
	table := createTable(10)

	assert.Len(t, table.Frame.Fields, 4)
	assert.Equal(t, "time", table.Frame.Fields[0].Name)
	assert.Equal(t, 10, table.Frame.Fields[0].Len())
	assert.Equal(t, "values", table.Frame.Fields[1].Name)
	assert.Equal(t, 10, table.Frame.Fields[1].Len())
	assert.Equal(t, "", table.Frame.Fields[2].Name)
	assert.Equal(t, 10, table.Frame.Fields[2].Len())
	assert.Equal(t, "labels", table.Frame.Fields[3].Name)
	assert.Equal(t, 10, table.Frame.Fields[3].Len())
}

func TestTable_GetTimestamps(t *testing.T) {
	table := createTable(10)
	timestamps := table.GetTimestamps()
	assert.Equal(t, []time.Time{
		time.Date(2022, time.June, 4, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 6, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 7, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 8, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 9, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 11, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 12, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 13, 0, 0, 0, 0, time.UTC),
	}, timestamps)
}

func TestTable_GetColumns(t *testing.T) {
	table := createTable(1)
	columns := table.GetColumns()
	assert.Equal(t, []string{"time", "values", "", "labels"}, columns)
}

func TestTable_GetValues(t *testing.T) {
	table := createTable(10)
	values, found := table.GetValues("values")
	require.True(t, found)
	assert.Equal(t, []interface{}{0.0, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0}, values)

	values, found = table.GetValues("labels")
	require.True(t, found)
	assert.Equal(t, []interface{}{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}, values)

	_, found = table.GetValues("invalid")
	assert.False(t, found)
}

func TestTable_GetTimeValues(t *testing.T) {
	table := createTable(10)
	values, found := table.GetTimeValues("time")
	require.True(t, found)
	assert.Equal(t, []time.Time{
		time.Date(2022, time.June, 4, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 6, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 7, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 8, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 9, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 11, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 12, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 13, 0, 0, 0, 0, time.UTC),
	}, values)

	_, found = table.GetTimeValues("not a column")
	assert.False(t, found)
}

func TestTable_GetFloatValues(t *testing.T) {
	table := createTable(10)
	values, found := table.GetFloatValues("values")
	require.True(t, found)
	assert.Equal(t, []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, values)

	_, found = table.GetFloatValues("not a column")
	assert.False(t, found)
}

func TestTable_GetStringValues(t *testing.T) {
	table := createTable(10)
	values, found := table.GetStringValues("labels")
	require.True(t, found)
	assert.Equal(t, []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}, values)

	_, found = table.GetStringValues("not a column")
	assert.False(t, found)
}

func TestTable_DeleteColumn(t *testing.T) {
	table := createTable(10)

	table = table.DeleteColumn("labels")
	assert.Equal(t, []string{"time", "values", ""}, table.GetColumns())

	table = table.DeleteColumn("labels")
	assert.Equal(t, []string{"time", "values", ""}, table.GetColumns())

	table = table.DeleteColumn("time")
	assert.Equal(t, []string{"values", ""}, table.GetColumns())

	table = table.DeleteColumn("values", "")
	assert.Empty(t, table.GetColumns())
}

func createTable(rows int) *data.Table {
	var timestamps []time.Time
	var values []float64
	var labels []string
	timestamp := time.Date(2022, 6, 4, 0, 0, 0, 0, time.UTC)
	for i := 0; i < rows; i++ {
		timestamps = append(timestamps, timestamp)
		values = append(values, float64(i))
		labels = append(labels, strconv.Itoa(i))
		timestamp = timestamp.Add(24 * time.Hour)
	}
	return data.New(
		data.Column{Name: "time", Values: timestamps},
		data.Column{Name: "values", Values: values},
		data.Column{Name: "", Values: values},
		data.Column{Name: "labels", Values: labels},
	)
}

func BenchmarkNew(b *testing.B) {
	var timestamps []time.Time
	var values []float64
	var labels []string
	timestamp := time.Date(2022, 6, 4, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 1000; i++ {
		timestamps = append(timestamps, timestamp)
		values = append(values, float64(i))
		labels = append(labels, timestamp.String())
		timestamp = timestamp.Add(24 * time.Hour)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = data.New(
			data.Column{Name: "time", Values: timestamps},
			data.Column{Name: "values", Values: values},
			data.Column{Name: "labels", Values: labels},
		)
	}
}
