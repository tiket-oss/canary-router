package server

import (
	canaryrouter "canary-router"
	"canary-router/config"
	"canary-router/handler"
	"fmt"
	"net/http"
)

func Run(config config.Config) error {

	proxies, err := canaryrouter.BuildProxies(config)
	if err != nil {
		return err
	}

	http.HandleFunc("/", handler.Index(config, proxies))

	return http.ListenAndServe(fmt.Sprintf(":%d", config.ListenPort), nil)
}
