package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tiket-libre/canary-router"
	"github.com/tiket-libre/canary-router/sidecar"
)

func main() {

	http.HandleFunc("/sidecar", func(w http.ResponseWriter, req *http.Request) {
		decoder := json.NewDecoder(req.Body)
		var oriReq sidecar.OriginRequest
		err := decoder.Decode(&oriReq)
		if err != nil {
			log.Printf("Failed to decode sidecar.OriginRequest json: +%v", err)

			w.WriteHeader(canaryrouter.StatusCodeMain)
			return
		}

		log.Printf("Origin http req: %+v", oriReq)

		if oriReq.Body == "type=1" {
			w.WriteHeader(canaryrouter.StatusCodeMain)
			return
		} else if oriReq.Body == "type=2" {
			w.WriteHeader(canaryrouter.StatusCodeCanary)
			return
		}
	})

	port := os.Getenv("PORT")
	log.Printf("Start canary sidecar server at port %s", port)

	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatal(err)
	}
}
