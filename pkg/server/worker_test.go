package server

import (
	"testing"
)

func TestWorkerSstatus2str(t *testing.T) {
	if status2str(wsPrepareForSleep) != "prepareForSleep" {
		t.Error("status not match")
	}

	if status2str(wsRunning) != "running" {
		t.Error("status not match")
	}

	if status2str(wsSleep) != "sleep" {
		t.Error("status not match")
	}

	if status2str(10) != "unknown" {
		t.Error("status not match")
	}
}
