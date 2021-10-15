package main

import (
  "io"
  "log"
	"net/http"
  
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func main() {
	tracer.Start()
	mux := httptrace.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello World!\n")
	})
  log.Println("HTTP server listening on 0.0.0.0:7777")
	http.ListenAndServe("0.0.0.0:7777", mux)
}