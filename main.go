// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package main

import (
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"go-dvwa/vulnerable"

	"gopkg.in/DataDog/dd-trace-go.v1/appsec"
	muxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
)

//go:embed template
var contentFS embed.FS

func main() {
	tracer.Start()
	defer tracer.Stop()

	profiler.Start()
	defer profiler.Stop()

	templateFS, err := fs.Sub(contentFS, "template")
	if err != nil {
		log.Fatalln(err)
	}

	mux := NewRouter(templateFS)
	addr := ":7777"
	log.Println("Serving application on", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalln(err)
	}
}

func NewRouter(templateFS fs.FS) *muxtrace.Router {
	db, err := vulnerable.PrepareSQLDB(10)
	if err != nil {
		log.Println("could not prepare the sql database :", err)
	}

	t, err := template.ParseFS(templateFS, "category.html")
	if err != nil {
		log.Fatalln(err)
	}

	r := muxtrace.NewRouter()

	// /products vulnerable to SQL injections
	r.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		category := r.FormValue("category")
		products, err := vulnerable.GetProducts(r.Context(), db, category)
		if err != nil {
			log.Println(err)
			return
		}
		if err := t.Execute(w, products); err != nil {
			log.Fatalln(err)
		}
	})

	// /products/{category} vulnerable to SQL injections through path parameters
	// example: curl "127.0.0.1:8080/products/toto';select%20*%20from%20'user"
	r.HandleFunc("/products/{category}", func(w http.ResponseWriter, r *http.Request) {
		category := mux.Vars(r)["category"]
		products, err := vulnerable.GetProducts(r.Context(), db, category)
		if err != nil {
			log.Println(err)
			return
		}
		if err := t.Execute(w, products); err != nil {
			log.Fatalln(err)
		}

		w.Header().Set("content-type", "text/html")
	})

	// /api/health vulnerable to shell injections
	r.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		extra := r.FormValue("extra")
		output, err := vulnerable.System(r.Context(), "ping -c1 sqreen.com"+extra)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		enc := json.NewEncoder(w)
		enc.Encode(struct {
			Output string
		}{
			Output: string(output),
		})
	})

	// /api/product allows to manage the list of product catalog
	r.PathPrefix("/api/catalog/").Methods("PUT").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		uid := r.Header.Get("x-api-user-id")
		if span, ok := tracer.SpanFromContext(ctx); ok {
			tracer.SetUser(span, uid)
		}
		var payload interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		appsec.MonitorParsedHTTPBody(ctx, payload)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"status":"ok"}`)
	})

	r.PathPrefix("/").Handler(http.FileServer(http.FS(templateFS)))

	return r
}
