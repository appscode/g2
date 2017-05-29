//go:generate stringer -type=PT ../../pkg/runtime/protocol.go
package main

import (
	_ "net/http/pprof"
	"os"

	gearmand "github.com/appscode/g2/pkg/server"
	"github.com/appscode/go/flags"
	"github.com/appscode/go/runtime"
	logs "github.com/appscode/log/golog"
	"github.com/spf13/pflag"
)

func main() {
	cfg := &gearmand.Config{}

	pflag.StringVar(&cfg.ListenAddr, "addr", ":4730", "listening on, such as 0.0.0.0:4730")
	pflag.StringVar(&cfg.Storage, "storage-dir", os.TempDir()+"/gearmand", "Directory where LevelDB file is stored.")
	pflag.StringVar(&cfg.WebAddress, "web.addr", ":3000", "Server HTTP api Address")

	defer runtime.HandleCrash()

	flags.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	flags.DumpAll()

	gearmand.NewServer(cfg).Start()
}
