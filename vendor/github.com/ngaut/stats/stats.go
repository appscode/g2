//
// The package is sometimes only imported for the side effect of
// registering its HTTP handler and the above variables.  To use it
// this way, link this package into your program:
//	import _ "expvar"
//
package stats

import (
	"bytes"
	"container/list"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sync"
	"time"
)

type KeyValue struct {
	key   string
	value interface{}
}

type LimitList struct {
	l          *list.List
	capability int
}

func NewLimitList(capability int) *LimitList {
	return &LimitList{l: list.New(), capability: capability}
}

func (self *LimitList) PushBack(value interface{}) {
	if self.l.Len() == self.capability {
		self.l.Remove(self.l.Front())
	}
	self.l.PushBack(value)
}

func (self *LimitList) Front() *list.Element {
	return self.l.Front()
}

func (self *LimitList) Len() int {
	return self.l.Len()
}

func (self *LimitList) MarshalJSON() ([]byte, error) {
	length := self.Len()
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("[ "))

	i := 0
	for e := self.Front(); e != nil; e = e.Next() {
		if _, ok := e.Value.(string); ok {
			buf.WriteString(fmt.Sprintf("\"%v\"", e.Value))
		} else {
			buf.WriteString(fmt.Sprintf("%v", e.Value))
		}

		if i < length-1 {
			buf.WriteString(fmt.Sprintf(", "))
		}
		i++
	}

	buf.WriteString(fmt.Sprintf(" ]"))

	return buf.Bytes(), nil
}

// All published variables.
var (
	mutex sync.RWMutex
	vars  map[string]*LimitList = make(map[string]*LimitList)
)

const (
	maxCapability = 100
)

// Publish declares a named exported variable. This should be called from a
// package's init function when it creates its Vars. If the name is already
// registered then this will log.Panic.
func Publish(name string, v string) {
	mutex.Lock()
	defer mutex.Unlock()

	l, ok := vars[name]
	if !ok {
		l = NewLimitList(maxCapability)
		vars[name] = l
	}

	l.PushBack(v)
}

func PubInt(name string, n int) {
	PubInt64(name, int64(n))
}

func PubInt64(name string, n int64) {
	mutex.Lock()
	defer mutex.Unlock()

	l, ok := vars[name]
	if !ok {
		l = NewLimitList(maxCapability)
		vars[name] = l
	}

	l.PushBack(n)
}

func Inc(key string) {
	mutex.Lock()
	defer mutex.Unlock()

	l, ok := vars[key]
	if !ok {
		l = NewLimitList(1)
		vars[key] = l
		l.PushBack(int64(1))
	} else {
		n := l.Front().Value.(int64)
		n++
		l.PushBack(n)
	}
}

func Dec(key string) {
	mutex.Lock()
	defer mutex.Unlock()

	l, ok := vars[key]
	if !ok {
		l = NewLimitList(1)
		vars[key] = l
		l.PushBack(int64(-1))
	} else {
		n := l.Front().Value.(int64)
		n--
		l.PushBack(n)
	}
}

func ExpvarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	fmt.Fprintf(w, "{\n")

	mutex.RLock()
	defer mutex.RUnlock()

	i := 0
	for k, l := range vars {
		fmt.Fprintf(w, "\"%v\": ", k)

		b, err := l.MarshalJSON()
		if err != nil {
			println("MarshalJSON failed" + err.Error())
		}

		fmt.Fprintf(w, string(b))

		if i < len(vars)-1 {
			fmt.Fprintf(w, ",")
		}

		fmt.Fprintf(w, "\n\n")
		i++
	}

	fmt.Fprintf(w, "}\n")
}


type DebugStatus struct {
	BinaryName string
	State      string
	Hostname   string
	StartTime  time.Time
	Key        string
}

func ShowStatus(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.FormValue("key")
	tmpl, err := template.New("status.tpl").ParseFiles("status.tpl")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hn, _ := os.Hostname()

	err = tmpl.Execute(w, &DebugStatus{
		BinaryName: "scheduler", State: "runing", Hostname: hn, StartTime: time.Now(), Key: key,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func init() {
	http.HandleFunc("/debug/charts", ShowStatus)
	http.HandleFunc("/debug/stats", ExpvarHandler)
}
