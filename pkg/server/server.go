package server

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/appscode/g2/pkg/metrics"
	. "github.com/appscode/g2/pkg/runtime"
	"github.com/appscode/g2/pkg/storage"
	"github.com/appscode/g2/pkg/storage/leveldb"
	"github.com/appscode/go/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	lberror "github.com/syndtr/goleveldb/leveldb/errors"
	"gopkg.in/robfig/cron.v2"
)

type Config struct {
	ListenAddr string
	Storage    string
	WebAddress string
}

type Server struct {
	config         Config
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
	cronJobs       map[string]*CronJob
	mu             *sync.RWMutex
}

var ( //const replys, to avoid building it every time
	wakeupReply = constructReply(PT_Noop, nil)
	nojobReply  = constructReply(PT_NoJob, nil)
)

func NewServer(cfg Config) *Server {
	srv := &Server{
		config:     cfg,
		funcWorker: make(map[string]*jobworkermap),
		protoEvtCh: make(chan *event, 100),
		ctrlEvtCh:  make(chan *event, 100),
		worker:     make(map[int64]*Worker),
		client:     make(map[int64]*Client),
		jobs:       make(map[string]*Job),
		opCounter:  make(map[PT]int64),
		cronSvc:    cron.New(),
		cronJobs:   make(map[string]*CronJob),
		mu:         &sync.RWMutex{},
	}

	// Initiate data storage
	if len(cfg.Storage) > 0 {
		s, err := leveldbq.New(cfg.Storage)
		if err != nil {
			log.Info(err)
		}
		srv.store = s
	}
	return srv
}

func (s *Server) loadAllJobs() {
	if s.store == nil {
		return
	}
	jobs, err := s.store.GetAll(&Job{})
	if err != nil {
		log.Error(err)
		return
	}
	for _, jb := range jobs {
		j, ok := jb.(*Job)
		if !ok {
			log.Errorln("invalid job")
			continue
		}
		j.ProcessBy = 0 //no body handle it now
		j.CreateBy = 0  //clear
		log.Debugf("handle: %v\tfunc: %v\tis_background: %v", j.Handle, j.FuncName, j.IsBackGround)
		s.doAddJob(j)
	}

}

func (s *Server) loadAllCronJobs() {
	if s.store == nil {
		return
	}
	schedJobs, err := s.store.GetAll(&CronJob{})
	if err != nil {
		log.Error(err)
		return
	}
	for _, sji := range schedJobs {
		sj, ok := sji.(*CronJob)
		if !ok {
			log.Errorln("invalid cronjob")
			continue
		}
		if epoch, ok := s.ExpressionToEpoch(sj.Expression); ok {
			log.Debugf("handle: %v func: %v schedule: %v", sj.Handle, sj.JobTemplete.FuncName, time.Unix(epoch, 0).Local())
			s.doAddEpochJob(sj)
		} else {
			log.Debugf("handle: %v func: %v expr: %v", sj.Handle, sj.JobTemplete.FuncName, sj.Expression)
			s.doAddCronJob(sj)
		}
	}
}

func (s *Server) Start() {
	ln, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		log.Fatal(err)
	}

	log.Debug("listening on", s.config.ListenAddr)
	go s.EvtLoop()

	// Run REST API Server
	if len(s.config.WebAddress) > 0 {
		go func() {
			prometheus.MustRegister(metrics.NewServerCollector(s))
			http.Handle("/metrics", promhttp.Handler())

			registerAPIHandlers(s)

			log.Infoln("Running web api at", s.config.WebAddress)
			log.Fatalln(http.ListenAndServe(s.config.WebAddress, nil))
		}()
	}

	go s.WatcherLoop()
	go s.WatchJobTimeout()
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
	s.handleCanDoTimeout(funcName, w, DefaultTimeout)
}

func (s *Server) handleCanDoTimeout(funcName string, w *Worker, timeout int32) {
	w.canDo[funcName] = timeout
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
	if cron, ok := s.getCronJobFromMap(j.CronHandle); ok {
		cron.Created++
		s.addCronJob(cron)
	}
	s.saveJobInDB(j)
}

func (s *Server) doAddCronJob(sj *CronJob) {
	if _, ok := s.getCronJobFromMap(sj.Handle); ok {
		log.Infoln("cronjob already exists with handle ", sj.Handle)
		return
	}
	scdT, err := NewCronSchedule(sj.Expression)
	if err != nil {
		log.Errorln(err)
		return
	}
	sj.CronEntryID = int(s.cronSvc.Schedule(
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
				sj.Next = scdT.Schedule().Next(time.Now())
				sj.Prev = time.Now()
				//Update cronJob with new Next and Prev time
				s.addCronJob(sj)
				s.doAddJob(jb)
			})))
	sj.Next = scdT.Schedule().Next(time.Now())
	s.addCronJob(sj)
}

func (s *Server) doAddEpochJob(cj *CronJob) {
	if _, ok := s.getCronJobFromMap(cj.Handle); ok {
		log.Infoln("epochjob already exists with handle ", cj.Handle)
		return
	}
	epoch, ok := s.ExpressionToEpoch(cj.Expression)
	if !ok {
		log.Errorln("invalid epoch job expression, ", cj.Expression)
		return
	}
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
		s.doAddJob(j)
		err := s.removeCronJob(cj)
		if err != nil {
			log.Errorln(err)
		}
	})
	cj.Next = time.Unix(epoch, 0)
	s.addCronJob(cj)

}

func (s *Server) popJob(sessionId int64) (j *Job) {
	for funcName := range s.worker[sessionId].canDo {
		if wj, ok := s.funcWorker[funcName]; ok {
			for it := wj.jobs.Front(); it != nil; it = it.Next() {
				jtmp := it.Value.(*Job)
				//Don't return running job. This case arise when server restarted but some job still executing
				if jtmp.Running {
					continue
				}
				j = jtmp
				wj.jobs.Remove(it)
				return
			}
		}
	}
	return
}

func (s *Server) wakeupWorker(funcName string) bool {
	wj, ok := s.funcWorker[funcName]
	if !ok || wj.jobs.Len() == 0 || wj.workers.Len() == 0 {
		return false
	}
	//Don't wakeup for running job
	var allRunning = true
	for it := wj.jobs.Front(); it != nil; it = it.Next() {
		j := it.Value.(*Job)
		if !j.Running {
			allRunning = false
			break
		}
	}
	if allRunning {
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

func (s *Server) removeJob(j *Job, isSuccess bool) {
	delete(s.jobs, j.Handle)
	if pw, found := s.worker[j.ProcessBy]; found {
		delete(pw.runningJobs, j.Handle)
	}
	log.Debugf("job removed: %v", j.Handle)
	if j.IsBackGround {
		if cron, ok := s.getCronJobFromMap(j.CronHandle); ok {
			if isSuccess {
				cron.SuccessfulRun++
			} else {
				cron.FailedRun++
			}
			s.addCronJob(cron)
		}
	}
	if s.store == nil {
		return
	}
	if err := s.store.Delete(j); err != nil {
		log.Warning(err)
	}
}

func (s *Server) jobDone(j *Job) {
	s.removeJob(j, true)
}

func (s *Server) jobFailed(j *Job) {
	s.removeJob(j, false)
}

func (s *Server) jobFailedWithException(j *Job, cause string) {
	log.Warningf("Job failed with cause `%v`", cause)
	s.removeJob(j, false)
}

func (s *Server) handleCloseSession(e *event) error {
	sessionId := e.fromSessionId
	switch {
	case s.isClient(e.fromSessionId):
		s.handleCloseSessionForClient(sessionId)
	case s.isWorker(e.fromSessionId):
		s.handleCloseSessionForWorker(sessionId)
	default:
		log.Errorln("invalid sessionId")
	}
	e.result <- true //notify close finish
	return nil
}

func (s *Server) handleCloseSessionForWorker(sessionId int64) {
	if w, ok := s.worker[sessionId]; ok {
		if sessionId != w.SessionId {
			log.Fatalf("sessionId not match %d-%d, bug found", sessionId, w.SessionId)
		}
		s.removeWorkerBySessionId(w.SessionId)
		log.Debugf("worker with sessionId: %v unregistered.", sessionId)
	}
}

func (s *Server) handleCloseSessionForClient(sessionId int64) {
	if c, ok := s.client[sessionId]; ok {
		log.Debug("removeClient with sessionId ", sessionId)
		delete(s.client, c.SessionId)
	}
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
		var cjs []*CronJob = make([]*CronJob, 0)

		s.mu.RLock()
		for _, v := range s.cronJobs {
			cjs = append(cjs, v)
		}
		s.mu.RUnlock()

		buf, err = json.Marshal(cjs)
		if err != nil {
			log.Error(err)
			return err
		}
		return nil
	}

	if v, ok := s.getCronJobFromMap(e.handle); ok {
		data, err := json.Marshal(v)
		if err != nil {
			log.Error(err)
			return err
		}
		buf = []byte(data)
	}
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
	s.doAddJob(j)
}

func (s *Server) handleSubmitCronJob(e *event) {
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
	s.doAddCronJob(sj)
	log.Debugf("add cron job with handle: %v func: %v expr: %v", sj.Handle, sj.JobTemplete.FuncName, sj.Expression)
}

func (s *Server) handleSubmitEpochJob(e *event) {
	args := e.args
	c := args.t0.(*Client)
	s.client[c.SessionId] = c
	funcName := bytes2str(args.t1)
	epochStr := bytes2str(args.t3)
	_, err := strconv.ParseInt(epochStr, 10, 64)
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

	// persistent Cron Job
	s.doAddEpochJob(sj)
	log.Debugf("add epoch job with handle: %v func: %v", sj.Handle, sj.JobTemplete.FuncName)
}

func (s *Server) handleWorkReport(e *event) {
	args := e.args
	slice := args.t0.([][]byte)
	jobhandle := bytes2str(slice[0])

	j, ok := s.getRunningJobByHandle(jobhandle)
	if !ok {
		log.Warningf("job information lost, %v job handle %v, %+v",
			e.tp, jobhandle, s.jobs)
		return
	}

	switch e.tp {
	case PT_WorkStatus:
		j.Percent, _ = strconv.Atoi(string(slice[1]))
		j.Denominator, _ = strconv.Atoi(string(slice[2]))
	case PT_WorkException:
		s.jobFailedWithException(j, string(slice[1]))
	case PT_WorkFail:
		s.jobFailed(j)
	case PT_WorkComplete:
		s.jobDone(j)
	}

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
		log.Debugf("worker with sessionId: %v add function `%v`", w.SessionId, funcName)
	case PT_CanDoTimeout: //todo: fix timeout support, now just as CAN_DO
		w := args.t0.(*Worker)
		funcName := args.t1.(string)
		var timeout int32
		err := binary.Read(bytes.NewReader(args.t2.([]byte)), binary.BigEndian, &timeout)
		if err != nil {
			log.Fatal("error when parsing timeout")
		}
		s.handleCanDoTimeout(funcName, w, timeout)
		log.Debugf("worker with sessionId: %v add function `%v` with timeout %v sec", w.SessionId, funcName, timeout)
	case PT_CantDo:
		sessionId := e.fromSessionId
		funcName := args.t0.(string)
		if jw, ok := s.funcWorker[funcName]; ok {
			s.removeWorker(jw.workers, sessionId)
		}
		delete(s.worker[sessionId].canDo, funcName)
		log.Debugf("worker with sessionId: %v remove function `%v`", sessionId, funcName)
	case PT_SetClientId:
		w := args.t0.(*Worker)
		w.workerId = args.t1.(string)
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
			j.TimeoutSec = w.canDo[j.FuncName]
			//track this job
			j.Running = true
			w.runningJobs[j.Handle] = j
			s.saveJobInDB(j)

		} else { //no job
			w.status = wsPrepareForSleep
		}
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
		s.handleSubmitCronJob(e)
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

func (s *Server) EvtLoop() {
	for {
		select {
		case e := <-s.protoEvtCh:
			s.handleProtoEvt(e)
		case e := <-s.ctrlEvtCh:
			s.handleCtrlEvt(e)
		}
	}
}

func (s *Server) WatcherLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	for {
		select {
		case <-ticker.C:
			rep := 0
			one := 0

			s.mu.RLock()
			cLen := len(s.cronJobs)
			for _, cj := range s.cronJobs {
				if _, isOne := s.ExpressionToEpoch(cj.Expression); isOne {
					one++
				} else {
					rep++
				}
			}
			s.mu.RUnlock()

			log.Infof("total cron job: %v #repeated job: %v #onetime job: %v", cLen, rep, one)
			var b, r int = 0, 0
			for _, j := range s.jobs {
				if j.IsBackGround {
					b++
				}
				if j.Running {
					r++
				}
			}
			log.Infof("total job: %v #background: %v #running: %v", len(s.jobs), b, r)
		}
	}
}

func (s *Server) WatchJobTimeout() {
	for range time.NewTicker(time.Second).C {
		for _, job := range s.jobs {
			if !job.Running {
				continue
			}
			if time.Now().Sub(job.ProcessAt) > time.Duration(job.TimeoutSec)*time.Second {
				log.Infof("job %v failed, cause timeout expired", job.Handle)
				s.jobFailed(job)

				if job.IsBackGround {
					continue
				}
				c, ok := s.client[job.CreateBy]
				if !ok {
					log.Debug(job.Handle, "sessionId", job.CreateBy, "missing")
					continue
				}
				sendTimeoutException(c.in, job.Handle, "timeout expired")
				s.forwardReport++
			}

		}
	}
}

func (s *Server) allocSessionId() int64 {
	return atomic.AddInt64(&s.startSessionId, 1)
}

func (s *Server) DeleteCronJob(cj *CronJob) error {
	err := s.removeCronJob(cj)
	if err != nil {
		log.Errorln(err)
		return err
	}
	s.cronSvc.Remove(cron.EntryID(cj.CronEntryID))
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

func (s *Server) isWorker(sessionId int64) bool {
	for k := range s.worker {
		if k == sessionId {
			return true
		}
	}
	return false
}

func (s *Server) getRunningJobByHandle(handle string) (*Job, bool) {
	for _, j := range s.jobs {
		if j.Handle == handle && j.Running {
			return j, true
		}
	}
	return nil, false
}

func (s *Server) isClient(sessionId int64) bool {
	for k := range s.client {
		if k == sessionId {
			return true
		}
	}
	return false
}

func (s *Server) addCronJob(cj *CronJob) {
	s.mu.Lock()
	s.cronJobs[cj.Handle] = cj
	s.mu.Unlock()

	if s.store != nil {
		err := s.store.Add(cj)
		if err != nil {
			log.Error(err)
		}
	}
}

func (s *Server) saveJobInDB(j *Job) {
	if s.store == nil {
		return
	}
	if err := s.store.Add(j); err != nil {
		log.Warning(err)
	}
}

func (s *Server) removeCronJob(cj *CronJob) error {
	s.mu.Lock()
	delete(s.cronJobs, cj.Handle)
	s.mu.Unlock()

	if s.store != nil {
		err := s.store.Delete(cj)
		if err == lberror.ErrNotFound {
			log.Errorf("handle `%v` not found", cj.Handle)
			return fmt.Errorf("handle `%v` not found", cj.Handle)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) getCronJobFromMap(handle string) (cronJob *CronJob, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cronJob, ok = s.cronJobs[handle]
	return
}
