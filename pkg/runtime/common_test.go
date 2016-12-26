// common_test.go
package runtime

import (
	"strings"
	"testing"
)

func TestCmdDescription(t *testing.T) {
	if strings.Index(CmdDescription(PT_SubmitJobEpoch+1), "unknown") != 0 {
		t.Error("index should be 0")
	}

	if strings.Index(CmdDescription(PT_CanDo), "CAN_DO") != 0 {
		t.Error("index should be 0")
	}

	if strings.Index(CmdDescription(PT_SubmitJobEpoch), "SUBMIT_JOB_EPOCH") != 0 {
		t.Error("index should be 0")
	}
}

func TestCmdArgCount(t *testing.T) {
	if ArgCount(PT_UnUNUSED) != 0 {
		t.Error("argument count not match")
	}

	if ArgCount(PT_SubmitJobEpoch) != 4 {
		t.Error("argument count not match")
	}
}
