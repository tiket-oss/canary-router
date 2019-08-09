package instrumentation

import (
	"context"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

type contextKey string

var startTimeKey = contextKey("startTime")

var (
	// MLatencyMs records the time it took for request to be served (routed to proxy)
	MLatencyMs = stats.Float64("request/latency", "Latency of request served", "ms")

	// KeyTarget holds target information of the request being routed. It will be either "canary" or "main"
	KeyTarget, _ = tag.NewKey("target")
)

func sinceInMilliseconds(startTime time.Time) float64 {
	return float64(time.Since(startTime).Nanoseconds()) / 1e6
}

// InitializeLatencyTracking ...
func InitializeLatencyTracking(ctx context.Context) context.Context {
	return context.WithValue(ctx, startTimeKey, time.Now())
}

// RecordLatency ...
func RecordLatency(ctx context.Context) {
	startTimeVal := ctx.Value(startTimeKey)
	if startTime, ok := startTimeVal.(time.Time); ok {
		stats.Record(ctx, MLatencyMs.M(sinceInMilliseconds(startTime)))
	}
}

// AddTargetTag ...
func AddTargetTag(ctx context.Context, target string) (context.Context, error) {
	return tag.New(ctx, tag.Upsert(KeyTarget, target))
}
