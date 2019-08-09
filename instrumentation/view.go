package instrumentation

import (
	"fmt"
	"log"
	"net/http"

	"github.com/tiket-libre/canary-router/config"

	"contrib.go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	// RequestCountView provide View for request count grouped by target
	RequestCountView = &view.View{
		Name:        "request/count",
		Measure:     MLatencyMs,
		Description: "The count of requests per target",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyTarget},
	}

	// RequestLatencyView provide view for latency count distribution
	RequestLatencyView = &view.View{
		Name:        "request/latency",
		Measure:     MLatencyMs,
		Description: "The latency distribution per request target",

		// Latency in buckets:
		// [>=0ms, >=25ms, >=50ms, >=75ms, >=100ms, >=200ms, >=400ms, >=600ms, >=800ms, >=1s, >=2s, >=4s, >=6s]
		Aggregation: view.Distribution(0, 25, 50, 75, 100, 200, 400, 600, 800, 1000, 2000, 4000, 6000),
		TagKeys:     []tag.Key{KeyTarget},
	}

	views = []*view.View{RequestCountView, RequestLatencyView}
)

// Initialize register views and default Prometheus exporter
func Initialize(cfg config.InstrumentationConfig) error {

	if err := view.Register(views...); err != nil {
		return err
	}

	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "canary_router",
	})
	if err != nil {
		return err
	}

	view.RegisterExporter(pe)
	go func() {
		addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
		mux := http.NewServeMux()
		mux.Handle("/metrics", pe)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("Failed to run Prometheus scrape endpoint: %v", err)
		}
	}()

	return nil
}
