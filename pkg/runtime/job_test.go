package runtime

import (
	"testing"
)

func TestJob2String(t *testing.T) {
	job := &Job{}
	if job.String() == "" {
		t.Error("job to string failed")
	}
}
