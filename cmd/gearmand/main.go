//go:generate stringer -type=PT ../../pkg/runtime/protocol.go
package main

import (
	"flag"
	"log"
	"os"

	logs "github.com/appscode/go/log/golog"
	"github.com/appscode/go/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	var rootCmd = &cobra.Command{
		Use: "gearmand",
		PersistentPreRun: func(c *cobra.Command, args []string) {
			c.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Printf("FLAG: --%s=%q", flag.Name, flag.Value)
			})
		},
	}
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	rootCmd.AddCommand(version.NewCmdVersion())
	rootCmd.AddCommand(NewCmdRun())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
