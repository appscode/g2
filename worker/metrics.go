package worker

func (w *Worker) Running() (string, int) {
	if w.running {
		return w.Id, 1
	}
	return w.Id, 0
}

func (w *Worker) Agents() int {
	return len(w.agents)
}
