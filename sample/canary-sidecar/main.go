package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	canaryrouter "github.com/tiket-libre/canary-router"
)

func main() {

	http.HandleFunc("/sidecar/", func(w http.ResponseWriter, req *http.Request) {
		bodyBytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Printf("Failed to read req.Body: +%v", err)

			w.WriteHeader(canaryrouter.StatusCodeMain)
			return
		}

		log.Printf("Origin http req: %+v", req)

		if string(bodyBytes) == "type=1" {
			w.WriteHeader(canaryrouter.StatusCodeMain)
			return
		} else if string(bodyBytes) == "type=2" {
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
