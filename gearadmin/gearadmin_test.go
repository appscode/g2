package gearadmin

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// MockGearmand implements io.ReadWriter and can be used in place of an actual connection to gearmand.
type MockGearmand struct {
	buf       bytes.Buffer
	Responses map[string]string
}

func (mg *MockGearmand) Read(p []byte) (int, error) {
	return mg.buf.Read(p)
}

func (mg *MockGearmand) Write(p []byte) (int, error) {
	cmd := strings.Trim(string(p), "\n")
	val, ok := mg.Responses[cmd]
	if !ok {
		mg.buf.Write([]byte("ERR UNKNOWN_COMMAND"))
	} else {
		mg.buf.Write([]byte(val))
	}
	return len(p), nil
}

func TestStatus(t *testing.T) {
	mockGearmand := MockGearmand{}
	mockGearmand.Responses = map[string]string{
		"status": `fn1	3	2	1
fn2	0	1	2
.`,
	}
	ga := GearmanAdmin{&mockGearmand}
	statuses, err := ga.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 2 {
		t.Fatalf("Expected two statuses")
	}
	checkStatus := func(status Status, fn string, total, running, available int) {
		if status.Function != fn {
			t.Fatalf("Incorrect function: expected '%v', got '%v'", fn, status.Function)
		}
		if status.Total != total {
			t.Fatalf("Incorrect total: expected %v, got %v", total, status.Total)
		}
		if status.Running != running {
			t.Fatalf("Incorrect running: expected %v, got %v", running, status.Running)
		}
		if status.AvailableWorkers != available {
			t.Fatalf("Incorrect AVAILABLE_WORKERS: expected %v, got %v", available, status.AvailableWorkers)
		}
	}
	checkStatus(statuses[0], "fn1", 3, 2, 1)
	checkStatus(statuses[1], "fn2", 0, 1, 2)
}

func TestPriorityStatus(t *testing.T) {
	mockGearmand := MockGearmand{}
	mockGearmand.Responses = map[string]string{
		"prioritystatus": `fn1	3	2	1	0
fn2	0	1	2	0
.`,
	}
	ga := GearmanAdmin{&mockGearmand}
	pStatuses, err := ga.PriorityStatus()
	if err != nil {
		t.Fatal(err)
	}
	if len(pStatuses) != 2 {
		t.Fatalf("Expected two prioritystatuses")
	}
	checkPriorityStatus := func(pStatus PriorityStatus, fn string, hq, nq, lq, available int) {
		if pStatus.Function != fn {
			t.Fatalf("Incorrect function: expected '%v', got '%v'", fn, pStatus.Function)
		}
		if pStatus.HighQueued != hq {
			t.Fatalf("Incorrect HIGH-QUEUED: expected %v, got %v", hq, pStatus.HighQueued)
		}
		if pStatus.NormalQueued != nq {
			t.Fatalf("Incorrect NORMAL-QUEUED: expected %v, got %v", nq, pStatus.NormalQueued)
		}
		if pStatus.LowQueued != lq {
			t.Fatalf("Incorrect LOW-QUEUED: expected %v, got %v", lq, pStatus.LowQueued)
		}
		if pStatus.AvailableWorkers != available {
			t.Fatalf("Incorrect AVAILABLE_WORKERS: expected %v, got %v", available, pStatus.AvailableWorkers)
		}
	}
	checkPriorityStatus(pStatuses[0], "fn1", 3, 2, 1, 0)
	checkPriorityStatus(pStatuses[1], "fn2", 0, 1, 2, 0)
}

func TestWorkers(t *testing.T) {
	mockGearmand := MockGearmand{}
	mockGearmand.Responses = map[string]string{
		"workers": `74 10.0.1.167 - :
284 10.0.2.16 - : fn1
284 10.0.2.16 - : fn1 fn2
.`,
	}
	ga := GearmanAdmin{&mockGearmand}
	workers, err := ga.Workers()
	if err != nil {
		t.Fatal(err)
	}
	if len(workers) != 3 {
		t.Fatalf("Expected three workers: %v", workers)
	}
	checkWorker := func(worker Worker, fd, ip, cid string, functions []string) {
		if worker.Fd != fd {
			t.Fatalf("Expected worker Fd '%v', got '%v'", fd, worker.Fd)
		}
		if worker.IPAddress != ip {
			t.Fatalf("Expected worker IPAddress '%v', got '%v'", ip, worker.IPAddress)
		}
		if worker.ClientID != cid {
			t.Fatalf("Expected worker ClientID '%v', got '%v'", cid, worker.ClientID)
		}
		if fmt.Sprintf("%v", worker.Functions) != fmt.Sprintf("%v", functions) {
			t.Fatalf("Expected worker Functions '%#v', got '%#v'", functions, worker.Functions)
		}
	}
	checkWorker(workers[0], "74", "10.0.1.167", "-", []string{})
	checkWorker(workers[1], "284", "10.0.2.16", "-", []string{"fn1"})
	checkWorker(workers[2], "284", "10.0.2.16", "-", []string{"fn1", "fn2"})
}

func TestCancel(t *testing.T) {
	mockGearmand := MockGearmand{}
	mockGearmand.Responses = map[string]string{}
	handleResponsePair := []struct {
		handle   string
		success  bool
		response string
	}{
		{"S:-icee:-17700-1483598255-1", true, "OK"},
		{"S:-icee:-17700-1483598255-2", true, "OK"},
		{"S:-icee:-17700-1483598255-5", false, "Error: handle `S:-icee:-17700-1483598255-5` not found"},
		{"H:-icee:-17700-1483598255-5", false, "Error: Invalid handle `H:-icee:-17700-1483598255-5`, valid schedule job handle should start with `S:`"},
		{"a-invalid-handle", false, "Error: Invalid handle `H:-icee:-17700-1483598255-5`, valid schedule job handle should start with `S:`"},
	}
	for _, o := range handleResponsePair {
		mockGearmand.Responses[fmt.Sprintf("cancel-job %v", o.handle)] = o.response
	}
	ga := GearmanAdmin{&mockGearmand}
	for _, v := range handleResponsePair {
		isSuccess, err := ga.Cancel(v.handle)
		if isSuccess != v.success {
			t.Fatalf("Expected response '%v', got '%v'", v.success, isSuccess)
		}
		if v.response != "OK" && err.Error() != v.response {
			t.Fatalf("Expected error '%v', got '%v'", v.response, err.Error())
		}
	}
}
