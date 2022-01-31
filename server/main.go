// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func main() {
	tracer.Start()
	defer tracer.Stop()

	bin, err := os.Executable()
	if err != nil {
		log.Panic("could not get the executable filename:", err)
	}
	templateDir := filepath.Join(filepath.Dir(bin), "template")

	mux := NewRouter(templateDir)

	addr := ":8080"
	log.Println("Serving application on", addr)

	err = http.ListenAndServe(addr, mux)
	if err != nil {
		log.Fatalln(err)
	}
}
