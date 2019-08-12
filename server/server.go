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
		MaxIdleConns:       config.Client.MaxIdleConns,
		IdleConnTimeout:    time.Duration(config.Client.IdleConnTimeout) * time.Second,
		DisableCompression: config.Client.DisableCompression,
	}
	server.sidecarHTTPClient = &http.Client{
		Transport: tr,
		Timeout:   time.Duration(config.Client.Timeout) * time.Second,
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
		ReadTimeout:  time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.config.Server.IdleTimeout) * time.Second,
		Handler:      serveMux,
		Addr:         fmt.Sprintf(":%s", s.config.Server.ListenPort),
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
		handlerFunc = s.serveMain
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
			if xCanary {
				s.serveCanary(w, req)
			} else {
				s.serveMain(w, req)
			}
			return
		}

		handlerFunc(w, req)
	}
}

func (s *Server) serveMain(w http.ResponseWriter, req *http.Request) {
	defer func() {
		ctx, err := instrumentation.AddTargetTag(req.Context(), "main")
		if err != nil {
			log.Println(err)
		}

		instrumentation.RecordLatency(ctx)
	}()

	s.proxies.Main.ServeHTTP(w, req)
}

func (s *Server) serveCanary(w http.ResponseWriter, req *http.Request) {
	defer func() {
		ctx, err := instrumentation.AddTargetTag(req.Context(), "canary")
		if err != nil {
			log.Println(err)
		}

		instrumentation.RecordLatency(ctx)
	}()

	s.proxies.Canary.ServeHTTP(w, req)
}

func (s *Server) viaProxyWithSidecar() http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		if s.IsCanaryLimited() && s.canaryBucket.Available() <= 0 {
			s.serveMain(w, req)
			return
		}

		sidecarURL, err := url.ParseRequestURI(s.config.SidecarURL)
		if err != nil {
			log.Printf("Failed to parse sidecar URL %s: %+v", sidecarURL, err)
			s.serveMain(w, req)
			return
		}

		oriURL := req.URL
		oriBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Printf("Failed to read body ori req: %+v", err)
			s.serveMain(w, req)
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
			s.serveMain(w, req)
			return
		}

		resp, err := s.sidecarHTTPClient.Post(sidecarURL.String(), "application/json", buf)
		if err != nil {
			log.Printf("Failed to get resp from sidecar: %+v", err)
			s.serveMain(w, req)
			return
		}
		defer resp.Body.Close()

		req.URL = oriURL
		req.Body = ioutil.NopCloser(bytes.NewBuffer(oriBody))

		switch resp.StatusCode {
		case canaryrouter.StatusCodeMain:
			s.serveMain(w, req)
		case canaryrouter.StatusCodeCanary:
			if s.IsCanaryLimited() && s.canaryBucket.TakeAvailable(1) == 0 {
				s.serveMain(w, req)
			} else {
				s.serveCanary(w, req)
			}
		default:
			s.serveMain(w, req)
		}
	}
}

func convertToBool(boolStr string) (bool, error) {
	if boolStr == "true" || boolStr == "false" {
		return strconv.ParseBool(boolStr)
	}

	return false, errors.New("neither 'true' nor 'false'")
}
