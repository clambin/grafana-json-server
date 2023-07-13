package data_test

import (
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"github.com/clambin/grafana-json-server/pkg/data"
	grafanaData "github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTable_FilterByTime(t *testing.T) {
	d := createTable(10)

	output := d.Filter(grafanaJSONServer.Range{
		From: time.Date(2022, 6, 5, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2022, 6, 7, 0, 0, 0, 0, time.UTC),
	})
	assert.Equal(t, []time.Time{
		time.Date(2022, time.June, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 6, 0, 0, 0, 0, time.UTC),
		time.Date(2022, time.June, 7, 0, 0, 0, 0, time.UTC),
	}, output.GetTimestamps())
}

func TestTable_FilterByTime_Empty(t *testing.T) {
	table := data.Table{Frame: grafanaData.NewFrame("bad")}

	f := table.Filter(grafanaJSONServer.Range{
		From: time.Date(2022, 6, 5, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2022, 6, 7, 0, 0, 0, 0, time.UTC),
	})
	assert.NotNil(t, f.Frame)
}

func BenchmarkFilter(b *testing.B) {
	var timestamps []time.Time
	var values []float64
	var labels []string
	timestamp := time.Date(2022, 6, 4, 0, 0, 0, 0, time.UTC)
	start := timestamp
	for i := 0; i < 1000; i++ {
		timestamps = append(timestamps, timestamp)
		values = append(values, float64(i))
		labels = append(labels, timestamp.String())
		timestamp = timestamp.Add(24 * time.Hour)
	}
	d := data.New(
		data.Column{Name: "time", Values: timestamps},
		data.Column{Name: "values", Values: values},
		data.Column{Name: "labels", Values: labels},
	)
	args := grafanaJSONServer.Range{
		From: start,
		To:   timestamp,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.Filter(args)
	}
}
