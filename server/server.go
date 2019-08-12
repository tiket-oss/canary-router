package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/juju/errors"
	"github.com/juju/ratelimit"
	"github.com/tiket-libre/canary-router/instrumentation"
	"github.com/tiket-libre/canary-router/sidecar"

	canaryrouter "github.com/tiket-libre/canary-router"
	"github.com/tiket-libre/canary-router/config"
)

const infinityDuration time.Duration = 0x7fffffffffffffff

// Server holds necessary components as a proxy server
type Server struct {
	config            config.Config
	proxies           *canaryrouter.Proxy
	sidecarHTTPClient *http.Client
	canaryBucket      *ratelimit.Bucket
}

// NewServer initiates a new proxy server
func NewServer(config config.Config) (*Server, error) {
	server := &Server{config: config}

	proxies, err := canaryrouter.BuildProxies(config.MainTarget, config.CanaryTarget)
	if err != nil {
		return nil, errors.Trace(err)
	}
	server.proxies = proxies

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	server.sidecarHTTPClient = &http.Client{
		Transport: tr,
		Timeout:   5 * time.Second,
	}

	if config.CircuitBreaker.RequestLimitCanary != 0 {
		server.canaryBucket = ratelimit.NewBucket(infinityDuration, int64(config.CircuitBreaker.RequestLimitCanary))
	}

	return server, nil
}

// Run initialize a new HTTP server
func (s *Server) Run() error {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", s.ServeHTTP)

	server := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      serveMux,
		Addr:         fmt.Sprintf(":%d", s.config.ListenPort),
	}

	return server.ListenAndServe()
}

// ServeHTTP handles incoming traffics via provided proxies
func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	s.viaProxy()(res, req)
}

// IsCanaryLimited checks if circuit breaker (canary request limiter) feature is enabled
func (s *Server) IsCanaryLimited() bool {
	return s.canaryBucket != nil
}

func (s *Server) viaProxy() http.HandlerFunc {
	var handlerFunc http.HandlerFunc

	if s.config.SidecarURL == "" {
		handlerFunc = s.proxies.Main.ServeHTTP
	} else {
		handlerFunc = s.viaProxyWithSidecar()
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
				s.proxies.Canary.ServeHTTP(w, req)
			} else {
				target = "main"
				s.proxies.Main.ServeHTTP(w, req)
			}
			return
		}

		handlerFunc(w, req)
	}
}

func (s *Server) viaProxyWithSidecar() http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {

		var target string
		defer func() {
			ctx, err := instrumentation.AddTargetTag(req.Context(), target)
			if err != nil {
				log.Print(err)
			}

			instrumentation.RecordLatency(ctx)
		}()

		if s.IsCanaryLimited() && s.canaryBucket.Available() <= 0 {
			s.proxies.Main.ServeHTTP(w, req)
			return
		}

		sidecarURL, err := url.ParseRequestURI(s.config.SidecarURL)
		if err != nil {
			log.Printf("Failed to parse sidecar URL %s: %+v", sidecarURL, err)
			s.proxies.Main.ServeHTTP(w, req)
			return
		}

		oriURL := req.URL
		oriBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Printf("Failed to read body ori req: %+v", err)
			s.proxies.Main.ServeHTTP(w, req)
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
			s.proxies.Main.ServeHTTP(w, req)
			return
		}

		resp, err := s.sidecarHTTPClient.Post(sidecarURL.String(), "application/json", buf)
		if err != nil {
			log.Printf("Failed to get resp from sidecar: %+v", err)
			s.proxies.Main.ServeHTTP(w, req)
			return
		}
		defer resp.Body.Close()

		req.URL = oriURL
		req.Body = ioutil.NopCloser(bytes.NewBuffer(oriBody))

		switch resp.StatusCode {
		case canaryrouter.StatusCodeMain:
			target = "main"
			s.proxies.Main.ServeHTTP(w, req)
		case canaryrouter.StatusCodeCanary:
			if s.IsCanaryLimited() && s.canaryBucket.TakeAvailable(1) == 0 {
				target = "main"
				s.proxies.Main.ServeHTTP(w, req)
			} else {
				target = "canary"
				s.proxies.Canary.ServeHTTP(w, req)
			}
		default:
			target = "main"
			s.proxies.Main.ServeHTTP(w, req)
		}
	}
}

func convertToBool(boolStr string) (bool, error) {
	if boolStr == "true" || boolStr == "false" {
		return strconv.ParseBool(boolStr)
	}

	return false, errors.New("neither 'true' nor 'false'")
}
