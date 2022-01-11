package cmds

import (
	"fmt"
	"net"
	"os"

	"github.com/spf13/cobra"

	"github.com/hitzhangjie/dlv/pkg/config"
	"github.com/hitzhangjie/dlv/pkg/logflags"
	"github.com/hitzhangjie/dlv/service"
	"github.com/hitzhangjie/dlv/service/dap"
	"github.com/hitzhangjie/dlv/service/debugger"
)

// 'dap' subcommand.
var dapCommand = &cobra.Command{
	Use:   "dap",
	Short: "[EXPERIMENTAL] Starts a headless TCP server communicating via Debug Adaptor Protocol (DAP).",
	Long: `[EXPERIMENTAL] Starts a headless TCP server communicating via Debug Adaptor Protocol (DAP).

The server is always headless and requires a DAP client like vscode to connect and request a binary
to be launched or process to be attached to. The following modes can be specified via client's launch config:
- launch + exec (executes precompiled binary, like 'dlv exec')
- launch + debug (builds and launches, like 'dlv debug')
- launch + test (builds and tests, like 'dlv test')
- launch + replay (replays an rr trace, like 'dlv replay')
- launch + core (replays a core dump file, like 'dlv core')
- attach + local (attaches to a running process, like 'dlv attach')
Program and output binary paths will be interpreted relative to dlv's working directory.

The server does not yet accept multiple client connections (--accept-multiclient).
While --continue is not supported, stopOnEntry launch/attach attribute can be used to control if
execution is resumed at the start of the debug session.

The --client-addr flag is a special flag that makes the server initiate a debug session
by dialing in to the host:port where a DAP client is waiting. This server process
will exit when the debug session ends.`,
	Run: dapCmdRun,
}

func init() {
	dapCommand.Flags().StringVar(&dapClientAddr, "client-addr", "", "host:port where the DAP client is waiting for the DAP server to dial in")

	// TODO(polina): support --tty when dlv dap allows to launch a program from command-line
	rootCommand.AddCommand(dapCommand)
}

func dapCmdRun(cmd *cobra.Command, args []string) {
	conf, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	status := func() int {
		if err := logflags.Setup(log, logOutput, logDest); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		defer logflags.Close()

		if loadConfErr != nil {
			logflags.DebuggerLogger().Errorf("%v", loadConfErr)
		}

		if cmd.Flag("headless").Changed {
			fmt.Fprintf(os.Stderr, "Warning: dap mode is always headless\n")
		}
		if acceptMulti {
			fmt.Fprintf(os.Stderr, "Warning: accept-multiclient mode not supported with dap\n")
		}
		if initFile != "" {
			fmt.Fprint(os.Stderr, "Warning: init file ignored with dap\n")
		}
		if continueOnStart {
			fmt.Fprintf(os.Stderr, "Warning: continue ignored with dap; specify via launch/attach request instead\n")
		}
		if backend != "default" {
			fmt.Fprintf(os.Stderr, "Warning: backend ignored with dap; specify via launch/attach request instead\n")
		}
		if buildFlags != "" {
			fmt.Fprintf(os.Stderr, "Warning: build flags ignored with dap; specify via launch/attach request instead\n")
		}
		if workingDir != "" {
			fmt.Fprintf(os.Stderr, "Warning: working directory ignored with dap: specify via launch request instead\n")
		}
		dlvArgs, targetArgs := splitArgs(cmd, args)
		if len(dlvArgs) > 0 {
			fmt.Fprintf(os.Stderr, "Warning: debug arguments ignored with dap; specify via launch/attach request instead\n")
		}
		if len(targetArgs) > 0 {
			fmt.Fprintf(os.Stderr, "Warning: program flags ignored with dap; specify via launch/attach request instead\n")
		}

		disconnectChan := make(chan struct{})
		config := &service.Config{
			DisconnectChan: disconnectChan,
			Debugger: debugger.Config{
				Backend:              backend,
				Foreground:           true, // server always runs without terminal client
				DebugInfoDirectories: conf.DebugInfoDirectories,
				CheckGoVersion:       checkGoVersion,
			},
			CheckLocalConnUser: checkLocalConnUser,
		}
		var conn net.Conn
		if dapClientAddr == "" {
			listener, err := net.Listen("tcp", addr)
			if err != nil {
				fmt.Printf("couldn't start listener: %s\n", err)
				return 1
			}
			config.Listener = listener
		} else { // with a predetermined client.
			var err error
			conn, err = net.Dial("tcp", dapClientAddr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to connect to the DAP client: %v\n", err)
				return 1
			}
		}

		server := dap.NewServer(config)
		defer server.Stop()
		if conn == nil {
			server.Run()
		} else { // work with a predetermined client.
			server.RunWithClient(conn)
		}
		waitForDisconnectSignal(disconnectChan)
		return 0
	}()

	os.Exit(status)
}
