package storage

import (
	. "github.com/appscode/g2/pkg/runtime"
)

type Db interface {
	JobQueue
	SchedJobQueue
}

type JobQueue interface {
	AddJob(j *Job) error
	DeleteJob(j *Job, isSuccess bool) error
	GetJob(handle string) (*Job, error)
	GetJobs() ([]*Job, error)
}

type SchedJobQueue interface {
	AddCronJob(sj *CronJob) error
	UpdateCronJob(string, map[string]interface{}) error
	DeleteCronJob(sj *CronJob) (*CronJob, error)
	GetCronJob(handle string) (*CronJob, error)
	GetCronJobs() ([]*CronJob, error)
}
