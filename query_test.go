package grafana_json_server_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	grafanaJSONServer "github.com/clambin/grafana-json-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "update .golden files")

func TestQueryRequest_Unmarshal(t *testing.T) {
	input := `{
	"maxDataPoints": 100,
	"interval": "1h",
	"range": {
		"from": "2020-01-01T00:00:00.000Z",
		"to": "2020-12-31T00:00:00.000Z"
	},
	"targets": [
		{ "target": "A", "type": "dataserie" },
		{ "target": "B", "type": "table" }
	]
}`

	var output grafanaJSONServer.QueryRequest

	err := json.Unmarshal([]byte(input), &output)
	require.NoError(t, err)
	assert.Equal(t, 100, output.MaxDataPoints)
	// assert.Equal(t, server.QueryRequestDuration(1*time.Hour), output.Interval)
	// assert.Equal(t, 1*time.Hour, time.duration(output.Interval))
	assert.Equal(t, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), output.Range.From)
	assert.Equal(t, time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC), output.Range.To)
	require.Len(t, output.Targets, 2)
	assert.Equal(t, "A", output.Targets[0].Target)
	assert.Equal(t, "dataserie", output.Targets[0].Type)
	assert.Equal(t, "B", output.Targets[1].Target)
	assert.Equal(t, "table", output.Targets[1].Type)
}

func TestQueryResponse_Marshal(t *testing.T) {
	tests := []struct {
		name    string
		payload grafanaJSONServer.QueryResponse
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "timeseries",
			payload: grafanaJSONServer.TimeSeriesResponse{
				Target: "A",
				DataPoints: []grafanaJSONServer.DataPoint{
					{Value: 100, Timestamp: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
					{Value: 101, Timestamp: time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC)},
					{Value: 102, Timestamp: time.Date(2020, 1, 1, 2, 0, 0, 0, time.UTC)},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "table",
			payload: grafanaJSONServer.TableResponse{
				Columns: []grafanaJSONServer.Column{
					{Text: "Time", Data: grafanaJSONServer.TimeColumn{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2020, 1, 1, 0, 1, 0, 0, time.UTC)}},
					{Text: "Label", Data: grafanaJSONServer.StringColumn{"foo", "bar"}},
					{Text: "Series A", Data: grafanaJSONServer.NumberColumn{42, 43}},
					{Text: "Series B", Data: grafanaJSONServer.NumberColumn{64.5, 100.0}},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name:    "combined",
			payload: makeCombinedQueryResponse(),
			wantErr: assert.NoError,
		},
		{
			name: "invalid",
			payload: grafanaJSONServer.TableResponse{
				Columns: []grafanaJSONServer.Column{
					{Text: "Time", Data: grafanaJSONServer.TimeColumn{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2020, 1, 1, 0, 1, 0, 0, time.UTC)}},
					{Text: "Label", Data: grafanaJSONServer.StringColumn{"foo"}},
					{Text: "Series A", Data: grafanaJSONServer.NumberColumn{42, 43}},
					{Text: "Series B", Data: grafanaJSONServer.NumberColumn{64.5, 100.0, 105.0}},
				},
			},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			w := bufio.NewWriter(&b)
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			err := enc.Encode(tt.payload)

			tt.wantErr(t, err)

			if err != nil {
				return
			}
			_ = w.Flush()

			gp := filepath.Join("testdata", strings.ToLower(t.Name())+".golden")
			if *update {
				t.Logf("updating golden file for %s", t.Name())
				dirname := filepath.Dir(gp)
				require.NoError(t, os.MkdirAll(dirname, 0755))
				err = os.WriteFile(gp, b.Bytes(), 0644)
				require.NoError(t, err, "failed to update golden file")
			}

			var golden []byte
			golden, err = os.ReadFile(gp)
			require.NoError(t, err)

			assert.Equal(t, string(golden), b.String())

		})
	}
}

type combinedResponse struct {
	responses []interface{}
}

func (r combinedResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.responses)
}

func makeCombinedQueryResponse() combinedResponse {
	testDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	dataseries := []grafanaJSONServer.TimeSeriesResponse{{
		Target: "A",
		DataPoints: []grafanaJSONServer.DataPoint{
			{Value: 100, Timestamp: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
			{Value: 101, Timestamp: time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC)},
			{Value: 102, Timestamp: time.Date(2020, 1, 1, 2, 0, 0, 0, time.UTC)},
		},
	}}

	tables := []grafanaJSONServer.TableResponse{{
		Columns: []grafanaJSONServer.Column{
			{Text: "Time", Data: grafanaJSONServer.TimeColumn{testDate, testDate}},
			{Text: "Label", Data: grafanaJSONServer.StringColumn{"foo", "bar"}},
			{Text: "Series A", Data: grafanaJSONServer.NumberColumn{42, 43}},
			{Text: "Series B", Data: grafanaJSONServer.NumberColumn{64.5, 100.0}},
		},
	}}

	var r combinedResponse
	//r.responses = make([]interface{}, 0)
	for _, dataserie := range dataseries {
		r.responses = append(r.responses, dataserie)
	}
	for _, table := range tables {
		r.responses = append(r.responses, table)
	}

	return r
}

func BenchmarkTimeSeriesResponse_MarshalJSON(b *testing.B) {
	response := buildTimeSeriesResponse(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := response.MarshalJSON(); err != nil {
			b.Fatal(err)
		}
	}
}

func buildTimeSeriesResponse(count int) grafanaJSONServer.TimeSeriesResponse {
	var datapoints []grafanaJSONServer.DataPoint
	timestamp := time.Date(2022, time.November, 27, 0, 0, 0, 0, time.UTC)
	for i := 0; i < count; i++ {
		datapoints = append(datapoints, grafanaJSONServer.DataPoint{
			Timestamp: timestamp,
			Value:     float64(i),
		})
	}
	return grafanaJSONServer.TimeSeriesResponse{Target: "foo", DataPoints: datapoints}
}

func BenchmarkTableResponse_MarshalJSON(b *testing.B) {
	response := buildTableResponse(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := response.MarshalJSON(); err != nil {
			b.Fatal(err)
		}
	}
}

func buildTableResponse(count int) grafanaJSONServer.TableResponse {
	var timestamps []time.Time
	var values []float64

	timestamp := time.Date(2022, time.November, 27, 0, 0, 0, 0, time.UTC)
	for i := 0; i < count; i++ {
		timestamps = append(timestamps, timestamp)
		values = append(values, 1.0)
		timestamp = timestamp.Add(time.Minute)
	}
	return grafanaJSONServer.TableResponse{Columns: []grafanaJSONServer.Column{
		{Text: "time", Data: grafanaJSONServer.TimeColumn(timestamps)},
		{Text: "value", Data: grafanaJSONServer.NumberColumn(values)},
	}}
}

func TestQueryRequest_GetPayload(t *testing.T) {
	req := grafanaJSONServer.QueryRequest{Targets: []grafanaJSONServer.QueryRequestTarget{
		{Target: "valid", Payload: json.RawMessage(`{ "bar": "snafu" }`)},
		{Target: "empty", Payload: nil},
	}}

	testCases := []struct {
		name    string
		target  string
		wantErr assert.ErrorAssertionFunc
		want    string
	}{
		{
			name:    "valid",
			target:  "valid",
			wantErr: assert.NoError,
			want:    "snafu",
		},
		{
			name:    "empty",
			target:  "empty",
			wantErr: assert.Error,
		},
		{
			name:    "missing",
			target:  "missing",
			wantErr: assert.Error,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			var payload struct {
				Bar string
			}

			tt.wantErr(t, req.GetPayload(tt.target, &payload))
			if tt.want != "" {
				assert.Equal(t, tt.want, payload.Bar)
			}
		})
	}
}

func TestQueryRequest_GetScopedVars(t *testing.T) {
	req := grafanaJSONServer.QueryRequest{
		ScopedVars: json.RawMessage(`
{
	"var1": { "selected": false, "text": "snafu", "value": "snafu" },
	"query0": { "selected": false, "text": "All", "value": ["Value1","Value2","Value3"] }
}`)}

	var scopedVars struct {
		Var1   grafanaJSONServer.ScopedVar[string]
		Query0 grafanaJSONServer.ScopedVar[[]string]
	}

	assert.NoError(t, req.GetScopedVars(&scopedVars))
	assert.Equal(t, "snafu", scopedVars.Var1.Value)
	assert.Equal(t, []string{"Value1", "Value2", "Value3"}, scopedVars.Query0.Value)
}
