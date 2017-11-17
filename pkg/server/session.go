package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	. "github.com/appscode/g2/pkg/runtime"
	"github.com/appscode/go/log"
)

type session struct {
	sessionId int64
	w         *Worker
	c         *Client
}

func (s *session) getWorker(sessionId int64, inbox chan []byte, conn net.Conn) *Worker {
	if s.w != nil {
		return s.w
	}

	s.w = &Worker{
		Conn: conn, status: wsSleep, Session: Session{SessionId: sessionId,
			in: inbox, ConnectAt: time.Now()}, runningJobs: make(map[string]*Job),
		canDo: make(map[string]int32)}

	return s.w
}

func (se *session) handleConnection(s *Server, conn net.Conn) {
	sessionId := s.allocSessionId()
	inbox := make(chan []byte, 200)
	out := make(chan []byte, 200)
	defer func() {
		if se.w != nil || se.c != nil {
			e := &event{tp: ctrlCloseSession, fromSessionId: sessionId,
				result: createResCh()}
			s.protoEvtCh <- e
			<-e.result
			close(inbox) //notify writer to quit
		}
	}()
	log.Debugf("new session with sessionId %v and address: %v", sessionId, conn.RemoteAddr())

	go queueingWriter(inbox, out)
	go writer(conn, out)

	r := bufio.NewReaderSize(conn, 256*1024)
	//todo:1. reuse event's result channel, create less garbage.
	//2. heavily rely on goroutine switch, send reply in EventLoop can make it faster, but logic is not that clean
	//so i am not going to change it right now, maybe never
	fb, err := r.Peek(1)
	if err != nil {
		log.Errorln(err)
		return
	}
	if fb[0] == byte(0) {
		se.handleBinaryConnection(s, conn, r, sessionId, inbox)
	} else {
		se.handleAdminConnection(s, conn, r, sessionId, inbox)
	}

}

func (se *session) handleBinaryConnection(s *Server, conn net.Conn, r *bufio.Reader, sessionId int64, inbox chan []byte) {

	for {
		tp, buf, err := ReadMessage(r)
		if err != nil {
			log.Debugf("%v with sessionId %v", err, sessionId)
			return
		}
		args, ok := decodeArgs(tp, buf)
		if !ok {
			log.Debugf("protocol: %v argc not match details: %s", tp.String(), string(buf))
			return
		}

		log.Debugf("incoming<= sessionId: %v protocol: %v len(args): %v details: %s", sessionId, tp.String(), len(args), string(buf))

		switch tp {
		case PT_CanDo:
			se.w = se.getWorker(sessionId, inbox, conn)
			s.protoEvtCh <- &event{tp: tp, args: &Tuple{
				t0: se.w, t1: string(args[0])}}
		case PT_CanDoTimeout:
			se.w = se.getWorker(sessionId, inbox, conn)
			s.protoEvtCh <- &event{tp: tp, args: &Tuple{
				t0: se.w, t1: string(args[0]), t2: args[1]}}
		case PT_CantDo:
			s.protoEvtCh <- &event{tp: tp, fromSessionId: sessionId,
				args: &Tuple{t0: string(args[0])}}
		case PT_EchoReq:
			sendReply(inbox, PT_EchoRes, [][]byte{buf})
		case PT_PreSleep:
			se.w = se.getWorker(sessionId, inbox, conn)
			s.protoEvtCh <- &event{tp: tp, args: &Tuple{t0: se.w}, fromSessionId: sessionId}
		case PT_SetClientId:
			se.w = se.getWorker(sessionId, inbox, conn)
			s.protoEvtCh <- &event{tp: tp, args: &Tuple{t0: se.w, t1: string(args[0])}}
		case PT_GrabJobUniq:
			if se.w == nil {
				log.Errorf("can't perform %s, need send CAN_DO first", tp.String())
				return
			}
			e := &event{tp: tp, fromSessionId: sessionId,
				result: createResCh()}
			s.protoEvtCh <- e
			job := (<-e.result).(*Job)
			if job == nil {
				log.Debugf("sessionId %v has no job", sessionId)
				sendReplyResult(inbox, nojobReply)
				break
			}

			sendReply(inbox, PT_JobAssignUniq, [][]byte{
				[]byte(job.Handle), []byte(job.FuncName), []byte(job.Id), job.Data})
		case PT_SubmitJobLow, PT_SubmitJob, PT_SubmitJobHigh, PT_SubmitJobLowBG, PT_SubmitJobBG, PT_SubmitJobHighBG:
			if se.c == nil {
				se.c = &Client{Session: Session{SessionId: sessionId, in: inbox,
					ConnectAt: time.Now()}}
			}
			e := &event{tp: tp,
				args:   &Tuple{t0: se.c, t1: args[0], t2: args[1], t3: args[2]},
				result: createResCh(),
			}
			s.protoEvtCh <- e
			handle := <-e.result
			sendReply(inbox, PT_JobCreated, [][]byte{[]byte(handle.(string))})
		case PT_SubmitJobSched:
			if se.c == nil {
				se.c = &Client{Session: Session{SessionId: sessionId, in: inbox,
					ConnectAt: time.Now()}}
			}
			e := &event{tp: tp,
				args:   &Tuple{t0: se.c, t1: args[0], t2: args[1], t3: args[2], t4: args[3], t5: args[4], t6: args[5], t7: args[6], t8: args[7]},
				result: createResCh(),
			}
			s.protoEvtCh <- e
			shcedJobId := <-e.result
			sendReply(inbox, PT_JobCreated, [][]byte{[]byte(shcedJobId.(string))})
		case PT_SubmitJobEpoch:
			if se.c == nil {
				se.c = &Client{Session: Session{SessionId: sessionId, in: inbox,
					ConnectAt: time.Now()}}
			}
			e := &event{tp: tp,
				args:   &Tuple{t0: se.c, t1: args[0], t2: args[1], t3: args[2], t4: args[3]},
				result: createResCh(),
			}
			s.protoEvtCh <- e
			jobId := <-e.result
			sendReply(inbox, PT_JobCreated, [][]byte{[]byte(jobId.(string))})
		case PT_GetStatus:
			e := &event{tp: tp, args: &Tuple{t0: args[0]},
				result: createResCh()}
			s.protoEvtCh <- e

			resp := (<-e.result).(*Tuple)
			sendReply(inbox, PT_StatusRes, [][]byte{resp.t0.([]byte),
				bool2bytes(resp.t1), bool2bytes(resp.t2),
				int2bytes(resp.t3),
				int2bytes(resp.t4)})
		case PT_WorkData, PT_WorkWarning, PT_WorkStatus, PT_WorkComplete,
			PT_WorkFail, PT_WorkException:
			if se.w == nil {
				log.Errorf("can't perform %s, need send CAN_DO first", tp.String())
				return
			}
			s.protoEvtCh <- &event{tp: tp, args: &Tuple{t0: args},
				fromSessionId: sessionId}
		default:
			log.Warningf("not support type %s", tp.String())
		}
	}
}

func (se *session) handleAdminConnection(s *Server, conn net.Conn, r *bufio.Reader, sessionId int64, inbox chan []byte) {
	for {
		rcv, err := r.ReadBytes('\n')
		if err != nil {
			sendTextReply(inbox, fmt.Sprintf("Error: %v\n", err))
			log.Errorln(err)
			return
		}
		trimedRcv := strings.TrimSpace(string(rcv))
		if trimedRcv == "" {
			continue
		}
		ap, arg := ParseTextMessage(trimedRcv)
		switch ap {
		case AP_Show, AP_Create, AP_Drop, AP_MaxQueue, AP_GetPid, AP_Shutdown, AP_Verbose, AP_Version:
			sendTextError(inbox, fmt.Sprintf("command `%s` is currently unimplemented", ap))
		case AP_Cancel:
			if IsValidCronJobHandle(arg) {
				err := s.DeleteCronJob(&CronJob{Handle: arg})
				if err != nil {
					log.Errorln(err)
					sendTextError(inbox, err.Error())
					continue
				}
				log.Debugf("job `%v` successfully cancelled.\n", arg)
				sendTextOK(inbox)
			} else {
				log.Errorf("invalid handle `%v`\n", arg)
				sendTextError(inbox, fmt.Sprintf("Invalid handle `%v`, valid schedule job handle should start with `S:`\n", arg))
			}
		case AP_Status:
			resp := ""
			for fnName, v := range s.funcWorker {
				runningCnt := 0
				for _, j := range s.jobs {
					if fnName == j.FuncName && j.Running {
						runningCnt++
					}
				}
				resp += fmt.Sprintf("%v\t%v\t%v\t%v\n", fnName, v.jobs.Len(), runningCnt, v.workers.Len())
			}
			resp += ".\n"
			sendTextReply(inbox, resp)
		case AP_PRIORITY_STATUS:
			resp := ""
			for fnName, v := range s.funcWorker {
				high := 0
				normal := 0
				low := 0
				for _, j := range s.jobs {
					if fnName == j.FuncName && !j.Running {
						switch j.Priority {
						case JobHigh:
							high++
						case JobNormal:
							normal++
						case JobLow:
							low++
						}
					}
				}
				resp += fmt.Sprintf("%v\t%v\t%v\t%v\t%v\n", fnName, high, normal, low, v.workers.Len())
			}
			resp += ".\n"
			sendTextReply(inbox, resp)
		case AP_Workers:
			resp := ""
			for _, v := range s.worker {
				resp += fmt.Sprintf("%v %v %v : ", "-", v.Conn.RemoteAddr().String(), v.workerId)
				isFirst := true
				for fnName := range v.canDo {
					if !isFirst {
						resp += " "
					}
					isFirst = false
					resp += fmt.Sprintf("%v", fnName)
				}
				resp += "\n"
			}
			resp += ".\n"
			sendTextReply(inbox, resp)
		default:
			log.Errorf("Invalid command `%s`\n", ap)
			sendTextError(inbox, fmt.Sprintf("Invalid command `%s`\n", ap))
		}
	}
}
