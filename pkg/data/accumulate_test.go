package data_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestTable_Accumulate(t *testing.T) {
	input := createTable(10)

	d := input.Accumulate()

	require.Len(t, d.Frame.Fields, 4)
	require.Equal(t, input.GetTimestamps(), d.GetTimestamps())
	assert.Equal(t, time.Date(2022, time.June, 4, 0, 0, 0, 0, time.UTC), d.GetTimestamps()[0])
	assert.Equal(t, time.Date(2022, time.June, 13, 0, 0, 0, 0, time.UTC), d.GetTimestamps()[9])

	assert.Equal(t, input.Frame.Fields[0].Len(), d.Frame.Fields[0].Len())
	assert.Equal(t, input.Frame.Fields[1].Len(), d.Frame.Fields[1].Len())

	assert.Equal(t, input.GetTimestamps(), d.GetTimestamps())
	values, ok := d.GetValues("values")
	require.True(t, ok)
	assert.Equal(t, []interface{}{0.0, 1.0, 3.0, 6.0, 10.0, 15.0, 21.0, 28.0, 36.0, 45.0}, values)
}
