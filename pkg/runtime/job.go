package runtime

import (
	"bytes"
	"encoding/json"
	"time"
)

const (
	PRIORITY_LOW  = 0
	PRIORITY_HIGH = 1

	JobPrefix = "H:"
)

type Job struct {
	Handle       string //server job handle
	Id           string
	Data         []byte
	Running      bool
	Percent      int
	Denominator  int
	CreateAt     time.Time
	ProcessAt    time.Time
	TimeoutSec   int
	CreateBy     int64 //client sessionId
	ProcessBy    int64 //worker sessionId
	FuncName     string
	IsBackGround bool
	Priority     int
}

func (self *Job) String() string {
	b := &bytes.Buffer{}
	enc := json.NewEncoder(b)
	m := make(map[string]interface{})
	m["Handle"] = self.Handle
	m["Id"] = self.Id
	m["Data"] = string(self.Data)
	m["Running"] = self.Running
	m["Percent"] = self.Percent
	m["Denominator"] = self.Denominator
	m["CreateAt"] = self.CreateAt
	m["ProcessAt"] = self.ProcessAt
	m["Running"] = self.Running
	m["TimeoutSec"] = self.TimeoutSec
	m["CreateBy"] = self.CreateBy
	m["ProcessBy"] = self.ProcessBy
	m["FuncName"] = self.FuncName
	m["IsBackGround"] = self.IsBackGround
	m["Priority"] = self.Priority

	if err := enc.Encode(m); err != nil {
		return ""
	}

	return string(b.Bytes())
}
