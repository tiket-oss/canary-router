package server

import (
	"fmt"
	"net/http"

	"github.com/tiket-libre/canary-router"
	"github.com/tiket-libre/canary-router/config"
	"github.com/tiket-libre/canary-router/handler"
)

func Run(config config.Config) error {

	proxies, err := canaryrouter.BuildProxies(config.MainTarget, config.CanaryTarget)
	if err != nil {
		return err
	}

	http.HandleFunc("/", handler.Index(config, proxies))

	return http.ListenAndServe(fmt.Sprintf(":%d", config.ListenPort), nil)
}
