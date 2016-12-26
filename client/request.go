package client

import (
	"encoding/binary"

	rt "github.com/appscode/g2/pkg/runtime"
)

// Request from client
type request struct {
	DataType rt.PT
	Data     []byte
}

// Encode a Request to byte slice
func (req *request) Encode() (data []byte) {
	l := len(req.Data)           // length of data
	tl := l + rt.MinPacketLength // add 12 bytes head
	data = rt.NewBuffer(tl)
	copy(data[:4], rt.ReqStr)
	binary.BigEndian.PutUint32(data[4:8], req.DataType.Uint32())
	binary.BigEndian.PutUint32(data[8:12], uint32(l))
	copy(data[rt.MinPacketLength:], req.Data)
	return
}

func getRequest() (req *request) {
	// TODO add a pool
	req = &request{}
	return
}

func getJob(id string, funcname, data []byte) (req *request) {
	req = getRequest()
	a := len(funcname)
	b := len(id)
	c := len(data)
	l := a + b + c + 2
	req.Data = rt.NewBuffer(l)
	copy(req.Data[0:a], funcname)
	copy(req.Data[a+1:a+b+1], []byte(id))
	copy(req.Data[a+b+2:], data)
	return
}
