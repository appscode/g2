//using key as queue

package leveldbq

import (
	"encoding/json"
	"strings"
	"sync"

	. "github.com/appscode/g2/pkg/runtime"
	"github.com/appscode/g2/pkg/storage"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type LevelDbQ struct {
	l  sync.RWMutex
	db *leveldb.DB
}

var _ storage.Db = &LevelDbQ{}

func New(dir string) (storage.Db, error) {
	db, err := leveldb.OpenFile(strings.TrimRight(dir, "/")+"/gearmand.ldb", nil)
	if err != nil {
		return nil, err
	}
	return &LevelDbQ{db: db, l: sync.RWMutex{}}, nil
}

func (q *LevelDbQ) AddJob(j *Job) error {
	buf, err := json.Marshal(j)
	if err != nil {
		return err
	}

	q.l.Lock()
	defer q.l.Unlock()
	return q.db.Put([]byte(j.Handle), buf, nil)
}

func (q *LevelDbQ) DeleteJob(j *Job, isSuccess bool) error {
	if j.CronHandle != "" {
		cj, err := q.GetCronJob(j.CronHandle)
		if err != nil {
			return err
		}
		if isSuccess {
			cj.SuccessfulRun++
		} else {
			cj.FailedRun++
		}
		q.AddCronJob(cj)
	}
	q.l.Lock()
	defer q.l.Unlock()
	return q.db.Delete([]byte(j.Handle), nil)
}

func (q *LevelDbQ) GetJob(handle string) (*Job, error) {
	q.l.RLock()
	data, err := q.db.Get([]byte(handle), nil)
	q.l.RUnlock()

	if err != nil {
		return nil, err
	}
	j := &Job{}
	err = json.Unmarshal(data, j)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (q *LevelDbQ) GetJobs() ([]*Job, error) {
	q.l.RLock()
	defer q.l.RUnlock()
	jobs := make([]*Job, 0)
	iter := q.db.NewIterator(util.BytesPrefix([]byte(JobPrefix)), nil)
	for iter.Next() {
		// key := iter.Key()
		// value := iter.Value()
		var j Job
		err := json.Unmarshal(iter.Value(), &j)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, &j)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func (q *LevelDbQ) AddCronJob(sj *CronJob) error {
	buf, err := json.Marshal(sj)
	if err != nil {
		return err
	}
	q.l.Lock()
	defer q.l.Unlock()
	return q.db.Put([]byte(sj.Handle), buf, nil)
}

func (q *LevelDbQ) GetCronJob(handle string) (*CronJob, error) {
	q.l.RLock()
	data, err := q.db.Get([]byte(handle), nil)
	q.l.RUnlock()

	if err != nil {
		return nil, err
	}
	cj := &CronJob{}
	err = json.Unmarshal(data, cj)
	if err != nil {
		return nil, err
	}
	return cj, nil
}

func (q *LevelDbQ) DeleteCronJob(sj *CronJob) (*CronJob, error) {
	cj, err := q.GetCronJob(sj.Handle)
	if err != nil {
		return nil, err
	}
	q.l.Lock()
	defer q.l.Unlock()
	return cj, q.db.Delete([]byte(cj.Handle), nil)
}

func (q *LevelDbQ) GetCronJobs() ([]*CronJob, error) {
	cronJobs := make([]*CronJob, 0)
	q.l.RLock()
	defer q.l.RUnlock()
	iter := q.db.NewIterator(util.BytesPrefix([]byte(SchedJobPrefix)), nil)
	for iter.Next() {
		// key := iter.Key()
		// value := iter.Value()
		var j CronJob
		err := json.Unmarshal(iter.Value(), &j)
		if err != nil {
			return nil, err
		}
		cronJobs = append(cronJobs, &j)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil, err
	}
	return cronJobs, nil
}
