package main

import (
	"flag"
	"fmt"
	"runtime"
	"strconv"

	gearmand "github.com/appscode/g2/pkg/server"
	"github.com/appscode/g2/pkg/storage"
	"github.com/appscode/g2/pkg/storage/leveldb"
	log "github.com/ngaut/logging"
)

var (
	addr       = flag.String("addr", ":4730", "listening on, such as 0.0.0.0:4730")
	path       = flag.String("coredump", "./", "coredump file path")
	storageDir = flag.String("storage-dir", "/tmp", "Directory where LevelDB file is stored.")
)

func main() {
	flag.Lookup("v").DefValue = fmt.Sprint(log.LOG_LEVEL_WARN)
	flag.Parse()
	gearmand.PublishCmdline()
	gearmand.RegisterCoreDump(*path)
	if lv, err := strconv.Atoi(flag.Lookup("v").Value.String()); err == nil {
		log.SetLevel(log.LogLevel(lv))
	}
	//log.SetHighlighting(false)
	runtime.GOMAXPROCS(1)
	var store storage.JobQueue
	if s, err := leveldbq.New(*storageDir); err == nil {
		store = s
	} else {
		log.Info(err)
	}
	gearmand.NewServer(store).Start(*addr)
}
