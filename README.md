[![Go Report Card](https://goreportcard.com/badge/github.com/appscode/g2)](https://goreportcard.com/report/github.com/appscode/g2)
[![Build Status](https://travis-ci.org/appscode/g2.svg?branch=master)](https://travis-ci.org/appscode/g2)
[![codecov](https://codecov.io/gh/appscode/g2/branch/master/graph/badge.svg)](https://codecov.io/gh/appscode/g2)
[![Docker Pulls](https://img.shields.io/docker/pulls/appscode/gearmand.svg)](https://hub.docker.com/r/appscode/gearmand/)
[![Slack](https://slack.appscode.com/badge.svg)](https://slack.appscode.com)
[![Twitter](https://img.shields.io/twitter/follow/appscodehq.svg?style=social&logo=twitter&label=Follow)](https://twitter.com/intent/follow?screen_name=AppsCodeHQ)

G2
==========

G2 is a modern implementation of [Gearman](http://gearman.org/) in [Go Programming Language](http://golang.org). G2 includes various improvements in retry and connection logic for using in Kubernetes. It comes with built-in Prometheus ready metrics. G2 also implements scheduled jobs via cron expressions.


The client package is used for sending jobs to the Gearman job server and getting responses from the server.

	"github.com/appscode/g2/client"

The worker package will help developers in developing Gearman worker service easily.

	"github.com/appscode/g2/worker"
	    
The gearadmin package implements a client for the [gearman admin protocol](http://gearman.org/protocol/).

    "github.com/appscode/g2/gearadmin"

[![GoDoc](https://godoc.org/github.com/appscode/g2?status.png)](https://godoc.org/github.com/appscode/g2)

Install
=======

Install the client package:

> $ go get github.com/appscode/g2/client

Install the worker package:

> $ go get github.com/appscode/g2/worker

Both of them:

> $ go get github.com/appscode/g2

G2 gearman server can be installed via pre-built Docker images mentioned in the Github releases. G2 gearman server can also be installed in Kubernetes via [Helm](https://helm.sh/) using the [chart](/chart/g2) included in this repository or from official charts repository. To install the chart with the release name `my-release`:
```bash
$ helm install chart/g2 --name my-release
```
To see the detailed configuration options, visit [here](/chart/g2/README.md).

Usage
=====
## Server
	how to start gearmand?

	./gearmand run --v=3 --addr="0.0.0.0:4730"

	how to specify leveldb location?

	./gearmand run --v=3 --storage-dir=/my-dir --addr="0.0.0.0:4730"

how to export metrics to Prometheus:

	http://localhost:3000/metrics

how to list all workers ?

	http://localhost:3000/workers

how to list workers by "cando" ?

	http://localhost:3000/workers/<function>

how to list all jobs ?

	http://localhost:3000/jobs

how to query job status ?

	http://localhost:3000/jobs/<jobhandle>

how to change monitor address ?

	./gearmand run --v=3 --web.addr=:4567

## Worker

```go
// Limit number of concurrent jobs execution.
// Use worker.Unlimited (0) if you want no limitation.
w := worker.New(worker.OneByOne)
w.ErrorHandler = func(e error) {
	log.Println(e)
}
w.AddServer("tcp4", "127.0.0.1:4730")
// Use worker.Unlimited (0) if you want no timeout
w.AddFunc("ToUpper", ToUpper, worker.Unlimited)
// This will give a timeout of 5 seconds
w.AddFunc("ToUpperTimeOut5", ToUpper, 5)

if err := w.Ready(); err != nil {
	log.Fatal(err)
	return
}
go w.Work()
```

## Client

```go
c, err := client.New("tcp4", "127.0.0.1:4730")
defer c.Close()
//error handling
c.ErrorHandler = func(e error) {
	log.Println(e)
}
echo := []byte("Hello\x00 world")
echomsg, err := c.Echo(echo)
log.Println(string(echomsg))
jobHandler := func(resp *client.Response) {
	log.Printf("%s", resp.Data)
}
handle, err := c.Do("ToUpper", echo, runtime.JobNormal, jobHandler)
```

## Gearman Admin Client
Package gearadmin provides simple bindings to the gearman admin protocol: http://gearman.org/protocol/. Here's an example program that outputs the status of all worker queues in gearman:

```go
c, err := net.Dial("tcp", "localhost:4730")
if err != nil {
	panic(err)
}
defer c.Close()
admin := gearadmin.NewGearmanAdmin(c)
status, _ := admin.Status()
fmt.Printf("%#v\n", status)
```

Build Instructions
==================
```sh
# dev build
./hack/make.py

# Install/Update dependency (needs glide)
glide slow

# Build Docker image
./hack/docker/setup.sh

# Push Docker image (https://hub.docker.com/r/appscode/gearmand/)
./hack/docker/setup.sh push

# Deploy to Kubernetes (one time setup operation)
kubectl run gearmand --image=appscode/gearmand:<tag> --replica=1

# Deploy new image
kubectl set image deployment/gearmand tc=appscode/gearmand:<tag>
```

Acknowledgement
===============
 * Client and Worker package forked from https://github.com/mikespook/gearman-go
 * Server package forked from https://github.com/ngaut/gearmand
 * Gearadmin client forked from https://github.com/Clever/gearadmin
 * Gearman project (http://gearman.org/protocol/)

License
==================================
Apache 2.0. See [LICENSE](LICENSE).

- Copyright (C) 2016-2017 by AppsCode Inc.
- Copyright (C) 2016 by Clever.com (portions of gearadmin client)
- Copyright (c) 2014 [ngaut](https://github.com/ngaut) (portions of gearmand)
- Copyright (C) 2011 by Xing Xing (portions of client and worker)
