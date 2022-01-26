package cmds

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/hitzhangjie/dlv/pkg/config"
	"github.com/hitzhangjie/dlv/service/debugger"
)

// 'exec' subcommand.
var execCommand = &cobra.Command{
	Use:   "exec <path/to/binary>",
	Short: "Execute a precompiled binary, and begin a debug session.",
	Long: `Execute a precompiled binary and begin a debug session.

This command will cause Delve to exec the binary and immediately attach to it to
begin a new debug session. Please note that if the binary was not compiled with
optimizations disabled, it may be difficult to properly debug it. Please
consider compiling debugging binaries with -gcflags="all=-N -l" on Go 1.10
or later, -gcflags="-N -l" on earlier versions of Go.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("you must provide a path to a binary")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		conf, err := config.LoadConfig()
		if err != nil {
			panic(err)
		}
		os.Exit(execute(0, args, conf, "", debugger.ExecutingExistingFile, args))
	},
}

func init() {
	execCommand.Flags().BoolVar(&continueOnStart, "continue", false, "Continue the debugged process on start.")
	rootCommand.AddCommand(execCommand)
}
