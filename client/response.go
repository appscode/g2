package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"

	rt "github.com/appscode/g2/pkg/runtime"
)

// Response handler
type ResponseHandler func(*Response)

// response
type Response struct {
	DataType  rt.PT
	Data, UID []byte
	Handle    string
}

// Extract the Response's result.
// if data == nil, err != nil, then worker failing to execute job
// if data != nil, err != nil, then worker has a exception
// if data != nil, err == nil, then worker complate job
// after calling this method, the Response.Handle will be filled
func (resp *Response) Result() (data []byte, err error) {
	switch resp.DataType {
	case rt.PT_WorkFail:
		resp.Handle = string(resp.Data)
		err = ErrWorkFail
		return
	case rt.PT_WorkException:
		err = ErrWorkException
		fallthrough
	case rt.PT_WorkComplete:
		data = resp.Data
	default:
		err = ErrDataType
	}
	return
}

// Extract the job's update
func (resp *Response) Update() (data []byte, err error) {
	if resp.DataType != rt.PT_WorkData &&
		resp.DataType != rt.PT_WorkWarning {
		err = ErrDataType
		return
	}
	data = resp.Data
	if resp.DataType == rt.PT_WorkWarning {
		err = ErrWorkWarning
	}
	return
}

// Decode a job from byte slice
func decodeResponse(data []byte) (resp *Response, l int, err error) {
	a := len(data)
	if a < rt.MinPacketLength { // valid package should not less 12 bytes
		err = fmt.Errorf("Invalid data: %v", data)
		return
	}
	dl := int(binary.BigEndian.Uint32(data[8:12]))
	if a < rt.MinPacketLength+dl {
		err = fmt.Errorf("Invalid data: %v", data)
		return
	}
	dt := data[rt.MinPacketLength : dl+rt.MinPacketLength]
	if len(dt) != int(dl) { // length not equal
		err = fmt.Errorf("Invalid data: %v", data)
		return
	}
	resp = getResponse()
	resp.DataType, err = rt.NewPT(binary.BigEndian.Uint32(data[4:8]))
	if err != nil {
		return
	}
	switch resp.DataType {
	case rt.PT_JobCreated:
		resp.Handle = string(dt)
	case rt.PT_StatusRes, rt.PT_WorkData, rt.PT_WorkWarning, rt.PT_WorkStatus,
		rt.PT_WorkComplete, rt.PT_WorkException:
		s := bytes.SplitN(dt, []byte{'\x00'}, 2)
		if len(s) >= 2 {
			resp.Handle = string(s[0])
			resp.Data = s[1]
		} else {
			err = fmt.Errorf("Invalid data: %v", data)
			return
		}
	case rt.PT_WorkFail:
		s := bytes.SplitN(dt, []byte{'\x00'}, 2)
		if len(s) >= 1 {
			resp.Handle = string(s[0])
		} else {
			err = fmt.Errorf("Invalid data: %v", data)
			return
		}
	case rt.PT_EchoRes:
		fallthrough
	default:
		resp.Data = dt
	}
	l = dl + rt.MinPacketLength
	return
}

func (resp *Response) Status() (status *Status, err error) {
	data := bytes.SplitN(resp.Data, []byte{'\x00'}, 2)
	if len(data) != 2 {
		err = fmt.Errorf("Invalid data: %v", resp.Data)
		return
	}
	status = &Status{}
	status.Handle = resp.Handle
	status.Known = true
	status.Running = true
	status.Numerator, err = strconv.ParseUint(string(data[0]), 10, 0)
	if err != nil {
		err = fmt.Errorf("Invalid Integer: %s", data[0])
		return
	}
	status.Denominator, err = strconv.ParseUint(string(data[1]), 10, 0)
	if err != nil {
		err = fmt.Errorf("Invalid Integer: %s", data[1])
		return
	}
	return
}

// status handler
func (resp *Response) _status() (status *Status, err error) {
	data := bytes.SplitN(resp.Data, []byte{'\x00'}, 4)
	if len(data) != 4 {
		err = fmt.Errorf("Invalid data: %v", resp.Data)
		return
	}
	status = &Status{}
	status.Handle = resp.Handle
	status.Known = (data[0][0] == '1')
	status.Running = (data[1][0] == '1')
	status.Numerator, err = strconv.ParseUint(string(data[2]), 10, 0)
	if err != nil {
		err = fmt.Errorf("Invalid Integer: %s", data[2])
		return
	}
	status.Denominator, err = strconv.ParseUint(string(data[3]), 10, 0)
	if err != nil {
		err = fmt.Errorf("Invalid Integer: %s", data[3])
		return
	}
	return
}

func getResponse() (resp *Response) {
	// TODO add a pool
	resp = &Response{}
	return
}
