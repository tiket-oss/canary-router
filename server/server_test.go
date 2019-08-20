package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/juju/errors"

	"github.com/tiket-libre/canary-router/config"

	canaryrouter "github.com/tiket-libre/canary-router"
)

type restRequest struct {
	httpHeader http.Header
	httpMethod string
	targetURL string
	bodyPayload string
}

//resp, gotBody := restClientCall(t, thisRouter.Client(), http.MethodPost, thisRouter.URL+"/foo/bar", map[string]string{}, fmt.Sprintf("%d", i))

type circuitBreakerParam struct {
	RequestLimitCanary uint64
	ErrorLimitCanary   uint64
}

func Test_Server_integration(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	emptyBodyBytes := []byte("")
	noCircuitBreakerParam := circuitBreakerParam{}

	backendMainBody := "Hello, I'm Main!"
	backendMain, _ := setupServer(t, []byte(backendMainBody), http.StatusOK, func(r *http.Request) {})
	defer backendMain.Close()

	backendCanaryBody := "Hello, I'm Canary!"
	backendCanary, _ := setupServer(t, []byte(backendCanaryBody), http.StatusOK, func(r *http.Request) {})
	defer backendCanary.Close()

	t.Run("[Given] No sideCarURL provided [then] default to Main", func(t *testing.T) {
		thisRouter := httptest.NewServer(setupThisRouterServer(t, backendMain.URL, backendCanary.URL, "", noCircuitBreakerParam))
		defer thisRouter.Close()

		restRequest := restRequest{httpHeader: http.Header{}, httpMethod:  http.MethodPost, targetURL:   thisRouter.URL+"/foo/bar", bodyPayload: "foo bar body",}
		_, gotBody := restClientCall(t, thisRouter.Client(), restRequest)
		if string(gotBody) != backendMainBody {
			t.Errorf("Not forwarded to Main. Gotbody: %s", string(gotBody))
		}

	})

	t.Run("[Given] SideCarURL (always to Main) and X-Canary=true [then] forward to Canary because X-Canary have higher precedence", func(t *testing.T) {
		sideCarToMain, sideCarToMainURL := setupServer(t, emptyBodyBytes, canaryrouter.StatusCodeMain, func(r *http.Request) {})
		defer sideCarToMain.Close()

		thisRouter := httptest.NewServer(setupThisRouterServer(t, backendMain.URL, backendCanary.URL, sideCarToMainURL.String(), noCircuitBreakerParam))
		defer thisRouter.Close()

		restRequest := restRequest{httpHeader: http.Header{"X-Canary": {"true"}}, httpMethod:  http.MethodPost, targetURL:   thisRouter.URL+"/foo/bar", bodyPayload: "foo bar body",}
		_, gotBody := restClientCall(t, thisRouter.Client(), restRequest)
		if string(gotBody) != backendCanaryBody {
			t.Errorf("Not forwarded to Canary. Gotbody: %s", string(gotBody))
		}
	})

	t.Run("[Given] SideCarURL (always to Main) and X-Canary header (with bad value) [then] forward to endpoint decided by sideCar (Main)", func(t *testing.T) {
		sideCarToMain, sideCarToMainURL := setupServer(t, emptyBodyBytes, canaryrouter.StatusCodeMain, func(r *http.Request) {})
		defer sideCarToMain.Close()

		thisRouter := httptest.NewServer(setupThisRouterServer(t, backendMain.URL, backendCanary.URL, sideCarToMainURL.String(), noCircuitBreakerParam))
		defer thisRouter.Close()

		restRequest := restRequest{httpHeader: http.Header{"X-Canary": {"NOTVALID"}}, httpMethod:  http.MethodPost, targetURL:   thisRouter.URL+"/foo/bar", bodyPayload: "foo bar body",}
		_, gotBody := restClientCall(t, thisRouter.Client(), restRequest)
		if string(gotBody) != backendMainBody {
			t.Errorf("Not forwarded to Main. Gotbody: %s", string(gotBody))
		}
	})

	t.Run("[Given] SideCarURL (always to Canary) and X-Canary header (with bad value) [then] forward to endpoint decided by sideCar (Canary)", func(t *testing.T) {
		sideCarToCanary, sideCarToCanaryURL := setupServer(t, emptyBodyBytes, canaryrouter.StatusCodeCanary, func(r *http.Request) {})
		defer sideCarToCanary.Close()

		thisRouter := httptest.NewServer(setupThisRouterServer(t, backendMain.URL, backendCanary.URL, sideCarToCanaryURL.String(), noCircuitBreakerParam))
		defer thisRouter.Close()

		restRequest := restRequest{httpHeader: http.Header{"X-Canary": {"NOTVALID"}}, httpMethod:  http.MethodPost, targetURL:   thisRouter.URL+"/foo/bar", bodyPayload: "foo bar body",}
		_, gotBody := restClientCall(t, thisRouter.Client(), restRequest)
		if string(gotBody) != backendCanaryBody {
			t.Errorf("Not forwarded to Canary. Gotbody: %s", string(gotBody))
		}
	})

	t.Run("Test X-Canary header", func(t *testing.T) {
		testCases := []struct {
			name       string
			argXCanary string
			wantBody   string
		}{
			{name: "X-Canary:true", argXCanary: "true", wantBody: backendCanaryBody},
			{name: "X-Canary:false", argXCanary: "false", wantBody: backendMainBody},

			{name: "Notvalid X-Canary:1", argXCanary: "1", wantBody: backendMainBody},
			{name: "Notvalid X-Canary:0", argXCanary: "0", wantBody: backendMainBody},
			{name: "Notvalid X-Canary:TRUE", argXCanary: "TRUE", wantBody: backendMainBody},
			{name: "Notvalid X-Canary:FALSE", argXCanary: "FALSE", wantBody: backendMainBody},
		}

		for _, tc := range testCases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				thisRouter := httptest.NewServer(setupThisRouterServer(t, backendMain.URL, backendCanary.URL, "", noCircuitBreakerParam))
				defer thisRouter.Close()

				restRequest := restRequest{httpHeader: http.Header{"X-Canary": {tc.argXCanary}}, httpMethod:  http.MethodPost, targetURL:   thisRouter.URL+"/foo/bar", bodyPayload: "foo bar body",}
				_, gotBody := restClientCall(t, thisRouter.Client(), restRequest)
				if string(gotBody) != tc.wantBody {
					t.Errorf("X-Canary:%s Gotbody: '%s' Wantbody: '%s'", tc.argXCanary, string(gotBody), tc.wantBody)
				}
			})
		}
	})

	t.Run("Test supported HTTP methods", func(t *testing.T) {
		testCases := []struct {
			name          string
			argStatusCode int
			wantBody      []byte
		}{{
			name:          "forward to Main",
			argStatusCode: canaryrouter.StatusCodeMain,
			wantBody:      []byte(backendMainBody),
		}, {
			name:          "forward to Canary",
			argStatusCode: canaryrouter.StatusCodeCanary,
			wantBody:      []byte(backendCanaryBody),
		}}

		methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

		for _, tc := range testCases {
			tc := tc
			t.Run(fmt.Sprintf("%d %s", tc.argStatusCode, tc.name), func(t *testing.T) {
				//t.Parallel()

				bodyResults := make(map[string]string)

				backendSidecar, backendSidecarURL := setupServer(t, emptyBodyBytes, tc.argStatusCode, func(r *http.Request) {
					byt, err := ioutil.ReadAll(r.Body)
					if err != nil {
						t.Fatal(err)
					}
					bodyResults[r.Method] = string(byt)
				})
				defer backendSidecar.Close()

				thisRouter := httptest.NewServer(setupThisRouterServer(t, backendMain.URL, backendCanary.URL, backendSidecarURL.String(), noCircuitBreakerParam))
				defer thisRouter.Close()

				originBodyContent := "This is DUMMY body"

				for _, httpMethod := range methods {
					t.Run(httpMethod, func(t *testing.T) {
						//t.Parallel()

						restRequest := restRequest{httpHeader: http.Header{}, httpMethod: httpMethod, targetURL:   thisRouter.URL+"/foo/bar", bodyPayload: originBodyContent,}
						_, gotBody := restClientCall(t, thisRouter.Client(), restRequest)

						if string(gotBody) != string(tc.wantBody) {
							t.Errorf("argStatusCode = %d got = %+v; want = %+v", tc.argStatusCode, gotBody, tc.wantBody)
							t.Errorf("(STR) argStatusCode = %d got = %+v; want = %+v", tc.argStatusCode, string(gotBody), string(tc.wantBody))
						}

					})
				}

				for _, gotBody := range bodyResults {
					if gotBody != originBodyContent {
						t.Errorf("Got ori body content: %s Want: %s", gotBody, originBodyContent)
					}
				}

			})
		}
	})

	t.Run("circuitbreaker", func(t *testing.T) {
		t.Run("request-limit-canary", func(t *testing.T) {
			canaryRequestLimit := uint64(45)

			sideCarServerFiftyFifty := setupCanaryServerFiftyFifty(t)
			defer sideCarServerFiftyFifty.Close()

			thisRouter := httptest.NewServer(setupThisRouterServer(t, backendMain.URL, backendCanary.URL, sideCarServerFiftyFifty.URL, circuitBreakerParam{RequestLimitCanary: canaryRequestLimit}))
			defer thisRouter.Close()

			gotCanaryCount, gotMainCount := 0, 0

			chanMainHit := make(chan int, 10)
			chanCanaryHit := make(chan int, 10)

			totalRequest := 100
			for i := 1; i <= totalRequest; i++ {
				i := i
				go func(chanMainHit chan int, chanCanaryHit chan int) {

					restRequest := restRequest{httpHeader: http.Header{}, httpMethod:  http.MethodPost, targetURL:   thisRouter.URL+"/foo/bar", bodyPayload: fmt.Sprintf("%d", i),}
					resp, gotBody := restClientCall(t, thisRouter.Client(), restRequest)
					defer resp.Body.Close()
					switch string(gotBody) {
					case backendMainBody:
						chanMainHit <- 1
					case backendCanaryBody:
						chanCanaryHit <- 1
					default:
						t.Errorf("Not supposed to be other content")
					}
				}(chanMainHit, chanCanaryHit)

			}

			for i := 0; i < totalRequest; i++ {
				select {
				case mainHit := <-chanMainHit:
					gotMainCount += mainHit
				case canaryHit := <-chanCanaryHit:
					gotCanaryCount += canaryHit
				}
			}

			close(chanMainHit)
			close(chanCanaryHit)

			if (uint64(gotCanaryCount) != canaryRequestLimit) || (gotMainCount != (totalRequest - gotCanaryCount)) {
				t.Errorf("gotCanaryCount:%d gotMainCount:%d canaryRequestLimit:%d totalRequest:%d", gotCanaryCount, gotMainCount, canaryRequestLimit, totalRequest)
			}
		})

		t.Run("error-limit-canary", func(t *testing.T) {
			// TODO: error-limit-canary integration test

		})
	})
}

func setupServer(t *testing.T, bodyResp []byte, statusCode int, middleFunc func(r *http.Request)) (*httptest.Server, *url.URL) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		middleFunc(r)

		w.WriteHeader(statusCode)
		_, err := w.Write(bodyResp)
		if err != nil {
			t.Fatalf("Method:%s Status:%d Body: %s Err:%+v", r.Method, statusCode, bodyResp, err)
		}
	}))

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return server, serverURL
}

func setupThisRouterServer(t *testing.T, backendMainURL, backendCanaryURL string, sidecarURL string, circuitBreakerParam circuitBreakerParam) *Server {
	t.Helper()

	c := config.Config{
		MainTarget:   backendMainURL,
		CanaryTarget: backendCanaryURL,
		SidecarURL:   sidecarURL,
		CircuitBreaker: config.CircuitBreaker{
			RequestLimitCanary: circuitBreakerParam.RequestLimitCanary,
			ErrorLimitCanary:   circuitBreakerParam.ErrorLimitCanary,
		}}
	s, err := NewServer(c)
	if err != nil {
		t.Fatal(errors.ErrorStack(err))
	}

	return s
}

// Create a mock canary server that give responses with 50:50 distribution for canaryrouter.StatusCodeMain and canaryrouter.StatusCodeCanary.
// Make sure sidecar.OriginRequest.Body from rest client has only integer value.
func setupCanaryServerFiftyFifty(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}

		i, err := strconv.Atoi(string(body))
		if err != nil {
			t.Fatal(err)
		}

		if i%2 == 0 {
			w.WriteHeader(canaryrouter.StatusCodeMain)
		} else {
			w.WriteHeader(canaryrouter.StatusCodeCanary)
		}

	}))

	return server
}

func newRequest(method, url, body string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return nil, errors.Trace(err)
	}
	return req, nil
}

func restClientCall(t *testing.T, client *http.Client, restRequest restRequest) (*http.Response, []byte) {
	req, err := newRequest(restRequest.httpMethod, restRequest.targetURL, restRequest.bodyPayload)
	if err != nil {
		t.Fatal(err)
	}

	req.Header = restRequest.httpHeader

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	gotBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	return resp, gotBody
}

func Test_convertToBool(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		want    bool
		wantErr bool
	}{
		{name: "'true'", args: "true", want: true, wantErr: false},
		{name: "'false'", args: "false", want: false, wantErr: false},
		{name: "'t'", args: "t", want: false, wantErr: true},
		{name: "'f'", args: "f", want: false, wantErr: true},
		{name: "'1'", args: "1", want: false, wantErr: true},
		{name: "'0'", args: "0", want: false, wantErr: true},
		{name: "'TRUE'", args: "TRUE", want: false, wantErr: true},
		{name: "empty", args: "", want: false, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertToBool(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertToBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("convertToBool() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_trimRequestPathPrefix(t *testing.T) {
	type args struct {
		url    string
		prefix string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "no prefix", args: args{url: "http://localhost:8090/foo/bar", prefix: ""}, want: "http://localhost:8090/foo/bar"},
		{name: "/foo", args: args{url: "http://localhost:8090/foo/bar", prefix: "/foo"}, want: "http://localhost:8090/bar"},
		{name: "foo", args: args{url: "http://localhost:8090/foo/bar", prefix: "foo"}, want: "http://localhost:8090/foo/bar"},
		{name: "/foo/", args: args{url: "http://localhost:8090/foo/bar", prefix: "/foo/"}, want: "http://localhost:8090/bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argsURL, err := url.Parse(tt.args.url)
			if err != nil {
				t.Fatal(err)
			}

			wantURL, err := url.Parse(tt.want)
			if err != nil {
				t.Fatal(err)
			}

			got := trimRequestPathPrefix(argsURL, tt.args.prefix)
			argsURL.Path = got

			if wantURL.String() != argsURL.String() {
				t.Errorf("trimPrefix() = %v, want %v", argsURL.String(), wantURL.String())
			}

		})
	}
}
