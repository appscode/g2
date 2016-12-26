package server

import (
	"bytes"
	"testing"

	. "github.com/appscode/g2/pkg/runtime"
)

func TestDecodeArgs(t *testing.T) {
	/*
		00 52 45 51                \0REQ        (Magic)
		00 00 00 07                7            (Packet type: SUBMIT_JOB)
		00 00 00 0d                13           (Packet length)
		72 65 76 65 72 73 65 00    reverse\0    (Function)
		00                         \0           (Unique ID)
		74 65 73 74                test         (Workload)
	*/

	data := []byte{
		0x72, 0x65, 0x76, 0x65, 0x72, 0x73, 0x65, 0x00,
		0x00,
		0x74, 0x65, 0x73, 0x74}
	slice, ok := decodeArgs(PT_SubmitJob, data)
	if !ok {
		t.Error("should be true")
	}

	if len(slice) != 3 {
		t.Error("arg count not match")
	}

	if !bytes.Equal(slice[0], []byte{0x72, 0x65, 0x76, 0x65, 0x72, 0x73, 0x65}) {
		t.Errorf("decode not match %+v", slice)
	}

	if !bytes.Equal(slice[1], []byte{}) {
		t.Error("decode not match")
	}

	if !bytes.Equal(slice[2], []byte{0x74, 0x65, 0x73, 0x74}) {
		t.Error("decode not match")
	}

	data = []byte{
		0x48, 0x3A, 0x2D, 0x51, 0x3A, 0x2D, 0x34, 0x37, 0x31, 0x36, 0x2D, 0x31,
		0x33, 0x39, 0x38, 0x31, 0x30, 0x36, 0x32, 0x33, 0x30, 0x2D, 0x32, 0x00, 0x00, 0x39, 0x38, 0x31,
		0x30,
	}

	slice, ok = decodeArgs(PT_WorkComplete, data)
	if !ok {
		t.Error("should be true")
	}

	if len(slice[0]) == 0 || len(slice[1]) == 0 {
		t.Error("arg count not match")
	}
}
