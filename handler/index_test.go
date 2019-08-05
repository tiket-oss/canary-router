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

						req, err := newRequest(m, thisRouter.URL+"/foo/bar", originBodyContent)
						if err != nil {
							t.Fatal(err)
						}

						resp, err := thisRouter.Client().Do(req)
						if err != nil {
							t.Fatal(err)
						}

						gotBody, err := ioutil.ReadAll(resp.Body)
						if err != nil {
							t.Fatal(err)
						}

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
