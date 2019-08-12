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
		Name:        "router/count",
		Measure:     MLatencyMs,
		Description: "The count of requests per path",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyTarget, KeyReason},
	}

	views = []*view.View{RequestCountView}
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
