package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tiket-libre/canary-router"
	"github.com/tiket-libre/canary-router/sidecar"
)

func Test_viaProxy_integration(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	backendMainBody := "Hello, I'm Main!"
	backendMain, backendMainURL := setupServer(t, []byte(backendMainBody), http.StatusOK, func(r *http.Request) {})
	defer backendMain.Close()

	backendCanaryBody := "Hello, I'm Canary!"
	backendCanary, backendCanaryURL := setupServer(t, []byte(backendCanaryBody), http.StatusOK, func(r *http.Request) {})
	defer backendCanary.Close()

	proxies, err := canaryrouter.BuildProxies(backendMainURL.String(), backendCanaryURL.String())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("[Given] No sideCarURL provided [then] default to Main", func(t *testing.T) {
		thisRouter := httptest.NewServer(http.HandlerFunc(viaProxy(proxies, &http.Client{}, "")))
		defer thisRouter.Close()

		_, gotBody := restClientCall(t, thisRouter.Client(), http.MethodPost, thisRouter.URL+"/foo/bar", "foo bar body")
		if string(gotBody) != backendMainBody {
			t.Errorf("Not forwarded to Main. Gotbody: %s", string(gotBody))
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

				bodyResults := map[string]sidecar.OriginRequest{}

				backendSidecar, backendSidecarURL := setupServer(t, []byte("Static sidecar body"), tc.argStatusCode, func(r *http.Request) {
					decoder := json.NewDecoder(r.Body)
					var oriReq sidecar.OriginRequest
					err := decoder.Decode(&oriReq)
					if err != nil {
						t.Fatal(err)
					}

					bodyResults[oriReq.Method] = oriReq
				})
				defer backendSidecar.Close()

				thisRouter := httptest.NewServer(http.HandlerFunc(viaProxy(proxies, &http.Client{}, backendSidecarURL.String())))
				defer thisRouter.Close()

				originBodyContent := "This is DUMMY body"

				for _, m := range methods {
					t.Run(m, func(t *testing.T) {
						//t.Parallel()

						_, gotBody := restClientCall(t, thisRouter.Client(), m, thisRouter.URL+"/foo/bar", originBodyContent)

						if string(gotBody) != string(tc.wantBody) {
							t.Errorf("argStatusCode = %d got = %+v; want = %+v", tc.argStatusCode, gotBody, tc.wantBody)
							t.Errorf("(STR) argStatusCode = %d got = %+v; want = %+v", tc.argStatusCode, string(gotBody), string(tc.wantBody))
						}

					})
				}

				for _, gotOriReq := range bodyResults {
					if gotOriReq.Body != originBodyContent {
						t.Errorf("Got ori body content: %s Want: %s", gotOriReq.Body, originBodyContent)
					}
				}

			})
		}
	})
}

func setupServer(t *testing.T, bodyResp []byte, statusCode int, middleFunc func(r *http.Request)) (*httptest.Server, *url.URL) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		middleFunc(r)

		w.WriteHeader(statusCode)
		w.Write(bodyResp)
	}))

	serverUrl, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return server, serverUrl
}

func newRequest(method, url, body string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	return req, nil
}

func restClientCall(t *testing.T, client *http.Client, method, url, payloadBody string) (*http.Response, []byte) {
	req, err := newRequest(method, url, payloadBody)
	if err != nil {
		t.Fatal(err)
	}

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
