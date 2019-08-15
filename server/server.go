package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tiket-libre/canary-router/version"

	"github.com/juju/errors"
	"github.com/juju/ratelimit"
	"github.com/tiket-libre/canary-router/instrumentation"

	canaryrouter "github.com/tiket-libre/canary-router"
	"github.com/tiket-libre/canary-router/config"
)

const infinityDuration time.Duration = 0x7fffffffffffffff

// Server holds necessary components as a proxy server
type Server struct {
	config       config.Config
	proxies      *canaryrouter.Proxy
	sidecarProxy *httputil.ReverseProxy
	canaryBucket *ratelimit.Bucket
}

// NewServer initiates a new proxy server
func NewServer(config config.Config) (*Server, error) {
	server := &Server{config: config}

	proxies, err := canaryrouter.BuildProxies(config.Client, config.MainTarget, config.CanaryTarget)
	if err != nil {
		return nil, errors.Trace(err)
	}
	server.proxies = proxies

	tr := &http.Transport{
		MaxIdleConns:       config.Client.MaxIdleConns,
		IdleConnTimeout:    time.Duration(config.Client.IdleConnTimeout) * time.Second,
		DisableCompression: config.Client.DisableCompression,
	}

	sidecarURL, err := url.Parse(server.config.SidecarURL)
	if err != nil {
		return nil, fmt.Errorf("Failed when creating proxy to sidecar: %v", err)
	}
	server.sidecarProxy = httputil.NewSingleHostReverseProxy(sidecarURL)
	server.sidecarProxy.Transport = tr

	if config.CircuitBreaker.RequestLimitCanary != 0 {
		server.canaryBucket = ratelimit.NewBucket(infinityDuration, int64(config.CircuitBreaker.RequestLimitCanary))
	}

	return server, nil
}

// Run initialize a new HTTP server
func (s *Server) Run() error {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", s.ServeHTTP)

	address := fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port)
	server := &http.Server{
		ReadTimeout:  time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.config.Server.IdleTimeout) * time.Second,
		Handler:      serveMux,
		Addr:         address,
	}

	log.Printf("Canary Router is now running on %s", address)

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
		defer func() {
			if r := recover(); r != nil {
				var err error
				switch t := r.(type) {
				case string:
					err = errors.New(t)
				case error:
					err = t
				default:
					msg := fmt.Sprintf("Unknown error: %v", r)
					err = errors.New(msg)
				}

				log.Printf("[Panic] Recovered in request handling: %v\nRequest payload: %v", r, req)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}()

		ctx := instrumentation.InitializeLatencyTracking(req.Context())
		req = req.WithContext(ctx)

		trimmedPath := strings.TrimPrefix(req.URL.Path, s.config.TrimPrefix)
		req.URL.Path = trimmedPath

		// NOTE: Override handlerFunc if X-Canary header is provided
		xCanaryVal := req.Header.Get("X-Canary")
		xCanary, err := convertToBool(xCanaryVal)
		if err == nil {
			req = setRoutingReason(req, "Routed via X-Canary header value: %s", xCanaryVal)
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
		recordMetricTarget(req.Context(), "main")
	}()

	s.proxies.Main.ServeHTTP(w, req)
}

func (s *Server) serveCanary(w http.ResponseWriter, req *http.Request) {
	defer func() {
		recordMetricTarget(req.Context(), "canary")
	}()

	s.proxies.Canary.ServeHTTP(w, req)
}

func (s *Server) callSidecar(req *http.Request) (int, error) {
	// Duplicate reader so that the original req.Body can still be used throughout
	// the request
	var bodyBuffer bytes.Buffer
	body := io.TeeReader(req.Body, &bodyBuffer)

	defer func() {
		req.Body = ioutil.NopCloser(&bodyBuffer)
	}()

	ctx := req.Context()
	outreq := req.WithContext(ctx)

	outBody, err := ioutil.ReadAll(body)
	if err != nil {
		return 0, err
	}
	outreq.Body = ioutil.NopCloser(bytes.NewReader(outBody))

	recorder := httptest.NewRecorder()
	s.sidecarProxy.ServeHTTP(recorder, outreq)

	return recorder.Code, nil
}

func (s *Server) viaProxyWithSidecar() http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		if s.IsCanaryLimited() && s.canaryBucket.Available() <= 0 {
			req = setRoutingReason(req, "Canary limit reached")

			s.serveMain(w, req)
			return
		}

		statusCode, err := s.callSidecar(req)
		if err != nil {
			req = setRoutingReason(req, err.Error())
			log.Print(fmt.Errorf("Error when calling sidecar: %v", err))

			s.serveMain(w, req)
			return
		}

		switch statusCode {
		case canaryrouter.StatusCodeMain:
			req = setRoutingReason(req, "Sidecar returns status code %d", statusCode)
			s.serveMain(w, req)
		case canaryrouter.StatusCodeCanary:
			if s.IsCanaryLimited() && s.canaryBucket.TakeAvailable(1) == 0 {
				req = setRoutingReason(req, "Sidecar returns status code %d, but canary limit reached", statusCode)
				s.serveMain(w, req)
			} else {
				req = setRoutingReason(req, "Sidecar returns status code %d", statusCode)
				s.serveCanary(w, req)
			}
		default:
			req = setRoutingReason(req, "Sidecar returns non standard status code %d", statusCode)
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

func setRoutingReason(req *http.Request, reason string, reasonArg ...interface{}) *http.Request {
	if len(reasonArg) > 0 {
		reason = fmt.Sprintf(reason, reasonArg...)
	}

	ctx, err := instrumentation.AddReasonTag(req.Context(), reason)
	if err != nil {
		log.Print(err)
		return req
	}

	return req.WithContext(ctx)
}

func recordMetricTarget(ctx context.Context, target string) {
	ctx, err := instrumentation.AddTargetTag(ctx, target)
	if err != nil {
		log.Println(err)
	}

	ctx, err = instrumentation.AddVersionTag(ctx, version.Info.String())
	if err != nil {
		log.Println(err)
	}

	instrumentation.RecordLatency(ctx)
}
