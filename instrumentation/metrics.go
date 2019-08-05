package instrumentation

import (
	"context"
	"log"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

var (
	// MLatencyMs records the time it took for request to be served (routed to proxy)
	MLatencyMs = stats.Float64("router/latency", "Number of requests routed", "ms")

	// KeyTarget holds target information of the request being routed. It will be either "canary" or "main"
	KeyTarget, _ = tag.NewKey("target")
)

// RequestRecord holds additional information to record request related metrics
type RequestRecord struct {
	Target    string
	StartTIme time.Time
}

// NewRequestRecord initialize a RequestRecord struct with startTime
// set as time.Now()
func NewRequestRecord() *RequestRecord {
	return &RequestRecord{
		StartTIme: time.Now(),
		Target:    "main",
	}
}

func sinceInMilliseconds(startTime time.Time) float64 {
	return float64(time.Since(startTime).Nanoseconds()) / 1e6
}

// Register add a new Measurement to Metrics by the RequestRecord field values
func (r *RequestRecord) Register() {
	ctx, err := tag.New(context.Background(), tag.Insert(KeyTarget, r.Target))
	if err != nil {
		log.Print(err)
	}

	stats.Record(ctx, MLatencyMs.M(sinceInMilliseconds(r.StartTIme)))
}
