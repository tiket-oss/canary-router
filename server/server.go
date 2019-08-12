package server

import (
	"fmt"
	"net/http"
	"time"

	canaryrouter "github.com/tiket-libre/canary-router"
	"github.com/tiket-libre/canary-router/config"
	"github.com/tiket-libre/canary-router/handler"
)

// Run initialize a new HTTP server
func Run(config config.Config) error {

	proxies, err := canaryrouter.BuildProxies(config.MainTarget, config.CanaryTarget)
	if err != nil {
		return err
	}

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", handler.Index(config, proxies))

	server := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      serveMux,
		Addr:         fmt.Sprintf(":%d", config.ListenPort),
	}

	return server.ListenAndServe()
}
