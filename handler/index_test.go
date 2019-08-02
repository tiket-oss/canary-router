package handler

import (
	"bytes"
	canaryrouter "canary-router"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func Test_viaProxy_integration(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	backendMainBody := "Hello, I'm Main!"
	backendMain, backendMainURL := setupServer(t, []byte(backendMainBody), http.StatusOK)
	defer backendMain.Close()

	backendCanaryBody := "Hello, I'm Canary!"
	backendCanary, backendCanaryURL := setupServer(t, []byte(backendCanaryBody), http.StatusOK)
	defer backendCanary.Close()

	proxies, err := canaryrouter.BuildProxies(backendMainURL.String(), backendCanaryURL.String())
	if err != nil {
		t.Fatal(err)
	}

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

	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("%d %s", tc.argStatusCode, tc.name), func(t *testing.T) {
			//t.Parallel()

			backendSidecar, backendSidecarURL := setupServer(t, []byte("Static sidecar body"), tc.argStatusCode)
			defer backendSidecar.Close()

			thisRouter := httptest.NewServer(http.HandlerFunc(viaProxy(proxies, &http.Client{}, backendSidecarURL.String())))
			defer thisRouter.Close()

			dummyBody := "This is DUMMY body"

			methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch}
			for _, m := range methods {
				t.Run(m, func(t *testing.T) {
					//t.Parallel()

					req, err := newRequest(m, thisRouter.URL+"/foo/bar", dummyBody)
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
		})
	}
}

func setupServer(t *testing.T, body []byte, statusCode int) (*httptest.Server, *url.URL) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write(body)
	}))

	serverUrl, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return server, serverUrl
}

func newRequest(method, url string, body interface{}) (*http.Request, error) {
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		return nil, err
	}
	return req, nil
}
