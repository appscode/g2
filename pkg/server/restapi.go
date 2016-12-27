package server

import (
	"net/http"
	"os"

	"github.com/appscode/log"
	"github.com/go-macaron/toolbox"
	"github.com/ngaut/stats"
	"gopkg.in/macaron.v1"
)

func getJob(s *Server, ctx *macaron.Context) string {
	e := &event{tp: ctrlGetJob,
		jobHandle: ctx.Params("jobhandle"), result: createResCh()}
	s.ctrlEvtCh <- e
	res := <-e.result

	return res.(string)
}

func getWorker(s *Server, ctx *macaron.Context) string {
	e := &event{tp: ctrlGetWorker,
		args: &Tuple{t0: ctx.Params("cando")}, result: createResCh()}
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

	m.Get("/job", func(ctx *macaron.Context) string {
		return getJob(s, ctx)
	})

	//get job information using job handle
	m.Get("/job/:jobhandle", func(ctx *macaron.Context) string {
		return getJob(s, ctx)
	})

	m.Get("/worker", func(ctx *macaron.Context) string {
		return getWorker(s, ctx)
	})

	m.Get("/worker/:cando", func(ctx *macaron.Context) string {
		return getWorker(s, ctx)
	})

	log.Fatal(http.ListenAndServe(addr, m))
}
