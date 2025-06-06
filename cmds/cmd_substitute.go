package cmds

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hitzhangjie/tinydbg/service/rpc2"
	"github.com/spf13/cobra"
)

var substituteCommand = &cobra.Command{
	Use:    "substitute-path-guess-helper",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		gsp, err := rpc2.MakeGuessSusbtitutePathIn()
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			os.Exit(1)
		}
		err = json.NewEncoder(os.Stdout).Encode(gsp)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	},
}
