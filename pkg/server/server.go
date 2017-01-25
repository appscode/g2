package server

import (
	"container/list"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/appscode/g2/pkg/runtime"
	"github.com/appscode/g2/pkg/storage"
	"github.com/appscode/log"
	"github.com/ngaut/stats"
	lberror "github.com/syndtr/goleveldb/leveldb/errors"
	"gopkg.in/robfig/cron.v2"
)

type Server struct {
	protoEvtCh     chan *event
	ctrlEvtCh      chan *event
	funcWorker     map[string]*jobworkermap //function worker
	worker         map[int64]*Worker
	client         map[int64]*Client
	jobs           map[string]*Job
	startSessionId int64
	opCounter      map[PT]int64
	store          storage.Db
	forwardReport  int64
	cronSvc        *cron.Cron
}

var ( //const replys, to avoid building it every time
	wakeupReply = constructReply(PT_Noop, nil)
	nojobReply  = constructReply(PT_NoJob, nil)
)

func NewServer(store storage.Db) *Server {
	return &Server{
		funcWorker: make(map[string]*jobworkermap),
		protoEvtCh: make(chan *event, 100),
		ctrlEvtCh:  make(chan *event, 100),
		worker:     make(map[int64]*Worker),
		client:     make(map[int64]*Client),
		jobs:       make(map[string]*Job),
		opCounter:  make(map[PT]int64),
		store:      store,
		cronSvc:    cron.New(),
	}
}

func (s *Server) loadAllJobs() {
	jobs, err := s.store.GetJobs()
	if err != nil {
		log.Error(err)
		return
	}
	for _, j := range jobs {
		j.ProcessBy = 0 //no body handle it now
		j.CreateBy = 0  //clear
		log.Debugf("handle: %v\tfunc: %v\tbg: %v", j.Handle, j.FuncName, j.IsBackGround)
		s.doAddJob(j)
	}
}

func (s *Server) loadAllCronJobs() {
	schedJobs, err := s.store.GetCronJobs()
	if err != nil {
		log.Error(err)
		return
	}
	for _, sj := range schedJobs {
		if epoch, ok := s.ExpressionToEpoch(sj.Expression); ok {
			log.Debugf("handle: %v func: %v schedule: %v", sj.Handle, sj.JobTemplete.FuncName, time.Unix(epoch, 0).Local())
			s.doAddEpochJob(sj, epoch)
		} else {
			log.Debugf("handle: %v func: %v expr: %v", sj.Handle, sj.JobTemplete.FuncName, sj.Expression)
			s.doAddCronJob(sj)
		}
	}
}

func (s *Server) Start(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	go s.EvtLoop()
	log.Debug("listening on", addr)

	go registerWebHandler(s)
	go s.WatcherLoop()
	if s.cronSvc != nil {
		s.cronSvc.Start()
	}
	//load background jobs from storage
	if s.store != nil {
		s.loadAllJobs()
		s.loadAllCronJobs()
	}

	for {
		conn, err := ln.Accept()
		if err != nil { // handle error
			continue
		}

		session := &session{}
		go session.handleConnection(s, conn)
	}
}

func (s *Server) addWorker(l *list.List, w *Worker) {
	for it := l.Front(); it != nil; it = it.Next() {
		if it.Value.(*Worker).SessionId == w.SessionId {
			log.Warning("already add")
			return
		}
	}

	l.PushBack(w) //add to worker list
}

func (s *Server) removeWorker(l *list.List, sessionId int64) {
	for it := l.Front(); it != nil; it = it.Next() {
		if it.Value.(*Worker).SessionId == sessionId {
			log.Debugf("removeWorker sessionId %d", sessionId)
			l.Remove(it)
			return
		}
	}
}

func (s *Server) removeWorkerBySessionId(sessionId int64) {
	for _, jw := range s.funcWorker {
		s.removeWorker(jw.workers, sessionId)
	}
	delete(s.worker, sessionId)
}

func (s *Server) handleCanDo(funcName string, w *Worker) {
	w.canDo[funcName] = true
	jw := s.getJobWorkPair(funcName)
	s.addWorker(jw.workers, w)
	s.worker[w.SessionId] = w
}

func (s *Server) getJobWorkPair(funcName string) *jobworkermap {
	jw, ok := s.funcWorker[funcName]
	if !ok { //create list
		jw = &jobworkermap{workers: list.New(), jobs: list.New()}
		s.funcWorker[funcName] = jw
	}

	return jw
}

func (s *Server) add2JobWorkerQueue(j *Job) {
	jw := s.getJobWorkPair(j.FuncName)
	jw.jobs.PushBack(j)
}

func (s *Server) doAddJob(j *Job) {
	j.ProcessBy = 0 //nobody handle it right now
	s.add2JobWorkerQueue(j)
	s.jobs[j.Handle] = j
	s.wakeupWorker(j.FuncName)
}

func (s *Server) doAddAndPersistJob(j *Job) {
	// persistent job
	log.Debugf("add job with handle: %v func: %v bg: %v", j.Handle, j.FuncName, j.IsBackGround)
	if s.store != nil {
		if err := s.store.AddJob(j); err != nil {
			log.Warning(err)
		}
	}
	s.doAddJob(j)
}

func (s *Server) doAddCronJob(sj *CronJob) cron.EntryID {
	scdT, err := NewCronSchedule(sj.Expression)
	if err != nil {
		log.Errorln(err)
		return cron.EntryID(0)
	}
	return s.cronSvc.Schedule(
		scdT.Schedule(),
		cron.FuncJob(
			func() {
				jb := &Job{
					Handle:       allocJobId(),
					Id:           sj.JobTemplete.Id,
					Data:         sj.JobTemplete.Data,
					CreateAt:     time.Now(),
					CreateBy:     sj.JobTemplete.CreateBy,
					FuncName:     sj.JobTemplete.FuncName,
					Priority:     sj.JobTemplete.Priority,
					IsBackGround: sj.JobTemplete.IsBackGround,
					CronHandle:   sj.Handle,
				}
				err := s.store.UpdateCronJob(sj.Handle, map[string]interface{}{
					"Next": scdT.Schedule().Next(time.Now()),
					"Prev": time.Now(),
				})
				if err != nil {
					log.Error(err)
				}
				s.doAddAndPersistJob(jb)
			}))
}

func (s *Server) doAddEpochJob(cj *CronJob, epoch int64) {
	j := &Job{
		Handle:       allocJobId(),
		Id:           cj.JobTemplete.Id,
		Data:         cj.JobTemplete.Data,
		CreateAt:     time.Now(),
		CreateBy:     cj.JobTemplete.CreateBy,
		FuncName:     cj.JobTemplete.FuncName,
		Priority:     cj.JobTemplete.Priority,
		IsBackGround: cj.JobTemplete.IsBackGround,
	}
	after := epoch - time.Now().UTC().Unix()
	if after < 0 {
		after = 0
	}
	time.AfterFunc(time.Second*time.Duration(after), func() {
		s.doAddAndPersistJob(j)
		err := s.DeleteCronJob(cj)
		if err != nil {
			log.Errorln(err)
		}
	})

}

func (s *Server) popJob(sessionId int64) (j *Job) {
	for funcName, cando := range s.worker[sessionId].canDo {
		if !cando {
			continue
		}

		if wj, ok := s.funcWorker[funcName]; ok {
			if wj.jobs.Len() == 0 {
				continue
			}

			job := wj.jobs.Front()
			wj.jobs.Remove(job)
			j = job.Value.(*Job)
			return
		}
	}

	return
}

func (s *Server) wakeupWorker(funcName string) bool {
	wj, ok := s.funcWorker[funcName]
	if !ok || wj.jobs.Len() == 0 || wj.workers.Len() == 0 {
		return false
	}

	for it := wj.workers.Front(); it != nil; it = it.Next() {
		w := it.Value.(*Worker)
		if w.status != wsSleep {
			continue
		}

		log.Debug("wakeup sessionId ", w.SessionId)

		w.Send(wakeupReply)
		return true
	}

	return false
}

func (s *Server) checkAndRemoveJob(tp PT, j *Job) {
	switch tp {
	case PT_WorkComplete:
		s.removeJob(j, true)
	case PT_WorkException, PT_WorkFail:
		s.removeJob(j, false)
	}
}

func (s *Server) removeJob(j *Job, isSuccess bool) {
	delete(s.jobs, j.Handle)
	delete(s.worker[j.ProcessBy].runningJobs, j.Handle)
	if j.IsBackGround {
		log.Debugf("done job: %v", j.Handle)
		if s.store != nil {
			if err := s.store.DeleteJob(j, isSuccess); err != nil {
				log.Warning(err)
			}
		}
	}
}

func (s *Server) handleCloseSession(e *event) error {
	sessionId := e.fromSessionId
	if w, ok := s.worker[sessionId]; ok {
		if sessionId != w.SessionId {
			log.Fatalf("sessionId not match %d-%d, bug found", sessionId, w.SessionId)
		}
		s.removeWorkerBySessionId(w.SessionId)

		//reschedule these jobs, so other workers can handle it
		for handle, j := range w.runningJobs {
			if handle != j.Handle {
				log.Fatal("handle not match %d-%d", handle, j.Handle)
			}
			j.Running = false
			s.doAddJob(j)
		}
	}
	if c, ok := s.client[sessionId]; ok {
		log.Debug("removeClient with sessionId ", sessionId)
		delete(s.client, c.SessionId)
	}
	e.result <- true //notify close finish

	return nil
}

func (s *Server) handleGetWorker(e *event) (err error) {
	var buf []byte
	defer func() {
		e.result <- string(buf)
	}()
	cando := e.args.t0.(string)
	log.Debug("get worker ", cando)
	if len(cando) == 0 {
		workers := make([]*Worker, 0, len(s.worker))
		for _, v := range s.worker {
			workers = append(workers, v)
		}
		buf, err = json.Marshal(workers)
		if err != nil {
			log.Error(err)
			return err
		}
		return nil
	}

	if jw, ok := s.funcWorker[cando]; ok {
		log.Debugf("func: %v #workers: %v", cando, jw.workers.Len())
		workers := make([]*Worker, 0, jw.workers.Len())
		for it := jw.workers.Front(); it != nil; it = it.Next() {
			workers = append(workers, it.Value.(*Worker))
		}
		buf, err = json.Marshal(workers)
		if err != nil {
			log.Error(err)
			return err
		}
		return nil
	}

	return
}

func (s *Server) handleGetJob(e *event) (err error) {
	log.Debug("get jobs ", e.handle)
	var buf []byte
	defer func() {
		e.result <- string(buf)
	}()

	if len(e.handle) == 0 {
		jobs := []*Job{}
		for _, v := range s.jobs {
			jobs = append(jobs, v)
		}
		buf, err = json.Marshal(jobs)
		if err != nil {
			log.Error(err)
			return err
		}
		return nil
	}

	if job, ok := s.jobs[e.handle]; ok {
		buf, err = json.Marshal(job)
		if err != nil {
			log.Error(err)
			return err
		}
		return nil
	}

	return
}

func (s *Server) handleGetCronJob(e *event) (err error) {
	log.Debug("get cronjobs ", e.handle)
	var buf []byte
	defer func() {
		e.result <- string(buf)
	}()

	if len(e.handle) == 0 {
		cjs, err := s.store.GetCronJobs()
		if err != nil {
			log.Error(err)
			return err
		}
		buf, err = json.Marshal(cjs)
		if err != nil {
			log.Error(err)
			return err
		}
		return nil
	}
	cj, err := s.store.GetCronJob(e.handle)
	if err != nil {
		log.Error(err)
		return err
	}
	data, err := json.Marshal(cj)
	if err != nil {
		log.Error(err)
		return err
	}
	buf = []byte(data)
	return
}

func (s *Server) handleCtrlEvt(e *event) (err error) {
	//args := e.args
	switch e.tp {
	case ctrlCloseSession:
		return s.handleCloseSession(e)
	case ctrlGetJob:
		return s.handleGetJob(e)
	case ctrlGetWorker:
		return s.handleGetWorker(e)
	case ctrlGetCronJob:
		return s.handleGetCronJob(e)
	default:
		log.Warningf("%s, %d", e.tp, e.tp)
	}

	return nil
}

func (s *Server) handleSubmitJob(e *event) {
	args := e.args
	c := args.t0.(*Client)
	s.client[c.SessionId] = c
	funcName := bytes2str(args.t1)
	j := &Job{
		Handle:       allocJobId(),
		Id:           bytes2str(args.t2),
		Data:         args.t3.([]byte),
		CreateAt:     time.Now(),
		CreateBy:     c.SessionId,
		FuncName:     funcName,
		Priority:     cmd2Priority(e.tp),
		IsBackGround: isBackGround(e.tp),
	}
	//log.Debugf("%v, job handle %v, %s", CmdDescription(e.tp), j.Handle, string(j.Data))
	e.result <- j.Handle
	s.doAddAndPersistJob(j)
}

func (s *Server) handleCronJob(e *event) {
	args := e.args
	c := args.t0.(*Client)
	s.client[c.SessionId] = c
	funcName := bytes2str(args.t1)
	sst, err := NewCronSchedule(fmt.Sprintf("%v %v %v %v %v",
		byte2strWithFixSpace(args.t3),
		byte2strWithFixSpace(args.t4),
		byte2strWithFixSpace(args.t5),
		byte2strWithFixSpace(args.t6),
		byte2strWithFixSpace(args.t7)),
	)
	if err != nil {
		log.Errorln(err)
		return
	}
	sj := &CronJob{
		JobTemplete: Job{
			Id:           bytes2str(args.t2),
			Data:         args.t8.([]byte),
			CreateAt:     time.Now(),
			CreateBy:     c.SessionId,
			FuncName:     funcName,
			Priority:     cmd2Priority(e.tp),
			IsBackGround: true,
		},
		Expression: sst.Expr(),
	}
	sj.Handle = allocSchedJobId()
	e.result <- sj.Handle
	// persistent Cron Job
	id := s.doAddCronJob(sj)
	sj.CronEntryID = int(id)
	scdT, err := NewCronSchedule(sj.Expression)
	if err != nil {
		log.Errorln(err)
	}
	sj.Next = scdT.Schedule().Next(time.Now())
	if s.store != nil {
		if err := s.store.AddCronJob(sj); err != nil {
			log.Errorln(err)
		}
	}
	log.Debugf("add cron job with handle: %v func: %v expr: %v", sj.Handle, sj.JobTemplete.FuncName, sj.Expression)
}

func (s *Server) handleSubmitEpochJob(e *event) {
	args := e.args
	c := args.t0.(*Client)
	s.client[c.SessionId] = c
	funcName := bytes2str(args.t1)
	epochStr := bytes2str(args.t3)
	val, err := strconv.ParseInt(epochStr, 10, 64)
	if err != nil {
		log.Errorln(err)
		return
	}
	sj := &CronJob{
		JobTemplete: Job{
			Id:           bytes2str(args.t2),
			Data:         args.t4.([]byte),
			CreateAt:     time.Now(),
			CreateBy:     c.SessionId,
			FuncName:     funcName,
			Priority:     cmd2Priority(e.tp),
			IsBackGround: true,
		},
		Expression: EpochTimePrefix + epochStr,
	}
	sj.Handle = allocSchedJobId()
	e.result <- sj.Handle
	epoch, _ := s.ExpressionToEpoch(sj.Expression)
	sj.Next = time.Unix(epoch, 0)
	// persistent Cron Job
	s.doAddEpochJob(sj, val)
	if s.store != nil {
		if err := s.store.AddCronJob(sj); err != nil {
			log.Errorln(err)
		}
	}
	log.Debugf("add epoch job with handle: %v func: %v schedule: %v", sj.Handle, sj.JobTemplete.FuncName, time.Unix(epoch, 0).Local())
}

func (s *Server) handleWorkReport(e *event) {
	args := e.args
	slice := args.t0.([][]byte)
	jobhandle := bytes2str(slice[0])
	sessionId := e.fromSessionId
	j, ok := s.worker[sessionId].runningJobs[jobhandle]

	log.Debugf("%v job handle %v", e.tp, jobhandle)
	if !ok {
		log.Warningf("job information lost, %v job handle %v, %+v",
			e.tp, jobhandle, s.jobs)
		return
	}

	if j.Handle != jobhandle {
		log.Fatal("job handle not match")
	}

	if PT_WorkStatus == e.tp {
		j.Percent, _ = strconv.Atoi(string(slice[1]))
		j.Denominator, _ = strconv.Atoi(string(slice[2]))
	}

	s.checkAndRemoveJob(e.tp, j)

	//the client is not updated with status or notified when the job has completed (it is detached)
	if j.IsBackGround {
		return
	}

	//broadcast all clients, which is a really bad idea
	//for _, c := range self.client {
	//	reply := constructReply(e.tp, slice)
	//	c.Send(reply)
	//}

	//just send to original client, which is a bad idea too.
	//if need work status notification, you should create co-worker.
	//let worker send status to this co-worker
	c, ok := s.client[j.CreateBy]
	if !ok {
		log.Debug(j.Handle, "sessionId", j.CreateBy, "missing")
		return
	}

	reply := constructReply(e.tp, slice)
	c.Send(reply)
	s.forwardReport++
}

func (s *Server) handleProtoEvt(e *event) {
	args := e.args
	if e.tp < ctrlCloseSession {
		s.opCounter[e.tp]++
	}

	if e.tp >= ctrlCloseSession {
		s.handleCtrlEvt(e)
		return
	}
	switch e.tp {
	case PT_CanDo:
		w := args.t0.(*Worker)
		funcName := args.t1.(string)
		s.handleCanDo(funcName, w)
	case PT_CantDo:
		sessionId := e.fromSessionId
		funcName := args.t0.(string)
		if jw, ok := s.funcWorker[funcName]; ok {
			s.removeWorker(jw.workers, sessionId)
		}
		delete(s.worker[sessionId].canDo, funcName)
	case PT_SetClientId:
		w := args.t0.(*Worker)
		w.workerId = args.t1.(string)
	case PT_CanDoTimeout: //todo: fix timeout support, now just as CAN_DO
		w := args.t0.(*Worker)
		funcName := args.t1.(string)
		s.handleCanDo(funcName, w)
	case PT_GrabJobUniq:
		sessionId := e.fromSessionId
		w, ok := s.worker[sessionId]
		if !ok {
			log.Fatalf("unregister worker, sessionId %d", sessionId)
			break
		}

		w.status = wsRunning

		j := s.popJob(sessionId)
		if j != nil {
			j.ProcessAt = time.Now()
			j.ProcessBy = sessionId
			//track this job
			j.Running = true
			w.runningJobs[j.Handle] = j
		} else { //no job
			w.status = wsPrepareForSleep
		}
		//send job back
		e.result <- j
	case PT_PreSleep:
		sessionId := e.fromSessionId
		w, ok := s.worker[sessionId]
		if !ok {
			log.Warningf("unregister worker, sessionId %d", sessionId)
			w = args.t0.(*Worker)
			s.worker[w.SessionId] = w
			break
		}

		w.status = wsSleep
		log.Debugf("worker with sessionId %d sleep", sessionId)
		//check if there are any jobs for this worker
		for k := range w.canDo {
			if s.wakeupWorker(k) {
				break
			}
		}
	case PT_SubmitJobLow, PT_SubmitJob, PT_SubmitJobHigh, PT_SubmitJobLowBG, PT_SubmitJobBG, PT_SubmitJobHighBG:
		s.handleSubmitJob(e)
	case PT_SubmitJobSched:
		s.handleCronJob(e)
	case PT_SubmitJobEpoch:
		s.handleSubmitEpochJob(e)
	case PT_GetStatus:
		jobhandle := bytes2str(args.t0)
		if job, ok := s.jobs[jobhandle]; ok {
			e.result <- &Tuple{t0: args.t0, t1: true, t2: job.Running,
				t3: job.Percent, t4: job.Denominator}
			break
		}

		e.result <- &Tuple{t0: args.t0, t1: false, t2: false,
			t3: 0, t4: 100} //always set Denominator to 100 if no status update
	case PT_WorkData, PT_WorkWarning, PT_WorkStatus, PT_WorkComplete,
		PT_WorkFail, PT_WorkException:
		s.handleWorkReport(e)
	default:
		log.Warningf("%s, %d", e.tp, e.tp)
	}
}

func (s *Server) wakeupTravel() {
	for k, jw := range s.funcWorker {
		if jw.jobs.Len() > 0 {
			s.wakeupWorker(k)
		}
	}
}

func (s *Server) pubCounter() {
	for k, v := range s.opCounter {
		stats.PubInt64(k.String(), v)
	}
}

func (s *Server) EvtLoop() {
	tick := time.NewTicker(1 * time.Second)
	for {
		select {
		case e := <-s.protoEvtCh:
			s.handleProtoEvt(e)
		case e := <-s.ctrlEvtCh:
			s.handleCtrlEvt(e)
		case <-tick.C:
			s.pubCounter()
			stats.PubInt("len(protoEvtCh)", len(s.protoEvtCh))
			stats.PubInt("worker count", len(s.worker))
			stats.PubInt("job queue length", len(s.jobs))
			stats.PubInt("queue count", len(s.funcWorker))
			stats.PubInt("client count", len(s.client))
			stats.PubInt64("forwardReport", s.forwardReport)
		}
	}
}

func (s *Server) WatcherLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	for {
		select {
		case <-ticker.C:
			cjs, err := s.store.GetCronJobs()
			if err != nil {
				log.Error(err)
				continue
			}
			rep := 0
			one := 0
			for _, cj := range cjs {
				if _, isOne := s.ExpressionToEpoch(cj.Expression); isOne {
					one++
				} else {
					rep++
				}
			}
			log.Infof("total cron job: %v #repeated job: %v #onetime job: %v", len(cjs), rep, one)
			js, err := s.store.GetJobs()
			if err != nil {
				log.Error(err)
				continue
			}
			var b, r int = 0, 0
			for _, j := range js {
				if j.IsBackGround {
					b++
				}
				if j.Running {
					r++
				}
			}
			log.Infof("total job: %v #background: %v #running: %v", len(js), b, r)
		}
	}

}

func (s *Server) allocSessionId() int64 {
	return atomic.AddInt64(&s.startSessionId, 1)
}

func (s *Server) DeleteCronJob(cj *CronJob) error {
	sj, err := s.store.DeleteCronJob(cj)
	if err == lberror.ErrNotFound {
		log.Errorf("handle `%v` not found", cj.Handle)
		return fmt.Errorf("handle `%v` not found", cj.Handle)
	}
	if err != nil {
		log.Errorln(err)
		return err
	}
	s.cronSvc.Remove(cron.EntryID(sj.CronEntryID))
	log.Debugf("job `%v` successfully cancelled.", cj.Handle)
	return nil
}

func (s *Server) ExpressionToEpoch(scdTime string) (int64, bool) {
	if strings.HasPrefix(scdTime, EpochTimePrefix) {
		value, err := strconv.ParseInt(scdTime[len(EpochTimePrefix):], 10, 64)
		if err != nil {
			log.Errorln(err)
		}
		return value, true
	}
	return 0, false
}
