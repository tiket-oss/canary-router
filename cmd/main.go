package main

import (
	"log"

	"github.com/juju/errors"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(errors.ErrorStack(err))
	}
}
