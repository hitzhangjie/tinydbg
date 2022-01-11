package cmds

import "github.com/spf13/cobra"

func init() {
	rootCommand.AddCommand(&cobra.Command{
		Use:   "backend",
		Short: "Help about the --backend flag.",
		Long: `The --backend flag specifies which backend should be used, possible values
are:

	default		Uses lldb on macOS, native everywhere else.
	native		Native backend.
	lldb		Uses lldb-server or debugserver.
	rr		Uses mozilla rr (https://github.com/mozilla/rr).

`})
}
