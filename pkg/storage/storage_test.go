package storage_test

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	. "github.com/appscode/g2/pkg/runtime"
	. "github.com/appscode/g2/pkg/storage"
	. "github.com/appscode/g2/pkg/storage/leveldb"
	"github.com/stretchr/testify/assert"
)

var (
	db ItemQueue
)

var testJobs = []*Job{
	{Handle: JobPrefix + "handle0_", Id: "id0_",
		Data: []byte("data0_"), CreateAt: time.Now(), ProcessAt: time.Now(), FuncName: "funcName0_", Priority: 0},
	{Handle: JobPrefix + "", Id: "id1_",
		Data: []byte("data1_"), CreateAt: time.Now().UTC(), ProcessAt: time.Now(), FuncName: "funcName1_", Priority: 1},
	{Handle: JobPrefix + "handle2_", Id: "id2_",
		Data: []byte("data2_"), FuncName: "funcName2_", CreateAt: time.Now().UTC(), ProcessAt: time.Now().UTC(), Priority: 2},
	{Handle: JobPrefix + "handle3_", Id: "id3_",
		CreateAt: time.Now().UTC(), ProcessAt: time.Now().UTC(), FuncName: "funcName3_", Priority: 3},
	{Handle: JobPrefix + "handle4_", Id: "id4_",
		CreateAt: time.Now().UTC(), ProcessAt: time.Now().UTC(), FuncName: "funcName4_", Priority: 4},
	{Handle: JobPrefix + "handle5_", Id: "id5_",
		Data: []byte("don't store"), FuncName: "funcName5_", CreateAt: time.Now().UTC(), ProcessAt: time.Now().UTC(), Priority: 5},
}

var testCronJobs = []*CronJob{
	{JobTemplete: *testJobs[0], Handle: CronJobPrefix + "handle0_", CronEntryID: 1, Expression: "* * 1 * *",
		Next: time.Now().Add(10), Prev: time.Now().Add(-10), Created: 1},
	{JobTemplete: *testJobs[0], Handle: CronJobPrefix + "handle1_", CronEntryID: 2, Expression: "* * 2 * *",
		Next: time.Now().Add(10), Prev: time.Now().Add(-10), Created: 2},
	{JobTemplete: *testJobs[0], Handle: CronJobPrefix + "handle2_", CronEntryID: 3, Expression: "* * 3 * *",
		Next: time.Now().Add(10), Prev: time.Now().Add(-10), Created: 3},
	{JobTemplete: *testJobs[0], Handle: CronJobPrefix + "handle3_", CronEntryID: 4, Expression: "* * 4 * *",
		Next: time.Now().Add(10), Prev: time.Now().Add(-10), Created: 4},
	{JobTemplete: *testJobs[0], Handle: CronJobPrefix + "handle4_", CronEntryID: 5, Expression: "* * 5 * *",
		Next: time.Now().Add(10), Prev: time.Now().Add(-10), Created: 5},
}

func init() {
	flag.Parse()

	dir, err := ioutil.TempDir(os.TempDir(), "g2")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Using temp dir %v", dir)
	db, err = New(dir)
	if err != nil {
		log.Fatal(err)
	}
}

func TestAll(t *testing.T) {
	//Add test
	for _, j := range testJobs {
		testAdd(t, db, j)
	}
	for _, cj := range testCronJobs {
		testAdd(t, db, cj)
	}
	//get test
	for _, j := range testJobs {
		jn := &Job{Handle: j.Handle}
		testGet(t, db, jn)
		assert.Equal(t, j, jn)
	}
	for _, cj := range testCronJobs {
		cjn := &CronJob{Handle: cj.Handle}
		testGet(t, db, cjn)
		assert.Equal(t, cj, cjn)
	}
	//get all test
	allJobs := testGetAll(t, db, &Job{})
	for _, j := range testJobs {
		var found bool = false
		for _, aj := range allJobs {
			if aj.(*Job).Handle == j.Handle {
				found = true
				assert.Equal(t, j, aj)
			}
		}
		if !found {
			t.Fatalf("job not found with handle %v", j.Handle)
		}
	}

	allCronJobs := testGetAll(t, db, &CronJob{})
	for _, j := range testCronJobs {
		var found bool = false
		for _, aj := range allCronJobs {
			if aj.(*CronJob).Handle == j.Handle {
				found = true
				assert.Equal(t, j, aj)
			}
		}
		if !found {
			t.Fatalf("cron job not found with handle %v", j.Handle)
		}
	}
	//Delete test
	testDelete(t, db, testJobs[0])
	testDelete(t, db, testCronJobs[0])
	allJobs = testGetAll(t, db, &Job{})
	allCronJobs = testGetAll(t, db, &CronJob{})
	assert.Equal(t, len(testJobs)-1, len(allJobs))
	assert.Equal(t, len(testCronJobs)-1, len(allCronJobs))
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
				testAdd(t, db, j)
				testGet(t, db, j)
			}
			testGetAll(t, db, &Job{})
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
				testDelete(t, db, j)
			}
		}()
	}
	wg.Wait()
}

func testAdd(t *testing.T, store ItemQueue, j DbItem) {
	if err := store.Add(j); err != nil {
		t.Errorf("failed to addjob, err:%s", err.Error())
	}
}

func testDelete(t *testing.T, store ItemQueue, j DbItem) {
	if err := store.Delete(j); err != nil {
		t.Errorf("failed to addjob, err:%s", err.Error())
	}
}

func testGet(t *testing.T, store ItemQueue, j DbItem) {
	err := store.Get(j)
	if err != nil {
		t.Errorf("failed to get items, err:%s", err.Error())
	}
}

func testGetAll(t *testing.T, store ItemQueue, itemType DbItem) []DbItem {
	items, err := store.GetAll(itemType)
	if err != nil {
		t.Fatalf("failed to get items, err:%s", err.Error())
	}
	return items
}
