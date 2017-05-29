package main

import (
	_ "net/http/pprof"
	"os"

	gearmand "github.com/appscode/g2/pkg/server"
	"github.com/appscode/go/runtime"
	"github.com/spf13/cobra"
)

func NewCmdRun() *cobra.Command {
	var cfg gearmand.Config

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run Gearman server",
		Run: func(cmd *cobra.Command, args []string) {
			defer runtime.HandleCrash()
			gearmand.NewServer(cfg).Start()
		},
	}

	cmd.Flags().StringVar(&cfg.ListenAddr, "addr", ":4730", "listening on, such as 0.0.0.0:4730")
	cmd.Flags().StringVar(&cfg.Storage, "storage-dir", os.TempDir()+"/gearmand", "Directory where LevelDB file is stored.")
	cmd.Flags().StringVar(&cfg.WebAddress, "web.addr", ":3000", "Server HTTP api Address")
	return cmd
}
