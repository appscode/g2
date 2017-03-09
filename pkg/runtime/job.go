package runtime

import (
	"time"
)

const (
	PRIORITY_LOW  = 0
	PRIORITY_HIGH = 1

	JobPrefix       = "H:"
	CronJobPrefix   = "S:"
	EpochTimePrefix = "UTC-"
)

type Job struct {
	Handle       string    `json:"job_handle,omitempty"` //server job handle
	Id           string    `json:"id,omitempty"`
	Data         []byte    `json:"data,omitempty"`
	Running      bool      `json:"is_running,omitempty"`
	Percent      int       `json:"percent,omitempty"`
	Denominator  int       `json:"denominator,omitempty"`
	CreateAt     time.Time `json:"created_at,omitempty"`
	ProcessAt    time.Time `json:"process_at,omitempty"`
	TimeoutSec   int32     `json:"timeout_sec,omitempty"`
	CreateBy     int64     `json:"created_by,omitempty"` //client sessionId
	ProcessBy    int64     `json:"process_by,omitempty"` //worker sessionId
	FuncName     string    `json:"function_name,omitempty"`
	IsBackGround bool      `json:"is_background_job"`
	Priority     int       `json:"priority"`
	CronHandle   string    `json:"cronjob_handle,omitempty"`
}

type CronJob struct {
	JobTemplete   Job       `json:"job_templete,omitempty"`
	Handle        string    `json:"cronjob_handle,omitempty"`
	CronEntryID   int       `json:"cron_entry_id"`
	Expression    string    `json:"expression,omitempty"`
	Next          time.Time `json:"next,omitempty"`
	Prev          time.Time `json:"prev,omitempty"`
	Created       int       `json:"created,omitempty"`
	SuccessfulRun int       `json:"successful_run,omitempty"`
	FailedRun     int       `json:"failed_run,omitempty"`
}

func (c *Job) Key() string {
	return c.Handle
}

func (c *Job) Prefix() string {
	return JobPrefix
}

func (c *CronJob) Key() string {
	return c.Handle
}

func (c *CronJob) Prefix() string {
	return CronJobPrefix
}
