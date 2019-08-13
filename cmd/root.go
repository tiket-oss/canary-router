package main

import (
	"log"

	"github.com/juju/errors"

	"github.com/tiket-libre/canary-router/config"
	"github.com/tiket-libre/canary-router/instrumentation"
	"github.com/tiket-libre/canary-router/server"
	routerversion "github.com/tiket-libre/canary-router/version"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var appConfig config.Config
var cfgFile string

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.json", "config file")
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

	err := viper.Unmarshal(&appConfig)
	log.Printf("%+v", appConfig)
	if err != nil {
		log.Fatalf("Unable to decode into config struct: %v", errors.ErrorStack(err))
	}
}

var rootCmd = &cobra.Command{
	Use:   "canary-router",
	Short: "A HTTP request forwarding tool",
	Long:  `canary-router forwards HTTP request based on your custom logic`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := instrumentation.Initialize(appConfig.Instrumentation); err != nil {
			return errors.Trace(err)
		}

		server, err := server.NewServer(appConfig)
		if err != nil {
			return errors.Trace(err)
		}

		return server.Run()
	},
}
