//go:generate stringer -type=PT ../../pkg/runtime
package main

import (
	"os"

	gearmand "github.com/appscode/g2/pkg/server"
	"github.com/appscode/g2/pkg/storage"
	"github.com/appscode/g2/pkg/storage/leveldb"
	"github.com/appscode/go/flags"
	"github.com/appscode/go/runtime"
	"github.com/appscode/log"
	logs "github.com/appscode/log/golog"
	"github.com/spf13/pflag"
)

var (
	addr        string
	coredumpDir string
	storageDir  string
)

func main() {
	pflag.StringVar(&addr, "addr", ":4730", "listening on, such as 0.0.0.0:4730")
	pflag.StringVar(&coredumpDir, "coredump", "./", "coredump file path")
	pflag.StringVar(&storageDir, "storage-dir", os.TempDir(), "Directory where LevelDB file is stored.")

	flags.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()
	flags.DumpAll()

	gearmand.RegisterCoreDump(coredumpDir)

	// runtime.GOMAXPROCS(1)
	var store storage.Db
	if s, err := leveldbq.New(storageDir); err == nil {
		store = s
	} else {
		log.Info(err)
	}

	defer runtime.HandleCrash()
	gearmand.NewServer(store).Start(addr)
}
