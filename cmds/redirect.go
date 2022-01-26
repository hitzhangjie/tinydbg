package cmds

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCommand.AddCommand(&cobra.Command{
		Use:   "log",
		Short: "Help about logging flags.",
		Long: `Logging can be enabled by specifying the --log flag and using the
--log-output flag to select which components should produce logs.

The argument of --log-output must be a comma separated list of component
names selected from this list:


	debugger	Log debugger commands
	gdbwire		Log connection to gdbserial backend
	lldbout		Copy output from debugserver/lldb to standard output
	debuglineerr	Log recoverable errors reading .debug_line
	rpcv2		Log all RPC messages
	fncall		Log function call protocol
	minidump	Log minidump loading

Additionally --log-dest can be used to specify where the logs should be
written. 
If the argument is a number it will be interpreted as a file descriptor,
otherwise as a file path.
This option also redirects the "server listening at" message in headless modes.
`,
	})
}

func parseRedirects(redirects []string) ([3]string, error) {
	r := [3]string{}
	names := [3]string{"stdin", "stdout", "stderr"}
	for _, redirect := range redirects {
		idx := 0
		for i, name := range names {
			pfx := name + ":"
			if strings.HasPrefix(redirect, pfx) {
				idx = i
				redirect = redirect[len(pfx):]
				break
			}
		}
		if r[idx] != "" {
			return r, fmt.Errorf("redirect error: %s redirected twice", names[idx])
		}
		r[idx] = redirect
	}
	return r, nil
}
