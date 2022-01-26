package cmds

import (
	"fmt"
	"github.com/hitzhangjie/dlv/pkg/terminal"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/hitzhangjie/dlv/pkg/config"
	"github.com/hitzhangjie/dlv/pkg/log"
	"github.com/hitzhangjie/dlv/service"
	"github.com/hitzhangjie/dlv/service/api"
	"github.com/hitzhangjie/dlv/service/debugger"
	"github.com/hitzhangjie/dlv/service/rpccommon"
	"github.com/hitzhangjie/dlv/service/rpcv2"
)

var (
	// debugger mode
	headless        bool // whether to run without terminal
	continueOnStart bool // whether to continue the process on startup
	acceptMulti     bool // whether allows multiple clients to connect to the same server

	// debugger settins
	addr        string // the debugging server listen address
	initFile    string // the path to initialization file
	workingDir  string // the working directory for running the program
	disableASLR bool   // whether disables ASLR
	backend     string // backend selection

	// checkGoVersion is true if the debugger should check the version of Go
	// used to compile the executable and refuse to work on incompatible
	// versions.
	checkGoVersion bool

	// checkLocalConnUser is true if the debugger should check that local
	// connections come from the same user that started the headless server
	checkLocalConnUser bool

	// trace settings
	traceAttachPid  int
	traceExecFile   string
	traceTestBinary bool
	traceStackDepth int
	traceUseEBPF    bool
)

// New returns an initialized command tree.
func New() *cobra.Command {
	return rootCommand
}

// 返回错误码给os.Exit(?)
func execute(attachPid int, processArgs []string, conf *config.Config, coreFile string, kind debugger.ExecuteKind, dlvArgs []string) int {
	if headless && (initFile != "") {
		fmt.Fprint(os.Stderr, "Warning: init file ignored with --headless\n")
	}
	if continueOnStart {
		if !headless {
			fmt.Fprint(os.Stderr, "Error: --continue only works with --headless; use an init file\n")
			return 1
		}
		if !acceptMulti {
			fmt.Fprint(os.Stderr, "Error: --continue requires --accept-multiclient\n")
			return 1
		}
	}

	if !headless && acceptMulti {
		fmt.Fprint(os.Stderr, "Warning accept-multi: ignored\n")
		// acceptMulti won't work in normal (non-headless) mode because we always
		// call server.Stop after the terminal client exits.
		acceptMulti = false
	}

	var listener net.Listener
	var clientConn net.Conn
	var err error

	// Make a TCP listener
	if headless {
		listener, err = net.Listen("tcp", addr)
	} else {
		listener, clientConn = service.ListenerPipe()
	}
	if err != nil {
		log.Error("couldn't start listener: %s", err)
		return 1
	}
	defer listener.Close()

	var server service.Server

	disconnectChan := make(chan struct{})

	if workingDir == "" {
		workingDir = "."
	}

	// Create and start a debugger server
	server = rpccommon.NewServer(&service.Config{
		Listener:           listener,
		ProcessArgs:        processArgs,
		AcceptMulti:        acceptMulti,
		CheckLocalConnUser: checkLocalConnUser,
		DisconnectChan:     disconnectChan,
		Debugger: debugger.Config{
			AttachPid:            attachPid,
			WorkingDir:           workingDir,
			Backend:              backend,
			CoreFile:             coreFile,
			Foreground:           headless,
			Packages:             dlvArgs,
			ExecuteKind:          kind,
			DebugInfoDirectories: conf.DebugInfoDirectories,
			CheckGoVersion:       checkGoVersion,
			DisableASLR:          disableASLR,
		},
	})

	if err := server.Run(); err != nil {
		if err == api.ErrNotExecutable {
			switch kind {
			case debugger.ExecutingGeneratedFile:
				fmt.Fprintln(os.Stderr, "Can not debug non-main package")
				return 1
			case debugger.ExecutingExistingFile:
				log.Error("%s is not executable", processArgs[0])
				return 1
			default:
			}
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	var status int
	if headless {
		if continueOnStart {
			client := rpcv2.NewClient(listener.Addr().String())
			client.Disconnect(true) // true = continue after disconnect
		}
		waitForDisconnectSignal(disconnectChan)
		err = server.Stop()
		if err != nil {
			log.Error("%v", err)
		}

		return status
	}

	return connect(listener.Addr().String(), clientConn, conf, kind)
}

// waitForDisconnectSignal is a blocking function that waits for either
// a SIGINT (Ctrl-C) or SIGTERM (kill -15) OS signal or for disconnectChan
// to be closed by the server when the client disconnects.
// Note that in headless mode, the debugged process is foregrounded
// (to have control of the tty for debugging interactive programs),
// so SIGINT gets sent to the debuggee and not to delve.
func waitForDisconnectSignal(disconnectChan chan struct{}) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ch:
	case <-disconnectChan:
	}
}

func splitArgs(cmd *cobra.Command, args []string) ([]string, []string) {
	if cmd.ArgsLenAtDash() >= 0 {
		return args[:cmd.ArgsLenAtDash()], args[cmd.ArgsLenAtDash():]
	}
	return args, []string{}
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
				log.Error("could not halt: %v", err)
				return 1
			}
		}
	}
	term := terminal.New(client, conf)
	term.InitFile = initFile
	status, err := term.Run()
	if err != nil {
		log.Error("%v", err)
	}
	return status
}
