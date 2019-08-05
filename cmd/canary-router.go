package main

import (
	"log"

	"github.com/tiket-libre/canary-router/instrumentation"
)

func main() {
	if err := instrumentation.Initialize(); err != nil {
		log.Fatal(err)
	}

	Execute()
}
