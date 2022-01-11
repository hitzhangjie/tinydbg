package cmds

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Deprecated 'run' subcommand.
var runCommand = &cobra.Command{
	Use:   "run",
	Short: "Deprecated command. Use 'debug' instead.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("This command is deprecated, please use 'debug' instead.")
		os.Exit(0)
	},
}

func init() {
	rootCommand.AddCommand(runCommand)
}
