package main

import (
	"github.com/imdario/mergo"
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tiket-libre/canary-router/canaryrouter"
	"github.com/tiket-libre/canary-router/config"
	"github.com/tiket-libre/canary-router/instrumentation"
	routerversion "github.com/tiket-libre/canary-router/version"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	appConfig config.Config
	cfgFile   string

	defaultConfig = config.Config{
		Server: config.HTTPServerConfig{
			ReadTimeout:  5,
			WriteTimeout: 15,
			IdleTimeout:  120,
		},
		Client: config.MultiHTTPClientConfig{
			MainAndCanary: config.HTTPClientConfig{
				Timeout:         5,
				MaxIdleConns:    1000,
				IdleConnTimeout: 30,
			},
			Sidecar: config.HTTPClientConfig{
				Timeout:         2,
				MaxIdleConns:    1000,
				IdleConnTimeout: 30,
			},
		},
	}
)

func main() {
	cobra.OnInitialize(initConfig)

	routerversion.Info = routerversion.Type{Version: version, Commit: routerversion.ShortHash(commit), Date: date}
	rootCmd := &cobra.Command{
		Use:   "canary-router",
		Short: "A HTTP request forwarding tool",
		Long:  `canary-router forwards HTTP request based on your custom logic`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := instrumentation.Initialize(appConfig.Instrumentation); err != nil {
				return errors.Trace(err)
			}

			server, err := canaryrouter.NewServer(appConfig)
			if err != nil {
				return errors.Trace(err)
			}

			return server.Run()
		},
	}
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.json", "config file")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(errors.ErrorStack(err))
	}
}

func initConfig() {
	// Don't forget to read config either from cfgFile or from home directory!
	viper.SetConfigFile(cfgFile)

	log.Printf("Canary Router version: %s", routerversion.Info)
	log.Printf("Loaded with config file: %s", cfgFile)

	viper.SetConfigType("json")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Can't read config: %v", errors.ErrorStack(err))
	}

	if err := viper.Unmarshal(&appConfig); err != nil {
		log.Fatalf("Unable to decode into config struct: %v", errors.ErrorStack(err))
	}

	if err := mergo.Merge(&appConfig, defaultConfig); err != nil {
		log.Fatalf("Unable to set default values: %v", errors.ErrorStack(err))
	}

	log.Printf("%+v", appConfig)
}
