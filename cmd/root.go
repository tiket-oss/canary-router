package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"canary-router/config"
	"canary-router/server"
	"log"
	"os"
)

var cfgFile string

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.json", "config file")
}

func initConfig() {
	// Don't forget to read config either from cfgFile or from home directory!
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	}

	log.Printf("=== config file: %s", cfgFile)

	//viper.SetConfigType("json")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Can't read config: %v", err)
	}

	err := viper.Unmarshal(&config.GlobalConfig)
	if err != nil {
		log.Fatalf("Unable to decode into config struct: %v", err)
	}
}



var rootCmd = &cobra.Command{
	Use:   "canary-router",
	Short: "A HTTP request forwarding tool",
	Long: `canary-router forwards HTTP request based on your custom logic`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Do Stuff Here
		return server.Run()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
