package main

import (
	"log"

	"github.com/juju/errors"
	routerversion "github.com/tiket-libre/canary-router/version"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	routerversion.Info = routerversion.Type{Version: version, Commit: commit, Date: date}

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(errors.ErrorStack(err))
	}
}
