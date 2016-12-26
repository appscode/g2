package main

import (
	"log"
	"os"
	"sync"

	"github.com/appscode/g2/client"
	rt "github.com/appscode/g2/pkg/runtime"
)

func main() {
	// Set the autoinc id generator
	// You can write your own id generator
	// by implementing IdGenerator interface.
	// client.IdGen = client.NewAutoIncId()

	c, err := client.New(rt.Network, "127.0.0.1:4730")
	if err != nil {
		log.Fatalln(err)
	}
	defer c.Close()
	c.ErrorHandler = func(e error) {
		log.Println(e)
		os.Exit(1)
	}
	echo := []byte("Hello\x00 world")
	echomsg, err := c.Echo(echo)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(string(echomsg))
	jobHandler := func(resp *client.Response) {
		switch resp.DataType {
		case rt.PT_WorkException:
			fallthrough
		case rt.PT_WorkFail:
			fallthrough
		case rt.PT_WorkComplete:
			if data, err := resp.Result(); err == nil {
				log.Printf("RESULT: %v\n", data)
			} else {
				log.Printf("RESULT: %s\n", err)
			}
		case rt.PT_WorkWarning:
			fallthrough
		case rt.PT_WorkData:
			if data, err := resp.Update(); err == nil {
				log.Printf("UPDATE: %v\n", data)
			} else {
				log.Printf("UPDATE: %v, %s\n", data, err)
			}
		case rt.PT_WorkStatus:
			if data, err := resp.Status(); err == nil {
				log.Printf("STATUS: %v\n", data)
			} else {
				log.Printf("STATUS: %s\n", err)
			}
		default:
			log.Printf("UNKNOWN: %v", resp.Data)
		}
	}
	handle, err := c.Do("ToUpper", echo, rt.JobNormal, jobHandler)
	if err != nil {
		log.Fatalln(err)
	}
	status, err := c.Status(handle)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("%v", *status)

	_, err = c.Do("Foobar", echo, rt.JobNormal, jobHandler)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Press Ctrl-C to exit ...")
	var mutex sync.Mutex
	mutex.Lock()
	mutex.Lock()
}
