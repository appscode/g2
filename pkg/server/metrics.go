package server

func (s *Server) Stats() map[string]int {
	ret := map[string]int{
		"proto_evt_ch":   len(s.protoEvtCh),
		"forward_report": int(s.forwardReport),
		"queue_count":    len(s.funcWorker),
		"job_queue":      len(s.jobs),
	}
	for k, v := range s.opCounter {
		ret[k.String()] = int(v)
	}
	return ret
}

func (s *Server) Workers() int {
	return len(s.worker)
}

func (s *Server) Jobs() int {
	return len(s.jobs) + len(s.cronJobs)
}

func (s *Server) Clients() int {
	return len(s.client)
}

func (s *Server) RunningJobsByWorker() map[string]int {
	ret := make(map[string]int)
	for _, worker := range s.worker {
		ret[worker.workerId] += len(worker.runningJobs)
	}
	return ret
}

func (s *Server) RunningJobsByFunction() map[string]int {
	ret := make(map[string]int)
	for _, worker := range s.worker {
		for _, job := range worker.runningJobs {
			ret[job.FuncName]++
		}
	}
	return ret
}
