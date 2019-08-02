package canaryrouter

import (
	"testing"
)

func Test_newReverseProxy(t *testing.T) {
	type urlCase struct {
		urlTarget string
		wantProxy bool
		wantErr   bool
	}

	type testCase struct {
		name  string
		cases []urlCase
	}

	testCases := []testCase{{
		name: "good url",
		cases: []urlCase{
			{urlTarget: "http://localhost:34556", wantProxy: true, wantErr: false},
			{urlTarget: "http://localhost", wantProxy: true, wantErr: false},
			{urlTarget: "http://192.168.0.1", wantProxy: true, wantErr: false},
			{urlTarget: "http://192.168.0.1:3456", wantProxy: true, wantErr: false},
		},
	}, {
		name: "bad url",
		cases: []urlCase{
			{urlTarget: "localhost", wantProxy: false, wantErr: true},
			{urlTarget: "19268", wantProxy: false, wantErr: true},
		}},
	}

	for _, tc := range testCases {
		tc := tc
		for _, c := range tc.cases {
			c := c
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				got, err := newReverseProxy(c.urlTarget)

				if (err != nil) != c.wantErr {
					t.Errorf("newReverseProxy() error = %v, wantErr %v", err, c.wantErr)
				}

				if (got != nil) != c.wantProxy {
					t.Errorf("newReverseProxy() gotProxy = %v, wantProxy %v", got, c.wantProxy)
				}
			})
		}
	}

}
