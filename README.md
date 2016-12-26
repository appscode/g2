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

Acknowledgement
==========
 * Client and Worker package forked from https://github.com/mikespook/gearman-go
 * Server package forked from https://github.com/ngaut/gearmand
 * Gearman project (http://gearman.org/protocol/)

Open Source - MIT Software License
==================================

See LICENSE.
