package main

import (
	"log"

	"github.com/tiket-libre/canary-router/config"
	"github.com/tiket-libre/canary-router/instrumentation"
	"github.com/tiket-libre/canary-router/server"

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

	log.Printf("=== config file: %s", cfgFile)

	viper.SetConfigType("json")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Can't read config: %v", err)
	}

	err := viper.Unmarshal(&appConfig)
	if err != nil {
		log.Fatalf("Unable to decode into config struct: %v", err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "canary-router",
	Short: "A HTTP request forwarding tool",
	Long:  `canary-router forwards HTTP request based on your custom logic`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := instrumentation.Initialize(appConfig.Instrumentation); err != nil {
			return err
		}

		server, err := server.NewServer(appConfig)
		if err != nil {
			return err
		}

		return server.Run()
	},
}
