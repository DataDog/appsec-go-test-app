// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//go:build appsec
// +build appsec

package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"

	"github.com/Datadog/appsec-go-test-app/vulnerable"
	muxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
)

func NewRouter(templateDir string) *muxtrace.Router {
	db, err := vulnerable.PrepareSQLDB(10)
	if err != nil {
		log.Println("could not prepare the sql database :", err)
	}

	t, err := template.ParseFiles(filepath.Join(templateDir, "category.html"))
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

	r.PathPrefix("/").Handler(http.FileServer(http.Dir(templateDir)))

	return r
}
