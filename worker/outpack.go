package worker

import (
	"encoding/binary"

	rt "github.com/appscode/g2/pkg/runtime"
)

// Worker side job
type outPack struct {
	dataType rt.PT
	data     []byte
	handle   string
}

func getOutPack() (outpack *outPack) {
	// TODO pool
	return &outPack{}
}

// Encode a job to byte slice
func (outpack *outPack) Encode() (data []byte) {
	var l int
	if outpack.dataType == rt.PT_WorkFail {
		l = len(outpack.handle)
	} else {
		l = len(outpack.data)
		if outpack.handle != "" {
			l += len(outpack.handle) + 1
		}
	}
	data = rt.NewBuffer(l + rt.MinPacketLength)
	binary.BigEndian.PutUint32(data[:4], rt.Req)
	binary.BigEndian.PutUint32(data[4:8], outpack.dataType.Uint32())
	binary.BigEndian.PutUint32(data[8:rt.MinPacketLength], uint32(l))
	i := rt.MinPacketLength
	if outpack.handle != "" {
		hi := len(outpack.handle) + i
		copy(data[i:hi], []byte(outpack.handle))
		if outpack.dataType != rt.PT_WorkFail {
			data[hi] = '\x00'
		}
		i = hi + 1
	}
	if outpack.dataType != rt.PT_WorkFail {
		copy(data[i:], outpack.data)
	}
	return
}
