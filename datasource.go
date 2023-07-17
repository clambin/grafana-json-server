package grafana_json_server

import "context"

// A DataSource represents one source of Grafana data.
//
// The Metric describes the attributes of the data, i.e. name and any possible payload options.  If the payload is a select,
// or multi-select type, but no options are provided in the metric, then MetricPayloadOptionFunc must also be provided.
//
// Query implements the query to generate data. It's either a type that implements the Query interface,  or a function
// converted to one using the QueryFunc adapter.
type DataSource struct {
	Metric
	MetricPayloadOptionFunc
	Query
}

type Query interface {
	Query(ctx context.Context, target string, request QueryRequest) (QueryResponse, error)
}

// The QueryFunc type is an adapter to allow the use of ordinary functions as a DataSource's Query.
type QueryFunc func(ctx context.Context, target string, request QueryRequest) (QueryResponse, error)

// Query calls qf(ctx, target, request)
func (qf QueryFunc) Query(ctx context.Context, target string, request QueryRequest) (QueryResponse, error) {
	return qf(ctx, target, request)
}
