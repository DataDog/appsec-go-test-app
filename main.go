// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package main

import (
	"embed"
	"encoding/json"
	"github.com/google/uuid"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go-dvwa/vulnerable"

	"gopkg.in/DataDog/dd-trace-go.v1/appsec"
	muxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
)

//go:embed template
var contentFS embed.FS
var sessions = make(map[string]session)

type session struct {
	username string
	expiry   time.Time
	token    string
}

func (s *session) active() bool {
	return s.expiry.After(time.Now())
}

func (s *session) terminate() {
	delete(sessions, s.token)
}

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

	r.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if s := sessionFromRequest(r); s != nil && s.active() {
			http.Redirect(w, r, "/auth", http.StatusFound)
			return
		}
		username := r.FormValue("username")
		password := r.FormValue("password")
		if username == "" || password == "" {
			w.Write([]byte("Username and password can't be empty"))
			w.Write([]byte("<br/><a href='/registration.html'>Registration form</a>."))
			w.Write([]byte("<br/><a href='/'>Home</a>."))
			return
		}
		vulnerable.AddUser(r.Context(), db, username, password)
		http.Redirect(w, r, "/login.html", http.StatusFound)
	})

	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if s := sessionFromRequest(r); s != nil && s.active() {
			http.Redirect(w, r, "/auth", http.StatusFound)
			return
		}
		username := r.FormValue("username")
		password := r.FormValue("password")
		//TODO: add user credential check (backed by db)
		// This endpoint currently only tests the appsec.SetUser SDK, no check is made on credentials
		// and user login is always considered successful.
		user, err := vulnerable.GetUser(r.Context(), db, username)
		if err != nil || user.Password != password {
			appsec.TrackUserLoginFailureEvent(r.Context(), username, user != nil, map[string]string{})
			http.Redirect(w, r, "/auth", http.StatusFound)
			return
		}
		if appsec.TrackUserLoginSuccessEvent(r.Context(), username, map[string]string{}, tracer.WithUserName(username)) != nil {
			return
		}
		token := uuid.NewString()
		s := session{
			username: username,
			expiry:   time.Now().Add(120 * time.Minute),
			token:    token,
		}
		sessions[token] = s
		http.SetCookie(w, &http.Cookie{
			Name:    "session-token",
			Value:   token,
			Expires: s.expiry,
		})
		http.Redirect(w, r, "/auth", http.StatusFound)
	})

	r.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		if s := sessionFromRequest(r); s != nil && s.active() {
			s.terminate()
			r.AddCookie(&http.Cookie{
				Name:    "session-token",
				Value:   "",
				Expires: time.Unix(0, 0),
			})
		}
		http.Redirect(w, r, "/", http.StatusFound)
	})

	r.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if s := sessionFromRequest(r); s != nil && s.active() {
			if appsec.SetUser(r.Context(), s.username) != nil {
				return
			}
			w.Write([]byte("Successfully logged in as <b>" + s.username + "</b>."))
			w.Write([]byte("<br/>Now try blocking the user in the dashboard and refreshing this page."))
			w.Write([]byte("<br/><br/><a href='/logout'>Logout</a>."))
		} else {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Couldn't log in (user probably doesn't exist)."))
			w.Write([]byte("<br/><a href='/login.html'>Login form</a>."))
		}

		w.Write([]byte("<br/><a href='/'>Home page.</a>"))
	})

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

func sessionFromRequest(r *http.Request) *session {
	if c, err := r.Cookie("session-token"); err == nil {
		if s, ok := sessions[c.Value]; ok {
			return &s
		} else {
			r.AddCookie(&http.Cookie{
				Name:    "session-token",
				Value:   "",
				Expires: time.Unix(0, 0),
			})
		}
	}
	return nil
}
