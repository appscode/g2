package server

import (
	"net/http"

	"github.com/appscode/go/runtime"
	"github.com/appscode/pat"
)

func safeHandler(f func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer runtime.HandleCrash()
		f(w, r)
	})
}

func registerAPIHandlers(s *Server) {
	m := pat.New()

	m.Get("/jobs", safeHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		e := &event{tp: ctrlGetJob, result: createResCh()}
		s.ctrlEvtCh <- e
		res := <-e.result
		switch r := res.(type) {
		case string:
			w.Write([]byte(r))
		}
	}))

	m.Get("/jobs/:handle", safeHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		params, _ := pat.FromContext(r.Context())
		e := &event{tp: ctrlGetJob, handle: params.Get(":handle"), result: createResCh()}
		s.ctrlEvtCh <- e
		res := <-e.result
		switch r := res.(type) {
		case string:
			w.Write([]byte(r))
		}
	}))

	m.Get("/workers", safeHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		e := &event{tp: ctrlGetWorker, args: &Tuple{t0: ""}, result: createResCh()}
		s.ctrlEvtCh <- e
		res := <-e.result
		switch r := res.(type) {
		case string:
			w.Write([]byte(r))
		}
	}))

	m.Get("/workers/:cando", safeHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		params, _ := pat.FromContext(r.Context())
		e := &event{tp: ctrlGetWorker, args: &Tuple{t0: params.Get(":cando")}, result: createResCh()}
		s.ctrlEvtCh <- e
		res := <-e.result
		switch r := res.(type) {
		case string:
			w.Write([]byte(r))
		}
	}))

	m.Get("/cronjobs", safeHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		e := &event{tp: ctrlGetCronJob, result: createResCh()}
		s.ctrlEvtCh <- e
		res := <-e.result
		switch r := res.(type) {
		case string:
			w.Write([]byte(r))
		}
	}))

	//get job information using job handle
	m.Get("/cronjobs/:handle", safeHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		params, _ := pat.FromContext(r.Context())
		e := &event{tp: ctrlGetCronJob, handle: params.Get(":handle"), result: createResCh()}
		s.ctrlEvtCh <- e
		res := <-e.result
		switch r := res.(type) {
		case string:
			w.Write([]byte(r))
		}
	}))

	http.Handle("/", m)
}
