package cmds

import (
	"errors"
	"fmt"
	"os"

	"github.com/hitzhangjie/tinydbg/pkg/logflags"
	"github.com/spf13/cobra"
)

// 'connect' subcommand.
var connectCommand = &cobra.Command{
	Use:   "connect addr",
	Short: "Connect to a headless debug server with a terminal client.",
	Long:  "Connect to a running headless debug server with a terminal client. Prefix with 'unix:' to use a unix domain socket.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("you must provide an address as the first argument")
		}
		return nil
	},
	Run:               connectCmd,
	ValidArgsFunction: cobra.NoFileCompletions,
}

func connectCmd(_ *cobra.Command, args []string) {
	if err := logflags.Setup(enableLogging, enableLoggers, logDest); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
		return
	}
	if loadConfErr != nil {
		logflags.LogDebuggerLogger().Errorf("%v", loadConfErr)
	}
	addr := args[0]
	if addr == "" {
		fmt.Fprint(os.Stderr, "An empty address was provided. You must provide an address as the first argument.\n")
		logflags.Close()
		os.Exit(1)
	}
	ec := connect(addr, nil, conf)
	logflags.Close()
	os.Exit(ec)
}
