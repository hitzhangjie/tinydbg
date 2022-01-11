package cmds

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/spf13/cobra"

	"github.com/hitzhangjie/dlv/pkg/config"
	"github.com/hitzhangjie/dlv/pkg/logflags"
	"github.com/hitzhangjie/dlv/pkg/terminal"
	"github.com/hitzhangjie/dlv/service/debugger"
	"github.com/hitzhangjie/dlv/service/rpcv2"
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
	Run: connectCmdRun,
}

func init() {
	rootCommand.AddCommand(connectCommand)
}

func connectCmdRun(cmd *cobra.Command, args []string) {
	conf, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	if err := logflags.Setup(log, logOutput, logDest); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
		return
	}
	defer logflags.Close()
	addr := args[0]
	if addr == "" {
		fmt.Fprint(os.Stderr, "An empty address was provided. You must provide an address as the first argument.\n")
		os.Exit(1)
	}
	os.Exit(connect(addr, nil, conf, debugger.ExecutingOther))
}

func connect(addr string, clientConn net.Conn, conf *config.Config, kind debugger.ExecuteKind) int {
	// Create and start a terminal - attach to running instance
	var client *rpcv2.RPCClient
	if clientConn != nil {
		client = rpcv2.NewClientFromConn(clientConn)
	} else {
		client = rpcv2.NewClient(addr)
	}
	if client.IsMulticlient() {
		state, _ := client.GetStateNonBlocking()
		// The error return of GetState will usually be the ErrProcessExited,
		// which we don't care about. If there are other errors they will show up
		// later, here we are only concerned about stopping a running target so
		// that we can initialize our connection.
		if state != nil && state.Running {
			_, err := client.Halt()
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not halt: %v", err)
				return 1
			}
		}
	}
	term := terminal.New(client, conf)
	term.InitFile = initFile
	status, err := term.Run()
	if err != nil {
		fmt.Println(err)
	}
	return status
}
