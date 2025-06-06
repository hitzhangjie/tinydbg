package cmds

import "github.com/spf13/cobra"

var logCommand = &cobra.Command{
	Use:   "log",
	Short: "Help about logging flags.",
	Long: `Logging can be enabled by specifying the --log flag and using the
--log-output flag to select which components should produce logs.

The argument of --log-output must be a comma separated list of component
names selected from this list:


	debugger	Log debugger commands
	debuglineerr	Log recoverable errors reading .debug_line
	rpc		Log all RPC messages
	fncall		Log function call protocol
	stack           Log stacktracer

Additionally --log-dest can be used to specify where the logs should be
written. 
If the argument is a number it will be interpreted as a file descriptor,
otherwise as a file path.
This option will also redirect the "server listening at" message in headless
and dap modes.

`,
}
