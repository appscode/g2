package worker_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	rt "github.com/appscode/g2/pkg/runtime"
	"github.com/appscode/g2/worker"
)

func ExampleWorker() {
	// An example of worker
	w := worker.New(worker.Unlimited)
	defer w.Close()
	// Add a gearman job server
	if err := w.AddServer(rt.Network, "127.0.0.1:4730"); err != nil {
		fmt.Println(err)
		return
	}
	// A function for handling jobs
	foobar := func(job worker.Job) ([]byte, error) {
		return nil, nil
	}
	// Add the function to worker
	if err := w.AddFunc("foobar", foobar, 0); err != nil {
		fmt.Println(err)
		return
	}
	var wg sync.WaitGroup
	// A custome handler, for handling other results, eg. ECHO, dtError.
	w.JobHandler = func(job worker.Job) error {
		if job.Err() == nil {
			fmt.Println(string(job.Data()))
		} else {
			fmt.Println(job.Err())
		}
		wg.Done()
		return nil
	}
	// An error handler for handling worker's internal errors.
	w.ErrorHandler = func(e error) {
		fmt.Println(e)
		// Ignore the error or shutdown the worker
	}
	// Tell Gearman job server: I'm ready!
	if err := w.Ready(); err != nil {
		fmt.Println(err)
		return
	}
	// Running main loop
	go w.Work()
	wg.Add(1)
	// calling Echo
	w.Echo([]byte("Hello"))
	// Waiting results
	wg.Wait()
	// Output: Hello
}

func TestScheduledJob(t *testing.T) {
	// An example of worker
	w := worker.New(worker.Unlimited)
	defer w.Close()
	// Add a gearman job server
	if err := w.AddServer(rt.Network, "127.0.0.1:4730"); err != nil {
		fmt.Println(err)
		return
	}
	// A function for handling jobs
	scheduledJobTest := func(job worker.Job) ([]byte, error) {
		fmt.Println(" Test Function executed. function name: ", "scheduledJobTest", "Parameter: ", string(job.Data()))

		for i := 0; i < 10; i++ {
			time.Sleep(time.Second * 3)
			job.UpdateStatus((i+1)*10, 100)
			fmt.Printf("Running %d%%\n", (i+1)*10)
		}
		fmt.Println("Job finished")
		return nil, nil
	}
	job1 := func(job worker.Job) ([]byte, error) {
		fmt.Println(" Test Function executed. function name: ", "scheduledJobTest", "Parameter: ", string(job.Data()))
		return nil, nil
	}
	// Add the function to worker
	if err := w.AddFunc("scheduledJobTest", scheduledJobTest, 40); err != nil {
		fmt.Println(err)
		return
	}
	// Add the function to worker
	if err := w.AddFunc("job1", job1, 0); err != nil {
		fmt.Println(err)
		return
	}

	var wg sync.WaitGroup
	// A custome handler, for handling other results, eg. ECHO, dtError.
	w.JobHandler = func(job worker.Job) error {
		if job.Err() == nil {
			fmt.Println(string(job.Data()))
		} else {
			fmt.Println(job.Err())
		}
		wg.Done()
		return nil
	}
	// An error handler for handling worker's internal errors.
	w.ErrorHandler = func(e error) {
		fmt.Println(e)
		// Ignore the error or shutdown the worker
	}
	// Tell Gearman job server: I'm ready!
	if err := w.Ready(); err != nil {
		fmt.Println(err)
		return
	}
	// Running main loop
	go w.Work()
	wg.Add(1)
	wg.Wait()
}

func TestScheduledJobWithReconnect(t *testing.T) {
	w := worker.New(worker.Unlimited)
	defer w.Close()
	// Add a gearman job server
	if err := w.AddServer(rt.Network, "127.0.0.1:4730"); err != nil {
		fmt.Println(err)
		return
	}
	// A function for handling jobs
	scheduledJobTest := func(job worker.Job) ([]byte, error) {
		fmt.Println(" Test Function executed. function name: ", "scheduledJobTest", "Parameter: ", string(job.Data()))
		for i := 0; i < 10; i++ {
			time.Sleep(time.Second * 3)
			fmt.Printf("Running %d%%\n", (i+1)*10)
		}
		fmt.Println("Job finished")
		return nil, nil
	}
	job1 := func(job worker.Job) ([]byte, error) {
		fmt.Println(" Test Function executed. function name: ", "scheduledJobTest", "Parameter: ", string(job.Data()))
		return nil, nil
	}
	// Add the function to worker
	if err := w.AddFunc("scheduledJobTest", scheduledJobTest, 0); err != nil {
		fmt.Println(err)
		return
	}
	// Add the function to worker
	if err := w.AddFunc("job1", job1, 0); err != nil {
		fmt.Println(err)
		return
	}

	var wg sync.WaitGroup
	// A custome handler, for handling other results, eg. ECHO, dtError.
	w.JobHandler = func(job worker.Job) error {
		if job.Err() == nil {
			fmt.Println(string(job.Data()))
		} else {
			fmt.Println(job.Err())
		}
		wg.Done()
		return nil
	}
	w.ErrorHandler = func(e error) {
		wdc, wdcok := e.(*worker.WorkerDisconnectError)
		if wdcok {
			fmt.Println("Reconnecting!")
			reconnected := false
			for tries := 20; !reconnected && tries > 0; tries-- {
				rcerr := wdc.Reconnect()
				if rcerr != nil {
					fmt.Println("err: ", rcerr)
					time.Sleep(5 * time.Second)
				} else {
					reconnected = true
				}
			}

			go w.Work()

		} else {
			//panic("Some other kind of error " + e.Error())
		}

	}
	go w.Work()
	time.Sleep(time.Second * 10)
	w.RemoveFunc("job1")
	wg.Add(1)
	wg.Wait()
}
