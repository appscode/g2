//using key as queue

package leveldbq

import (
	"encoding/json"
	"strings"

	. "github.com/appscode/g2/pkg/runtime"
	"github.com/appscode/g2/pkg/storage"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type LevelDbQ struct {
	db *leveldb.DB
}

var _ storage.JobQueue = &LevelDbQ{}

func New(dir string) (storage.JobQueue, error) {
	db, err := leveldb.OpenFile(strings.TrimRight(dir, "/")+"/gearmand.ldb", nil)
	if err != nil {
		return nil, err
	}
	return &LevelDbQ{db: db}, nil
}

func (q *LevelDbQ) AddJob(j *Job) error {
	buf, err := json.Marshal(j)
	if err != nil {
		return err
	}
	return q.db.Put([]byte(j.Handle), buf, nil)
}

func (q *LevelDbQ) DoneJob(j *Job) error {
	return q.db.Delete([]byte(j.Handle), nil)
}

func (q *LevelDbQ) GetJobs() ([]*Job, error) {
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
