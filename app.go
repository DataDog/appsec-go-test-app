package main

import (
	"io"
	"log"
	"net/http"
	"time"

	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func main() {
	tracer.Start()
	mux := httptrace.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello World!\n")
	})

	mux.HandleFunc("/sql", func(w http.ResponseWriter, r *http.Request) {
		span, _ := tracer.StartSpanFromContext(r.Context(), "mysql.query",
			tracer.SpanType(ext.SpanTypeSQL),
			tracer.StartTime(time.Now()),
		)
		defer span.Finish()
		span.SetTag("sql.query_type", "Query")
		span.SetTag(ext.ResourceName, "SELECT * FROM users")

		w.WriteHeader(500)
	})
	
	log.Println("HTTP server listening on 0.0.0.0:7777")
	http.ListenAndServe("0.0.0.0:7777", mux)
}
