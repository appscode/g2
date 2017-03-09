package client

import (
	"fmt"
	"sync"
	"testing"
	"time"

	rt "github.com/appscode/g2/pkg/runtime"
	"github.com/appscode/log"
)

const (
	TestStr = "Hello world"
)

var client *Client

func init() {
	if client == nil {
		var err error
		client, err = New(rt.Network, "127.0.0.1:4730")
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func TestClientAddServer(t *testing.T) {
	t.Log("Add local server 127.0.0.1:4730")
	var err error
	if client, err = New(rt.Network, "127.0.0.1:4730"); err != nil {
		t.Fatal(err)
	}
	client.ErrorHandler = func(e error) {
		t.Log(e)
	}
}

func TestClientEcho(t *testing.T) {
	echo, err := client.Echo([]byte(TestStr))
	if err != nil {
		t.Error(err)
		return
	}
	if string(echo) != TestStr {
		t.Errorf("Echo error, %s expected, %s got", TestStr, echo)
		return
	}
}

func TestClientDoBg(t *testing.T) {
	handle, err := client.DoBg("scheduledJobTest", []byte("abcdef"), rt.JobNormal)
	if err != nil {
		t.Error(err)
		return
	}
	if handle == "" {
		t.Error("Handle is empty.")
	} else {
		t.Log(handle)
	}
}

func TestClientDoCron(t *testing.T) {
	handle, err := client.DoCron("scheduledJobTest", "* * * * 5", []byte("test data"))
	if err != nil {
		t.Fatal(err)
	}
	if handle == "" {
		t.Error("Handle is empty.")
	} else {
		t.Log(handle)
	}
}

func TestClientDoAt(t *testing.T) {
	handle, err := client.DoAt("scheduledJobTest", time.Now().Add(20*time.Second).Unix(), []byte("test data"))
	if err != nil {
		t.Fatal(err)
	}
	if handle == "" {
		t.Error("Handle is empty.")
	} else {
		t.Log(handle)
	}
}

func TestClientDo(t *testing.T) {
	var wg sync.WaitGroup = sync.WaitGroup{}
	wg.Add(1)
	jobHandler := func(job *Response) {
		switch job.DataType {
		case rt.PT_WorkComplete:
			t.Log("Work complete, handle ", job.Handle)
			wg.Done()
		case rt.PT_WorkException, rt.PT_WorkFail:
			t.Log("Work fail, handle ", job.Handle, " cause: ", string(job.Data))
			wg.Done()
		case rt.PT_WorkData:
			t.Logf("Work data: %+v", string(job.Data))
		case rt.PT_WorkStatus:
			status, err := job.Status()
			if err != nil {
				t.Error(err)
			}
			fmt.Println("Work status, num: %v, denom: %v", status.Numerator, status.Denominator)
		}
	}
	handle, err := client.Do("scheduledJobTest", []byte("abcdef"),
		rt.JobHigh, jobHandler)
	if err != nil {
		t.Error(err)
		return
	}
	if handle == "" {
		t.Error("Handle is empty.")
	} else {
		t.Log(handle)
	}
	wg.Wait()

}

func TestClientStatus(t *testing.T) {
	status, err := client.Status("handle not exists")
	if err != nil {
		t.Error(err)
		return
	}
	if status.Known {
		t.Errorf("The job (%s) shouldn't be known.", status.Handle)
		return
	}
	if status.Running {
		t.Errorf("The job (%s) shouldn't be running.", status.Handle)
		return
	}

	handle, err := client.Do("Delay5sec", []byte("abcdef"), rt.JobLow, nil)
	if err != nil {
		t.Error(err)
		return
	}
	status, err = client.Status(handle)
	if err != nil {
		t.Error(err)
		return
	}
	if !status.Known {
		t.Errorf("The job (%s) should be known.", status.Handle)
		return
	}
	if status.Running {
		t.Errorf("The job (%s) shouldn't be running.", status.Handle)
		return
	}
}

func TestClientClose(t *testing.T) {
	if err := client.Close(); err != nil {
		t.Error(err)
	}
}
