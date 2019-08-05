package instrumentation

import (
	"log"
	"net/http"

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
		TagKeys:     []tag.Key{KeyTarget},
	}

	views = []*view.View{RequestCountView}
)

// Initialize register views and default Prometheus exporter
func Initialize() error {
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "canaryrouter",
	})
	if err != nil {
		return err
	}

	view.RegisterExporter(pe)
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", pe)
		if err := http.ListenAndServe(":8888", mux); err != nil {
			log.Fatalf("Failed to run Prometheus scrape endpoint: %v", err)
		}
	}()

	return view.Register(views...)
}
