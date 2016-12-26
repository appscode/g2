package server

import (
	"bufio"
	"net"
	"time"

	. "github.com/appscode/g2/pkg/runtime"
	"github.com/appscode/log"
)

type session struct {
	sessionId int64
	w         *Worker
	c         *Client
}

func (self *session) getWorker(sessionId int64, inbox chan []byte, conn net.Conn) *Worker {
	if self.w != nil {
		return self.w
	}

	self.w = &Worker{
		Conn: conn, status: wsSleep, Session: Session{SessionId: sessionId,
			in: inbox, ConnectAt: time.Now()}, runningJobs: make(map[string]*Job),
		canDo: make(map[string]bool)}

	return self.w
}

func (self *session) handleConnection(s *Server, conn net.Conn) {
	sessionId := s.allocSessionId()

	inbox := make(chan []byte, 200)
	out := make(chan []byte, 200)
	defer func() {
		if self.w != nil || self.c != nil {
			e := &event{tp: ctrlCloseSession, fromSessionId: sessionId,
				result: createResCh()}
			s.protoEvtCh <- e
			<-e.result
			close(inbox) //notify writer to quit
		}
	}()

	log.Debug("new sessionId", sessionId, "address:", conn.RemoteAddr())

	go queueingWriter(inbox, out)
	go writer(conn, out)

	r := bufio.NewReaderSize(conn, 256*1024)
	//todo:1. reuse event's result channel, create less garbage.
	//2. heavily rely on goroutine switch, send reply in EventLoop can make it faster, but logic is not that clean
	//so i am not going to change it right now, maybe never

	for {
		tp, buf, err := ReadMessage(r)
		if err != nil {
			log.Debug(err, "sessionId", sessionId)
			return
		}

		args, ok := decodeArgs(tp, buf)
		if !ok {
			log.Debug("tp:", tp.String(), "argc not match", "details:", string(buf))
			return
		}

		log.Debug("sessionId", sessionId, "tp:", tp.String(), "len(args):", len(args), "details:", string(buf))

		switch tp {
		case PT_CanDo, PT_CanDoTimeout: //todo: CAN_DO_TIMEOUT timeout support
			self.w = self.getWorker(sessionId, inbox, conn)
			s.protoEvtCh <- &event{tp: tp, args: &Tuple{
				t0: self.w, t1: string(args[0])}}
		case PT_CantDo:
			s.protoEvtCh <- &event{tp: tp, fromSessionId: sessionId,
				args: &Tuple{t0: string(args[0])}}
		case PT_EchoReq:
			sendReply(inbox, PT_EchoRes, [][]byte{buf})
		case PT_PreSleep:
			self.w = self.getWorker(sessionId, inbox, conn)
			s.protoEvtCh <- &event{tp: tp, args: &Tuple{t0: self.w}, fromSessionId: sessionId}
		case PT_SetClientId:
			self.w = self.getWorker(sessionId, inbox, conn)
			s.protoEvtCh <- &event{tp: tp, args: &Tuple{t0: self.w, t1: string(args[0])}}
		case PT_GrabJobUniq:
			if self.w == nil {
				log.Errorf("can't perform %s, need send CAN_DO first", tp.String())
				return
			}
			e := &event{tp: tp, fromSessionId: sessionId,
				result: createResCh()}
			s.protoEvtCh <- e
			job := (<-e.result).(*Job)
			if job == nil {
				log.Debug("sessionId", sessionId, "no job")
				sendReplyResult(inbox, nojobReply)
				break
			}

			//log.Debugf("%+v", job)
			sendReply(inbox, PT_JobAssignUniq, [][]byte{
				[]byte(job.Handle), []byte(job.FuncName), []byte(job.Id), job.Data})
		case PT_SubmitJob, PT_SubmitJobLowBG, PT_SubmitJobLow:
			if self.c == nil {
				self.c = &Client{Session: Session{SessionId: sessionId, in: inbox,
					ConnectAt: time.Now()}}
			}
			e := &event{tp: tp,
				args:   &Tuple{t0: self.c, t1: args[0], t2: args[1], t3: args[2]},
				result: createResCh(),
			}
			s.protoEvtCh <- e
			handle := <-e.result
			sendReply(inbox, PT_JobCreated, [][]byte{[]byte(handle.(string))})
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
			if self.w == nil {
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
