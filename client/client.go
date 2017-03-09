// The client package helps developers connect to Gearmand, send
// jobs and fetch result.
package client

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	rt "github.com/appscode/g2/pkg/runtime"
	"github.com/appscode/log"
)

var (
	DefaultTimeout time.Duration = 1000
)

// One client connect to one server.
// Use Pool for multi-connections.
type Client struct {
	sync.Mutex

	net, addr    string
	respHandler  *responseHandlerMap
	innerHandler *responseHandlerMap
	in           chan *Response
	conn         net.Conn
	rw           *bufio.ReadWriter

	ResponseTimeout time.Duration // response timeout for do() in ms

	ErrorHandler ErrorHandler
}

type responseHandlerMap struct {
	sync.RWMutex
	holder map[string]ResponseHandler
}

func newResponseHandlerMap() *responseHandlerMap {
	return &responseHandlerMap{holder: make(map[string]ResponseHandler, int(rt.QueueSize))}
}

func (r *responseHandlerMap) remove(key string) {
	r.Lock()
	delete(r.holder, key)
	r.Unlock()
}

func (r *responseHandlerMap) get(key string) (ResponseHandler, bool) {
	r.RLock()
	rh, b := r.holder[key]
	r.RUnlock()
	return rh, b
}
func (r *responseHandlerMap) put(key string, rh ResponseHandler) {
	r.Lock()
	r.holder[key] = rh
	r.Unlock()
}

// Return a client.
func New(network, addr string) (client *Client, err error) {
	client = &Client{
		net:             network,
		addr:            addr,
		respHandler:     newResponseHandlerMap(),
		innerHandler:    newResponseHandlerMap(),
		in:              make(chan *Response, rt.QueueSize),
		ResponseTimeout: DefaultTimeout,
	}
	client.conn, err = net.Dial(client.net, client.addr)
	if err != nil {
		return
	}
	client.rw = bufio.NewReadWriter(bufio.NewReader(client.conn),
		bufio.NewWriter(client.conn))
	go client.readLoop()
	go client.processLoop()
	return
}

func (client *Client) write(req *request) (err error) {
	var n int
	buf := req.Encode()
	for i := 0; i < len(buf); i += n {
		n, err = client.rw.Write(buf[i:])
		if err != nil {
			return
		}
	}
	return client.rw.Flush()
}

func (client *Client) read(length int) (data []byte, err error) {
	n := 0
	buf := rt.NewBuffer(rt.BufferSize)
	// read until data can be unpacked
	for i := length; i > 0 || len(data) < rt.MinPacketLength; i -= n {
		if n, err = client.rw.Read(buf); err != nil {
			return
		}
		data = append(data, buf[0:n]...)
		if n < rt.BufferSize {
			break
		}
	}
	return
}

func (client *Client) readLoop() {
	defer close(client.in)
	var data, leftdata []byte
	var err error
	var resp *Response
ReadLoop:
	for client.conn != nil {
		if data, err = client.read(rt.BufferSize); err != nil {
			if opErr, ok := err.(*net.OpError); ok {
				if opErr.Timeout() {
					client.err(err)
				}
				if opErr.Temporary() {
					continue
				}
				break
			}
			client.err(err)
			// If it is unexpected error and the connection wasn't
			// closed by Gearmand, the client should close the conection
			// and reconnect to job server.
			client.Close()
			client.conn, err = net.Dial(client.net, client.addr)
			if err != nil {
				client.err(err)
				break
			}
			client.rw = bufio.NewReadWriter(bufio.NewReader(client.conn),
				bufio.NewWriter(client.conn))
			continue
		}
		if len(leftdata) > 0 { // some data left for processing
			data = append(leftdata, data...)
			leftdata = nil
		}
		for {
			l := len(data)
			if l < rt.MinPacketLength { // not enough data
				leftdata = data
				continue ReadLoop
			}
			if resp, l, err = decodeResponse(data); err != nil {
				leftdata = data[l:]
				continue ReadLoop
			} else {
				client.in <- resp
			}
			data = data[l:]
			if len(data) > 0 {
				continue
			}
			break
		}
	}
}

func (client *Client) processLoop() {
	for resp := range client.in {
		switch resp.DataType {
		case rt.PT_Error:
			log.Errorln("Received error", resp.Data)
			client.err(getError(resp.Data))
		case rt.PT_StatusRes:
			resp = client.handleInner("s"+resp.Handle, resp)
		case rt.PT_JobCreated:
			resp = client.handleInner("c", resp)
		case rt.PT_EchoRes:
			resp = client.handleInner("e", resp)
		case rt.PT_WorkData, rt.PT_WorkWarning, rt.PT_WorkStatus:
			resp = client.handleResponse(resp.Handle, resp)
		case rt.PT_WorkComplete, rt.PT_WorkFail, rt.PT_WorkException:
			client.handleResponse(resp.Handle, resp)
			client.respHandler.remove(resp.Handle)
		}
	}
}

func (client *Client) err(e error) {
	if client.ErrorHandler != nil {
		client.ErrorHandler(e)
	}
}

func (client *Client) handleResponse(key string, resp *Response) *Response {
	if h, ok := client.respHandler.get(key); ok {
		h(resp)
		return nil
	}
	return resp
}

func (client *Client) handleInner(key string, resp *Response) *Response {
	if h, ok := client.innerHandler.get(key); ok {
		h(resp)
		client.innerHandler.remove(key)
		return nil
	}
	return resp
}

type handleOrError struct {
	handle string
	err    error
}

func (client *Client) do(funcname string, data []byte, flag rt.PT) (handle string, err error) {
	if client.conn == nil {
		return "", ErrLostConn
	}
	var result = make(chan handleOrError, 1)
	client.Lock()
	defer client.Unlock()
	client.innerHandler.put("c", func(resp *Response) {
		if resp.DataType == rt.PT_Error {
			err = getError(resp.Data)
			result <- handleOrError{"", err}
			return
		}
		handle = resp.Handle
		result <- handleOrError{handle, nil}
	})
	id := IdGen.Id()
	req := getJob(id, []byte(funcname), data)
	req.DataType = flag
	if err = client.write(req); err != nil {
		client.innerHandler.remove("c")
		return
	}
	var timer = time.After(client.ResponseTimeout * time.Millisecond)
	select {
	case ret := <-result:
		return ret.handle, ret.err
	case <-timer:
		client.innerHandler.remove("c")
		return "", ErrLostConn
	}
	return
}

// Call the function and get a response.
// flag can be set to: JobLow, JobNormal and JobHigh
func (client *Client) Do(funcname string, data []byte,
	flag byte, h ResponseHandler) (handle string, err error) {
	var datatype rt.PT
	switch flag {
	case rt.JobLow:
		datatype = rt.PT_SubmitJobLow
	case rt.JobHigh:
		datatype = rt.PT_SubmitJobHigh
	default:
		datatype = rt.PT_SubmitJob
	}
	handle, err = client.do(funcname, data, datatype)
	if err == nil && h != nil {
		client.respHandler.put(handle, h)
	}
	return
}

// Call the function in background, no response needed.
// flag can be set to: JobLow, JobNormal and JobHigh
func (client *Client) DoBg(funcname string, data []byte, flag byte) (handle string, err error) {
	var datatype rt.PT
	switch flag {
	case rt.JobLow:
		datatype = rt.PT_SubmitJobLowBG
	case rt.JobHigh:
		datatype = rt.PT_SubmitJobHighBG
	default:
		datatype = rt.PT_SubmitJobBG
	}
	handle, err = client.do(funcname, data, datatype)
	return
}

func (client *Client) DoCron(funcname string, cronExpr string, funcParam []byte) (string, error) {
	cf := strings.Fields(cronExpr)
	expLen := len(cf)
	switch expLen {
	case 5:
		return client.doCron(funcname, cronExpr, funcParam)
	case 6:
		if cf[5] == "*" {
			return client.doCron(funcname, strings.Join([]string{cf[0], cf[1], cf[2], cf[3], cf[4]}, " "), funcParam)
		} else {
			epoch, err := ToEpoch(strings.Join([]string{cf[0], cf[1], cf[2], cf[3], cf[5]}, " "))
			if err != nil {
				return "", err
			}
			return client.DoAt(funcname, epoch, funcParam)
		}
	default:
		return "", errors.New("invalid cron expression")

	}
}

func (client *Client) doCron(funcname string, cronExpr string, funcParam []byte) (handle string, err error) {
	ce, err := rt.NewCronSchedule(cronExpr)
	if err != nil {
		return "", err
	}
	dbyt := []byte(fmt.Sprintf("%v%v", string(ce.Bytes()), string(funcParam)))
	handle, err = client.do(funcname, dbyt, rt.PT_SubmitJobSched)
	return
}

func (client *Client) DoAt(funcname string, epoch int64, funcParam []byte) (handle string, err error) {
	if client.conn == nil {
		return "", ErrLostConn
	}
	dbyt := []byte(fmt.Sprintf("%v\x00%v", epoch, string(funcParam)))
	handle, err = client.do(funcname, dbyt, rt.PT_SubmitJobEpoch)
	return
}

// Get job status from job server.
func (client *Client) Status(handle string) (status *Status, err error) {
	if client.conn == nil {
		return nil, ErrLostConn
	}
	var mutex sync.Mutex
	mutex.Lock()
	client.innerHandler.put("s"+handle, func(resp *Response) {
		defer mutex.Unlock()
		var err error
		status, err = resp._status()
		if err != nil {
			client.err(err)
		}
	})
	req := getRequest()
	req.DataType = rt.PT_GetStatus
	req.Data = []byte(handle)
	client.write(req)
	mutex.Lock()
	return
}

// Echo.
func (client *Client) Echo(data []byte) (echo []byte, err error) {
	if client.conn == nil {
		return nil, ErrLostConn
	}
	var mutex sync.Mutex
	mutex.Lock()
	client.innerHandler.put("e", func(resp *Response) {
		echo = resp.Data
		mutex.Unlock()
	})
	req := getRequest()
	req.DataType = rt.PT_EchoReq
	req.Data = data
	client.write(req)
	mutex.Lock()
	return
}

// Close connection
func (client *Client) Close() (err error) {
	client.Lock()
	defer client.Unlock()
	if client.conn != nil {
		err = client.conn.Close()
		client.conn = nil
	}
	return
}
