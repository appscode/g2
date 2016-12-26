// common_test.go
package runtime

import (
	"strings"
	"testing"
)

func TestCmdDescription(t *testing.T) {
	if strings.Index(CmdDescription(SUBMIT_JOB_EPOCH+1), "unknown") != 0 {
		t.Error("index should be 0")
	}

	if strings.Index(CmdDescription(CAN_DO), "CAN_DO") != 0 {
		t.Error("index should be 0")
	}

	if strings.Index(CmdDescription(SUBMIT_JOB_EPOCH), "SUBMIT_JOB_EPOCH") != 0 {
		t.Error("index should be 0")
	}
}

func TestCmdArgCount(t *testing.T) {
	if ArgCount(UNUSED) != 0 {
		t.Error("argument count not match")
	}

	if ArgCount(SUBMIT_JOB_EPOCH) != 4 {
		t.Error("argument count not match")
	}
}
