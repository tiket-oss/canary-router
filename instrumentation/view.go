package instrumentation

import (
	"fmt"
	"log"
	"net/http"

	"github.com/juju/errors"

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
		Description: "The count of requests per path",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyTarget, KeyReason},
	}

	views = []*view.View{RequestCountView}
)

// Initialize register views and default Prometheus exporter
func Initialize(cfg config.InstrumentationConfig) error {

	if err := view.Register(views...); err != nil {
		return errors.Trace(err)
	}

	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "canary_router",
	})
	if err != nil {
		return errors.Trace(err)
	}

	view.RegisterExporter(pe)
	go func() {
		addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
		mux := http.NewServeMux()
		mux.Handle("/metrics", pe)
		log.Printf("Metrics endpoint will be running at: %s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("Failed to run Prometheus scrape endpoint: %v", errors.ErrorStack(err))
		}
	}()

	return nil
}
