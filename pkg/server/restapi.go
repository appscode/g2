package server

import (
	"net/http"
	"os"

	"github.com/appscode/log"
	"github.com/go-macaron/toolbox"
	"github.com/go-martini/martini"
	"github.com/ngaut/stats"
	"gopkg.in/macaron.v1"
)

func getJob(s *Server, params martini.Params) string {
	e := &event{tp: ctrlGetJob,
		jobHandle: params["jobhandle"], result: createResCh()}
	s.ctrlEvtCh <- e
	res := <-e.result

	return res.(string)
}

func getWorker(s *Server, params martini.Params) string {
	e := &event{tp: ctrlGetWorker,
		args: &Tuple{t0: params["cando"]}, result: createResCh()}
	s.ctrlEvtCh <- e
	res := <-e.result

	return res.(string)
}

func registerWebHandler(s *Server) {
	addr := os.Getenv("GEARMAND_MONITOR_ADDR")
	if addr == "" {
		addr = ":3000"
	} else if addr == "-" {
		// Don't start web monitor
		return
	}

	m := macaron.New()
	m.Use(macaron.Logger())
	m.Use(macaron.Recovery())
	m.Use(macaron.Renderer())

	m.Use(toolbox.Toolboxer(m))
	m.Get("/debug/stats", stats.ExpvarHandler)

	m.Get("/job", func(params martini.Params) string {
		return getJob(s, params)
	})

	//get job information using job handle
	m.Get("/job/:jobhandle", func(params martini.Params) string {
		return getJob(s, params)
	})

	m.Get("/worker", func(params martini.Params) string {
		return getWorker(s, params)
	})

	m.Get("/worker/:cando", func(params martini.Params) string {
		return getWorker(s, params)
	})

	log.Fatal(http.ListenAndServe(addr, m))
}
