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

	"github.com/juju/ratelimit"
	canaryrouter "github.com/tiket-libre/canary-router"
	"github.com/tiket-libre/canary-router/config"
	"github.com/tiket-libre/canary-router/instrumentation"
	"github.com/tiket-libre/canary-router/sidecar"
)

const infinityDuration time.Duration = 0x7fffffffffffffff

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

	var isCanaryLimited bool
	var canaryBucket *ratelimit.Bucket
	if requestLimitCanary != 0 {
		canaryBucket = ratelimit.NewBucket(infinityDuration, int64(requestLimitCanary))
		isCanaryLimited = true
	}

	var handlerFunc http.HandlerFunc

	if sidecarURL == "" {
		handlerFunc = proxies.Main.ServeHTTP
	} else {
		handlerFunc = viaProxyWithSidecar(proxies, client, sidecarURL, isCanaryLimited, canaryBucket)
	}

	return func(w http.ResponseWriter, req *http.Request) {
		ctx := instrumentation.InitializeLatencyTracking(req.Context())
		req = req.WithContext(ctx)

		// NOTE: Override handlerFunc if X-Canary header is provided
		xCanaryVal := req.Header.Get("X-Canary")
		xCanary, err := convertToBool(xCanaryVal)
		if err == nil {
			var target string
			defer func() {
				ctx, err := instrumentation.AddTargetTag(req.Context(), target)
				if err != nil {
					log.Println(err)
				}
				instrumentation.RecordLatency(ctx)
			}()

			if xCanary {
				target = "canary"
				proxies.Canary.ServeHTTP(w, req)
			} else {
				target = "main"
				proxies.Main.ServeHTTP(w, req)
			}
			return
		}

		handlerFunc(w, req)
	}
}

func viaProxyWithSidecar(proxies *canaryrouter.Proxy, client *http.Client, sidecarURL string, isCanaryLimited bool, canaryBucket *ratelimit.Bucket) http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {

		var target string
		defer func() {
			ctx, err := instrumentation.AddTargetTag(req.Context(), target)
			if err != nil {
				log.Print(err)
			}

			instrumentation.RecordLatency(ctx)
		}()

		if isCanaryLimited && canaryBucket.Available() <= 0 {
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
			target = "main"
			proxies.Main.ServeHTTP(w, req)
		case canaryrouter.StatusCodeCanary:
			if isCanaryLimited && canaryBucket.TakeAvailable(1) == 0 {
				target = "main"
				proxies.Main.ServeHTTP(w, req)
			} else {
				target = "canary"
				proxies.Canary.ServeHTTP(w, req)
			}
		default:
			target = "main"
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
