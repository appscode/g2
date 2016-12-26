package runtime

import (
	"testing"
)

func TestCmdArgCount(t *testing.T) {
	if PT_SubmitJobEpoch.ArgCount() != 4 {
		t.Error("argument count not match")
	}
}
