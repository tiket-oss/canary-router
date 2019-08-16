package main

import (
	"reflect"
	"testing"

	"github.com/tiket-libre/canary-router/config"
)

func Test_mergeConfig(t *testing.T) {
	defaultConfig := config.Config{
		Server: config.HTTPServerConfig{ReadTimeout: 5, WriteTimeout: 15, IdleTimeout: 120},
		Client: config.HTTPClientConfig{Timeout: 5, MaxIdleConns: 100, IdleConnTimeout: 30},
	}

	allSetConfig := config.Config{
		Server: config.HTTPServerConfig{ReadTimeout: 6, WriteTimeout: 7, IdleTimeout: 130},
		Client: config.HTTPClientConfig{Timeout: 8, MaxIdleConns: 200, IdleConnTimeout: 40},
	}

	type args struct {
		targetConfig  *config.Config
		defaultConfig config.Config
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		wantConfig config.Config
	}{
		{name: "if not set, set default", args: args{
			targetConfig: &config.Config{
				Server: config.HTTPServerConfig{ReadTimeout: 0, WriteTimeout: 0, IdleTimeout: 0},
				Client: config.HTTPClientConfig{Timeout: 0, MaxIdleConns: 0, IdleConnTimeout: 0},
			}, defaultConfig: defaultConfig,
		}, wantConfig: defaultConfig},
		{name: "if all set, leave it", args: args{
			targetConfig: &allSetConfig, defaultConfig: defaultConfig,
		}, wantConfig: allSetConfig},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mergeConfig(tt.args.targetConfig, tt.args.defaultConfig); (err != nil) != tt.wantErr {
				t.Errorf("mergeConfig() error = %+v, wantErr %+v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(*tt.args.targetConfig, tt.wantConfig) {
				t.Errorf("mergeConfig() gotConfig = %+v, wantConfig %+v", tt.args.targetConfig, tt.wantConfig)
			}
		})
	}
}
