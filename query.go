package grafana_json_server

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

// A Handler responds to a query request from the JSON API datasource.
type Handler interface {
	Query(ctx context.Context, target string, request QueryRequest) (QueryResponse, error)
}

// The HandlerFunc type is an adapter to allow the use of an ordinary function as Handler handlers.
// If f is a function with the appropriate signature, then HandlerFunc(f) is a Handler that calls f.
type HandlerFunc func(ctx context.Context, target string, request QueryRequest) (QueryResponse, error)

// Query calls f(ctx, target, request)
func (qf HandlerFunc) Query(ctx context.Context, target string, request QueryRequest) (QueryResponse, error) {
	return qf(ctx, target, request)
}

// The QueryRequest structure is the query request from Grafana to the data source.
type QueryRequest struct {
	App        string               `json:"app"`
	Timezone   string               `json:"timezone"`
	StartTime  int64                `json:"startTime"`
	Interval   string               `json:"interval"`
	IntervalMs int                  `json:"intervalMs"`
	PanelID    any                  `json:"panelId"`
	Targets    []QueryRequestTarget `json:"targets"`
	Range      Range                `json:"range"`
	RequestID  string               `json:"requestId"`
	RangeRaw   struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"rangeRaw"`
	ScopedVars    json.RawMessage `json:"scopedVars"`
	MaxDataPoints int             `json:"maxDataPoints"`
	LiveStreaming bool            `json:"liveStreaming"`
	AdhocFilters  []interface{}   `json:"adhocFilters"`
}

// QueryRequestTarget is one target in the QueryRequest structure. The main interesting fields as the Target, which is
// the Metric's name, and the Payload, which contains all selection options in any payload options. Use GetPayload to
// unmarshal the Payload into a Go structure.
type QueryRequestTarget struct {
	RefID      string `json:"refId"`
	Datasource struct {
		Type string `json:"type"`
		UID  string `json:"uid"`
	} `json:"datasource"`
	EditorMode string          `json:"editorMode"`
	Payload    json.RawMessage `json:"payload"`
	Target     string          `json:"target"`
	Key        string          `json:"key"`
	Type       string          `json:"type"` // TODO: is this really present?
}

// Range is the time range of the QueryRequest.
type Range struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
	Raw  struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"raw"`
}

// GetPayload unmarshals the target's raw payload into a provided payload.
func (r QueryRequest) GetPayload(target string, payload any) error {
	for _, t := range r.Targets {
		if t.Target == target {
			if t.Payload == nil {
				return errors.New("no payload found")
			}
			return json.Unmarshal(t.Payload, payload)
		}
	}
	return errors.New("target not found")
}

// A ScopedVar holds the value of a dashboard variable and is sent to the server as part of the QueryRequest.
type ScopedVar[T any] struct {
	Selected bool
	Text     string
	Value    T
}

// GetScopedVars unmarshals all variables in a QueryRequest into a Go structure. The vars variable should be struct
// of ScopedVar structs, matching the type of the variable. E.g. a multi-select variable should be represented by
// a ScopedVar[[]string].
func (r QueryRequest) GetScopedVars(vars any) error {
	return json.Unmarshal(r.ScopedVars, vars)
}

// QueryResponse is the output of the query function.  Both TimeSeriesResponse and TableResponse implement this interface.
type QueryResponse interface {
	json.Marshaler
}

var _ QueryResponse = TimeSeriesResponse{}

// TimeSeriesResponse is the response to a query as a time series. Target should match the Target of the received request.
type TimeSeriesResponse struct {
	Target     string      `json:"target"`
	DataPoints []DataPoint `json:"datapoints"`
}

// MarshalJSON converts a TimeSeriesResponse to JSON.
func (r TimeSeriesResponse) MarshalJSON() ([]byte, error) {
	type r2 TimeSeriesResponse
	v2 := r2(r)
	return json.Marshal(v2)
}

// DataPoint contains one entry of a TimeSeriesResponse.
type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

// MarshalJSON converts a DataPoint to JSON.
func (d DataPoint) MarshalJSON() ([]byte, error) {
	return []byte(`[` +
			strconv.FormatFloat(d.Value, 'f', -1, 64) + `,` +
			strconv.FormatInt(d.Timestamp.UnixMilli(), 10) +
			`]`),
		nil
}

var _ QueryResponse = TableResponse{}

// TableResponse is returned by a table query, i.e. a slice of Column structures.
type TableResponse struct {
	Columns []Column
}

// Column is a column returned by a table query.  Text holds the column's header,
// Data holds the slice of values and should be a TimeColumn, a StringColumn
// or a NumberColumn.
type Column struct {
	Text string
	Data interface{}
}

// TimeColumn holds a slice of time.Time values (one per row).
type TimeColumn []time.Time

// StringColumn holds a slice of string values (one per row).
type StringColumn []string

// NumberColumn holds a slice of float64 values (one per row).
type NumberColumn []float64

type tableResponse struct {
	Type    string                `json:"type"`
	Columns []tableResponseColumn `json:"columns"`
	Rows    []tableResponseRow    `json:"rows"`
}

type tableResponseColumn struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type tableResponseRow []interface{}

// MarshalJSON converts a TableResponse to JSON.
func (t TableResponse) MarshalJSON() (output []byte, err error) {
	var colTypes []string
	var rowCount int

	if colTypes, rowCount, err = t.getColumnDetails(); err == nil {
		output, err = json.Marshal(tableResponse{
			Type:    "table",
			Columns: t.buildColumns(colTypes),
			Rows:    t.buildRows(rowCount),
		})
	}

	return output, err
}

func (t TableResponse) getColumnDetails() (colTypes []string, rowCount int, err error) {
	for _, entry := range t.Columns {
		var dataCount int
		switch data := entry.Data.(type) {
		case TimeColumn:
			colTypes = append(colTypes, "time")
			dataCount = len(data)
		case StringColumn:
			colTypes = append(colTypes, "string")
			dataCount = len(data)
		case NumberColumn:
			colTypes = append(colTypes, "number")
			dataCount = len(data)
		}

		if rowCount == 0 {
			rowCount = dataCount
		}

		if dataCount != rowCount {
			return colTypes, rowCount, errors.New("error building table query output: all columns must have the same number of rows")
		}
	}
	return
}

func (t TableResponse) buildColumns(colTypes []string) []tableResponseColumn {
	columns := make([]tableResponseColumn, len(colTypes))
	for index, colType := range colTypes {
		columns[index] = tableResponseColumn{
			Text: t.Columns[index].Text,
			Type: colType,
		}
	}
	return columns
}

func (t TableResponse) buildRows(rowCount int) []tableResponseRow {
	rows := make([]tableResponseRow, rowCount)
	for row := 0; row < rowCount; row++ {
		newRow := make(tableResponseRow, len(t.Columns))
		for column, entry := range t.Columns {
			switch data := entry.Data.(type) {
			case TimeColumn:
				newRow[column] = data[row]
			case StringColumn:
				newRow[column] = data[row]
			case NumberColumn:
				newRow[column] = data[row]
			}
		}
		rows[row] = newRow
	}
	return rows
}
