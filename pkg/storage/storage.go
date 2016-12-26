package storage

import (
	. "github.com/appscode/g2/pkg/runtime"
)

type JobQueue interface {
	AddJob(j *Job) error
	DoneJob(j *Job) error
	GetJobs() ([]*Job, error)
}
