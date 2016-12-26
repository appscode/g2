package storage

import (
	"database/sql"
	"flag"
	"log"
	"sync"
	"testing"
	"time"

	. "github.com/appscode/g2/pkg/runtime"
	. "github.com/appscode/g2/pkg/storage/leveldb"
)

const (
	mysqlSource      = "root:@tcp(localhost:3306)/gogearmand?parseTime=true"
	mysqlCreateTable = `CREATE TABLE job(Handle varchar(128),Id varchar(128),Priority INT, CreateAt TIMESTAMP,
		FuncName varchar(128),Data varchar(16384)) ENGINE=InnoDB;`
	mysqlDropTable = `DROP TABLE job`
	TimeFormatStr  = "2006-01-02 15:04:05"
)

var (
	redis = flag.String("redis", "localhost:6379", "redis address")
)

var (
	redisQ *LevelDbQ
)

var testJobs = []*Job{
	{Handle: JobPrefix + "handle0_", Id: "id0_",
		Data: []byte("data0_"), CreateAt: time.Now().UTC(), FuncName: "funcName0_", Priority: 0},
	{Handle: JobPrefix + "", Id: "id1_",
		Data: []byte("data1_"), CreateAt: time.Now().UTC(), FuncName: "funcName1_", Priority: 1},
	{Handle: JobPrefix + "handle2_", Id: "id2_",
		Data: []byte("data2_"), FuncName: "funcName2_", Priority: 2},
	{Handle: JobPrefix + "handle3_", Id: "id3_",
		CreateAt: time.Now().UTC(), FuncName: "funcName3_", Priority: 3},
	{Handle: JobPrefix + "handle4_", Id: "id4_",
		Data: []byte(""), CreateAt: time.Now().UTC(), FuncName: "funcName4_", Priority: 4},
	{Handle: JobPrefix + "handle5_", Id: "id5_",
		Data: []byte("don't store"), FuncName: "funcName5_", Priority: 5},
}

func init() {
	flag.Parse()
	redisQ = &LevelDbQ{}

	//remove table if it was created before
	operateTable(mysqlDropTable)
	if err := operateTable(mysqlCreateTable); err != nil {
		log.Fatal(err)
	}
}

func TestInit(t *testing.T) {
	testInit(t, redisQ)
}

func TestAddAndGetJob(t *testing.T) {
	testGetJob(t, redisQ, nil)

	var jobs []*Job
	for i := 0; i < 2; i++ {
		for _, j := range testJobs {
			if i == 5 {
				continue
			}
			testAddjob(t, redisQ, j)
			jobs = append(jobs, j)
		}
	}
	testGetJob(t, redisQ, jobs[0:len(testJobs)])
}

func TestDoneJob(t *testing.T) {
	for i := 0; i < 2; i++ {
		for _, job := range testJobs {
			testDoneJob(t, redisQ, job)
		}
	}
}

func BenchmarkBasicOpts(b *testing.B) {
	b.StopTimer()
	n := b.N
	wg := sync.WaitGroup{}
	wg.Add(n)

	b.Log("benchmark, n:", n)
	b.StartTimer()
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			jobs := testJobs
			t := &testing.T{}

			for _, j := range jobs {
				testAddjob(t, redisQ, j)
			}
			testGetJob(t, redisQ, jobs)
		}()
	}
	wg.Wait()

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			jobs := testJobs
			t := &testing.T{}

			for _, j := range jobs {
				testDoneJob(t, redisQ, j)
			}
		}()
	}
	wg.Wait()
}

func testInit(t *testing.T, store JobQueue) {
	if err := store.Init(); err != nil {
		t.Errorf("failed to store init, err:%s", err.Error())
	}
}

func testAddjob(t *testing.T, store JobQueue, j *Job) {
	if err := store.AddJob(j); err != nil {
		t.Errorf("failed to addjob, err:%s", err.Error())
	}
}

func testGetJob(t *testing.T, store JobQueue, retJobs []*Job) {
	jobs, err := store.GetJobs()
	if err != nil {
		t.Errorf("failed to get jobs, err:%s", err.Error())
		return
	}

	if len(retJobs) != len(jobs) {
		t.Errorf("jobs length not match, len1:%+v, len2:%d, jobs1:%+v, jobs2:%+v",
			len(retJobs), len(jobs), retJobs, jobs)
		return
	}
}

func testDoneJob(t *testing.T, store JobQueue, j *Job) {
	err := store.DoneJob(j)
	if err != nil {
		t.Errorf("failed to done job, err:%s", err.Error())
	}
}

func operateTable(str string) (err error) {
	db, err := sql.Open("mysql", mysqlSource)
	if err != nil {
		return
	}
	defer db.Close()

	_, err = db.Exec(str)

	return
}
