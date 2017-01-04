[Website](https://appscode.com) • [Slack](https://slack.appscode.com) • [Forum](https://discuss.appscode.com) • [Twitter](https://twitter.com/AppsCodeHQ)

G2
==========

G2 is a server, worker and client implementation of [Gearman](http://gearman.org/) in [Go Programming Language](http://golang.org). It contains two sub-packages:

The client package is used for sending jobs to the Gearman job server,
and getting responses from the server.

	"github.com/appscode/g2/client"

The worker package will help developers in developing Gearman worker
service easily.

	"github.com/appscode/g2/worker"

[![GoDoc](https://godoc.org/github.com/appscode/g2?status.png)](https://godoc.org/github.com/appscode/g2)

Install
=======

Install the client package:

> $ go get github.com/appscode/g2/client

Install the worker package:

> $ go get github.com/appscode/g2/worker

Both of them:

> $ go get github.com/appscode/g2

Usage
=====
## Server
	how to start gearmand?

	./gearmand --addr="0.0.0.0:4730"

	how to not use leveldb as storage?

	./gearmand --storage-dir= --addr="0.0.0.0:4730"

how to track stats:

	http://localhost:3000/debug/stats

how to list workers by "cando" ?

	http://localhost:3000/worker/function

how to list all workers ?

	http://localhost:3000/worker

how to query job status ?

	http://localhost:3000/job/jobhandle

how to list all jobs ?

	http://localhost:3000/job

how to change monitor address ?

	export GEARMAND_MONITOR_ADDR=:4567

## Worker

```go
// Limit number of concurrent jobs execution.
// Use worker.Unlimited (0) if you want no limitation.
w := worker.New(worker.OneByOne)
w.ErrHandler = func(e error) {
	log.Println(e)
}
w.AddServer("127.0.0.1:4730")
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
// ...
c, err := client.New("tcp4", "127.0.0.1:4730")
// ... error handling
defer c.Close()
c.ErrorHandler = func(e error) {
	log.Println(e)
}
echo := []byte("Hello\x00 world")
echomsg, err := c.Echo(echo)
// ... error handling
log.Println(string(echomsg))
jobHandler := func(resp *client.Response) {
	log.Printf("%s", resp.Data)
}
handle, err := c.Do("ToUpper", echo, client.JobNormal, jobHandler)
// ...
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
 * Gearman project (http://gearman.org/protocol/)

Open Source - MIT Software License
==================================

See LICENSE.
