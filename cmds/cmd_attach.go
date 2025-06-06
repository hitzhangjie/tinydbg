package cmds

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/hitzhangjie/tinydbg/service/debugger"
	"github.com/spf13/cobra"
)

// 'attach' subcommand.
var attachCommand = &cobra.Command{
	Use:   "attach pid [executable]",
	Short: "Attach to running process and begin debugging.",
	Long: `Attach to an already running process and begin debugging it.

This command will cause Delve to take control of an already running process, and
begin a new debug session.  When exiting the debug session you will have the
option to let the process continue or kill it.
`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && attachWaitFor == "" {
			return errors.New("you must provide a PID")
		}
		return nil
	},
	Run: attachCmd,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 1 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveDefault
	},
}

func init() {
	attachCommand.Flags().BoolVar(&continueOnStart, "continue", false, "Continue the debugged process on start.")
	attachCommand.Flags().StringVar(&attachWaitFor, "waitfor", "", "Wait for a process with a name beginning with this prefix")
	must(attachCommand.RegisterFlagCompletionFunc("waitfor", cobra.NoFileCompletions))
	attachCommand.Flags().Float64Var(&attachWaitForInterval, "waitfor-interval", 1, "Interval between checks of the process list, in millisecond")
	must(attachCommand.RegisterFlagCompletionFunc("waitfor-interval", cobra.NoFileCompletions))
	attachCommand.Flags().Float64Var(&attachWaitForDuration, "waitfor-duration", 0, "Total time to wait for a process")
	must(attachCommand.RegisterFlagCompletionFunc("waitfor-duration", cobra.NoFileCompletions))
}

func attachCmd(_ *cobra.Command, args []string) {
	var pid int
	if len(args) > 0 {
		var err error
		pid, err = strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid pid: %s\n", args[0])
			os.Exit(1)
		}
		args = args[1:]
	}
	os.Exit(execute(pid, args, conf, "", debugger.ExecutingOther, args, buildFlags))
}
