package cmds

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hitzhangjie/dlv/service/debugger"
)

// 'connect' subcommand.
var connectCommand = &cobra.Command{
	Use:   "connect addr",
	Short: "Connect to a headless debug server.",
	Long:  "Connect to a running headless debug server.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("you must provide an address as the first argument")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		addr := args[0]
		if addr == "" {
			fmt.Fprint(os.Stderr, "An empty address was provided. You must provide an address as the 1st argument.\n")
			os.Exit(1)
		}
		os.Exit(connect(addr, nil, debugger.ExecutingOther))
	},
}

func init() {
	rootCommand.AddCommand(connectCommand)
}
