package server

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/g2/pkg/runtime"
	"github.com/appscode/go/log"
)

var (
	invalidMagic = errors.New("invalid magic")
	invalidArg   = errors.New("invalid argument")
)

type AP string

const (
	AP_Workers         AP = "workers"
	AP_Status          AP = "status"
	AP_Cancel          AP = "cancel-job"
	AP_Show            AP = "show"
	AP_Create          AP = "create"
	AP_Drop            AP = "drop"
	AP_MaxQueue        AP = "maxqueue"
	AP_GetPid          AP = "getpid"
	AP_Shutdown        AP = "shutdown"
	AP_Verbose         AP = "verbose"
	AP_Version         AP = "version"
	AP_PRIORITY_STATUS AP = "prioritystatus"
)

const (
	ctrlCloseSession = 1000 + iota
	ctrlGetJob
	ctrlGetWorker
	ctrlGetCronJob
)

var (
	startJid     int64 = 0
	handlePrefix string
	respMagic    = []byte(runtime.ResStr)
	idCh         = make(chan string, 50)
)

func init() {
	validProtocolDef()
	hn, err := os.Hostname()
	if err != nil {
		hn = os.Getenv("HOSTNAME")
	}

	if hn == "" {
		hn = "localhost"
	}

	//cache prefix
	handlePrefix = fmt.Sprintf("-%s:-%d-%d-", hn, os.Getpid(), time.Now().Unix())
	go func() {
		for {
			idCh <- genJid()
		}
	}()
}

func genJid() string {
	startJid++
	return handlePrefix + strconv.FormatInt(startJid, 10)
}

func allocJobId() string {
	return runtime.JobPrefix + <-idCh
}

func allocSchedJobId() string {
	return runtime.CronJobPrefix + <-idCh
}

func IsValidJobHandle(handle string) bool {
	return strings.HasPrefix(handle, runtime.JobPrefix)
}

func IsValidCronJobHandle(handle string) bool {
	return strings.HasPrefix(handle, runtime.CronJobPrefix)
}

type event struct {
	tp            runtime.PT
	args          *Tuple
	result        chan interface{}
	fromSessionId int64
	handle        string
}

type jobworkermap struct {
	workers *list.List
	jobs    *list.List
}

type Tuple struct {
	t0, t1, t2, t3, t4, t5, t6, t7, t8 interface{}
}

func decodeArgs(cmd runtime.PT, buf []byte) ([][]byte, bool) {
	argc := cmd.ArgCount()
	//log.Debug("cmd:", runtime.CmdDescription(cmd), "details:", buf)
	if argc == 0 {
		return nil, true
	}

	args := make([][]byte, 0, argc)

	if argc == 1 {
		args = append(args, buf)
		return args, true
	}
	endPos := 0
	cnt := 0
	for ; cnt < argc-1 && endPos < len(buf); cnt++ {
		startPos := endPos
		pos := bytes.IndexByte(buf[startPos:], 0x0)
		if pos == -1 {
			log.Warning("invalid protocol")
			return nil, false
		}
		endPos = startPos + pos
		args = append(args, buf[startPos:endPos])
		endPos++
	}
	args = append(args, buf[endPos:]) //option data
	cnt++

	if cnt != argc {
		log.Errorf("argc not match %d-%d", argc, len(args))
		return nil, false
	}

	return args, true
}

func sendReply(out chan []byte, tp runtime.PT, data [][]byte) {
	out <- constructReply(tp, data)
}

func sendTextReply(out chan []byte, resp string) {
	out <- []byte(resp)
}

func sendReplyResult(out chan []byte, data []byte) {
	out <- data
}

func sendTextOK(out chan []byte) {
	out <- []byte("OK\n")
}

func sendTextError(out chan []byte, errmsg string) {
	out <- []byte(fmt.Sprintf("Error: %s\n", errmsg))
}

func sendTimeoutException(out chan []byte, handle string, exception string) {
	data := [][]byte{[]byte(handle), []byte(exception)}
	out <- constructReply(runtime.PT_WorkException, data)
}

func constructReply(tp runtime.PT, data [][]byte) []byte {
	buf := &bytes.Buffer{}
	buf.Write(respMagic)

	err := binary.Write(buf, binary.BigEndian, tp.Uint32())
	if err != nil {
		panic("should never happend")
	}

	length := 0
	for i, arg := range data {
		length += len(arg)
		if i < len(data)-1 {
			length += 1
		}
	}

	err = binary.Write(buf, binary.BigEndian, uint32(length))
	if err != nil {
		panic("should never happend")
	}

	for i, arg := range data {
		buf.Write(arg)
		if i < len(data)-1 {
			buf.WriteByte(0x00)
		}
	}

	return buf.Bytes()
}

func bytes2str(o interface{}) string {
	return string(o.([]byte))
}

func bool2bytes(b interface{}) []byte {
	if b.(bool) {
		return []byte{'1'}
	}

	return []byte{'0'}
}

func byte2strWithFixSpace(o interface{}) string {
	str := bytes2str(o)
	if str == "" {
		return "*"
	}
	return str
}

func int2bytes(n interface{}) []byte {
	return []byte(strconv.Itoa(n.(int)))
}

func ReadMessage(r io.Reader) (runtime.PT, []byte, error) {
	_, tp, size, err := readHeader(r)
	if err != nil {
		return tp, nil, err
	}

	if size == 0 {
		return tp, nil, nil
	}

	buf := make([]byte, size)
	_, err = io.ReadFull(r, buf)

	return tp, buf, err
}

func ParseTextMessage(msg string) (AP, string) {
	dt := AP(strings.Fields(msg)[0])
	return dt, strings.TrimSpace(msg[len(dt):])
}

func readHeader(r io.Reader) (magic uint32, tp runtime.PT, size uint32, err error) {
	magic, err = readUint32(r)
	if err != nil {
		return
	}

	if magic != runtime.Req && magic != runtime.Res {
		log.Debugf("magic not match 0x%x", magic)
		err = invalidMagic
		return
	}

	var cmd uint32
	cmd, err = readUint32(r)
	if err != nil {
		return
	}
	tp, err = runtime.NewPT(cmd)
	if err != nil {
		return
	}
	size, err = readUint32(r)
	return
}

func clearOutbox(outbox chan []byte) {
	for {
		select {
		case b, ok := <-outbox:
			if !ok { //channel is closed
				return
			}

			_ = b //compiler bug (tip)
		}
	}
}

func writer(conn net.Conn, outbox chan []byte) {
	defer func() {
		conn.Close()
		clearOutbox(outbox) //incase reader is blocked
	}()

	b := bytes.NewBuffer(nil)

	for {
		select {
		case msg, ok := <-outbox:
			if !ok {
				return
			}
			b.Write(msg)
			for n := len(outbox); n > 0; n-- {
				b.Write(<-outbox)
			}

			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			_, err := conn.Write(b.Bytes())
			if err != nil {
				return
			}
			b.Reset()
		}
	}
}

func readUint32(r io.Reader) (uint32, error) {
	var value uint32
	err := binary.Read(r, binary.BigEndian, &value)
	return value, err
}

func validProtocolDef() {
	if runtime.PT_CanDo != 1 || runtime.PT_SubmitJobEpoch != 36 { //protocol check
		panic("protocol define not match")
	}
}

func createResCh() chan interface{} {
	return make(chan interface{}, 1)
}

func LocalIP() (net.IP, error) {
	tt, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, t := range tt {
		if strings.Contains(t.Name, "lo") { //exclude loop back address
			continue
		}

		aa, err := t.Addrs()
		if err != nil {
			return nil, err
		}
		for _, a := range aa {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if ipnet.IP.IsLoopback() {
				continue
			}

			v4 := ipnet.IP.To4()
			if v4 == nil || v4[0] == 127 { // loopback address
				continue
			}
			//log.Println(a)
			return v4, nil
		}
	}
	return nil, errors.New("cannot find local IP address")
}

func cmd2Priority(cmd runtime.PT) int {
	switch cmd {
	case runtime.PT_SubmitJobHigh, runtime.PT_SubmitJobHighBG:
		return runtime.PRIORITY_HIGH
	}
	return runtime.PRIORITY_LOW
}

func isBackGround(cmd runtime.PT) bool {
	switch cmd {
	case runtime.PT_SubmitJobLowBG, runtime.PT_SubmitJobBG, runtime.PT_SubmitJobHighBG:
		return true
	}
	return false
}
