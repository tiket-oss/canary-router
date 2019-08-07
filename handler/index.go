package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	canaryrouter "github.com/tiket-libre/canary-router"
	"github.com/tiket-libre/canary-router/config"
	"github.com/tiket-libre/canary-router/instrumentation"
	"github.com/tiket-libre/canary-router/sidecar"
)

// Index returns a http.HandlerFunc which will route incoming traffics using provided proxies
func Index(config config.Config, proxies *canaryrouter.Proxy) http.HandlerFunc {

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}

	client := &http.Client{Transport: tr}

	return viaProxy(proxies, client, config.SidecarURL, config.CircuitBreaker.RequestLimitCanary)
}

func viaProxy(proxies *canaryrouter.Proxy, client *http.Client, sidecarURL string, requestLimitCanary uint64) http.HandlerFunc {

	var counterCanary uint64
	var handlerFunc http.HandlerFunc

	if sidecarURL == "" {
		handlerFunc = proxies.Main.ServeHTTP
	} else {
		handlerFunc = viaProxyWithSidecar(proxies, client, sidecarURL, requestLimitCanary, &counterCanary)
	}

	return func(w http.ResponseWriter, req *http.Request) {

		// NOTE: Override handlerFunc if X-Canary header is provided
		xCanaryVal := req.Header.Get("X-Canary")
		xCanary, err := convertToBool(xCanaryVal)
		if err == nil {
			if xCanary {
				proxies.Canary.ServeHTTP(w, req)
			} else {
				proxies.Main.ServeHTTP(w, req)
			}
			return
		}

		handlerFunc(w, req)
	}
}

func viaProxyWithSidecar(proxies *canaryrouter.Proxy, client *http.Client, sidecarURL string, limitCanary uint64, counterCanary *uint64) http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {

		requestRecord := instrumentation.NewRequestRecord()
		defer requestRecord.Register()

		//log.Printf(">>> GOT LIMIT. limitCanary: %d getCounter: %d", limitCanary, getCounter(&counterCanary))

		if limitCanary != 0 && (getCounter(counterCanary) > (limitCanary - 1)) {
			proxies.Main.ServeHTTP(w, req)
			return
		}

		sidecarURL, err := url.ParseRequestURI(sidecarURL)
		if err != nil {
			log.Printf("Failed to parse sidecar URL %s: %+v", sidecarURL, err)
			proxies.Main.ServeHTTP(w, req)
			return
		}

		oriUR := req.URL
		oriBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Printf("Failed to read body ori req: %+v", err)
			proxies.Main.ServeHTTP(w, req)
			return
		}

		originReq := sidecar.OriginRequest{
			Method: req.Method,
			URL:    req.URL.String(),
			Header: req.Header,
			Body:   string(oriBody),
		}

		buf := new(bytes.Buffer)
		err = json.NewEncoder(buf).Encode(originReq)
		if err != nil {
			log.Printf("Failed to encode json: %+v", err)
			proxies.Main.ServeHTTP(w, req)
			return
		}

		resp, err := client.Post(sidecarURL.String(), "application/json", buf)
		if err != nil {
			log.Printf("Failed to get resp from sidecar: %+v", err)
			proxies.Main.ServeHTTP(w, req)
			return
		}
		defer resp.Body.Close()

		req.URL = oriUR
		req.Body = ioutil.NopCloser(bytes.NewBuffer(oriBody))

		switch resp.StatusCode {
		case canaryrouter.StatusCodeMain:
			requestRecord.Target = "main"
			proxies.Main.ServeHTTP(w, req)
		case canaryrouter.StatusCodeCanary:
			incCounter(counterCanary)
			requestRecord.Target = "canary"
			proxies.Canary.ServeHTTP(w, req)
		default:
			requestRecord.Target = "main"
			proxies.Main.ServeHTTP(w, req)
		}
	}
}

func incCounter(counter *uint64) {
	atomic.AddUint64(counter, 1)
}

func getCounter(counter *uint64) uint64 {
	return atomic.LoadUint64(counter)
}

func convertToBool(boolStr string) (bool, error) {
	if boolStr == "true" || boolStr == "false" {
		return strconv.ParseBool(boolStr)
	}

	return false, errors.New("neither 'true' nor 'false'")
}
