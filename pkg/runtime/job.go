package runtime

import (
	"time"
)

const (
	PRIORITY_LOW  = 0
	PRIORITY_HIGH = 1

	JobPrefix       = "H:"
	SchedJobPrefix  = "S:"
	EpochTimePrefix = "UTC-"
)

type Job struct {
	Handle       string    `json:"job_handle"` //server job handle
	Id           string    `json:"id,omitempty"`
	Data         []byte    `json:"data,omitempty"`
	Running      bool      `json:"is_running"`
	Percent      int       `json:"percent,omitempty"`
	Denominator  int       `json:"denominator,omitempty"`
	CreateAt     time.Time `json:"created_at,omitempty"`
	ProcessAt    time.Time `json:"process_at,omitempty"`
	TimeoutSec   int       `json:"timeout_sec,omitempty"`
	CreateBy     int64     `json:"created_by,omitempty"` //client sessionId
	ProcessBy    int64     `json:"process_by,omitempty"` //worker sessionId
	FuncName     string    `json:"function_name"`
	IsBackGround bool      `json:"is_background_job"`
	Priority     int       `json:"priority"`
	CronHandle   string    `json:"cronjob_handle"`
}

type CronJob struct {
	JobTemplete   Job    `json:"job_templete"`
	Handle        string `json:"cronjob_handle"`
	CronEntryID   int    `json:"cron_entry_id"`
	ScheduleTime  string `json:"schedule_time"`
	SuccessfulRun int    `json:"successful_run"`
	FailedRun     int    `json:"failed_run"`
}
