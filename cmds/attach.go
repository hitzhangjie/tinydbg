package cmds

import (
	"errors"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/hitzhangjie/dlv/pkg/config"
	"github.com/hitzhangjie/dlv/pkg/log"
	"github.com/hitzhangjie/dlv/service/debugger"
)

// 'attach' subcommand.
var attachCommand = &cobra.Command{
	Use:   "attach pid [executable]",
	Short: "Attach to running process and begin debugging.",
	Long: `Attach to an already running process and begin debugging it.

This command will cause Delve to take control of an already running process, 
and begin a new debug session. When exiting the debug session you will have 
the option to let the process continue or kill it.
`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("you must provide a PID")
		}
		return nil
	},
	Run: attachCmdRun,
}

func init() {
	attachCommand.Flags().BoolVar(&continueOnStart, "continue", false, "Continue the debugged process on start.")
	rootCommand.AddCommand(attachCommand)
}

func attachCmdRun(cmd *cobra.Command, args []string) {
	conf, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	pid, err := strconv.Atoi(args[0])
	if err != nil {
		log.Error("Invalid pid: %s", args[0])
		os.Exit(1)
	}
	os.Exit(execute(pid, args[1:], conf, "", debugger.ExecutingOther, args, buildFlags))
}
