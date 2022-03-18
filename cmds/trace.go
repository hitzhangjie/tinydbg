package cmds

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hitzhangjie/dlv/pkg/gobuild"
	"github.com/hitzhangjie/dlv/pkg/log"
	"github.com/hitzhangjie/dlv/pkg/terminal"
	"github.com/hitzhangjie/dlv/service"
	"github.com/hitzhangjie/dlv/service/api"
	"github.com/hitzhangjie/dlv/service/debugger"
)

// 'trace' subcommand.
var traceCommand = &cobra.Command{
	Use:   "trace [package] regexp",
	Short: "Compile and begin tracing program.",
	Long: `Trace program execution.

The trace sub command will set a tracepoint on every function matching the
provided regular expression and output information when tracepoint is hit.  This
is useful if you do not want to begin an entire debug session, but merely want
to know what functions your process is executing.

The output of the trace sub command is printed to stderr, so if you would like to
only see the output of the trace operations you can redirect stdout.`,
	Run: traceCmdRun,
}

func init() {
	traceCommand.Flags().IntVarP(&traceAttachPid, "pid", "p", 0, "Pid to attach to.")
	traceCommand.Flags().StringVarP(&traceExecFile, "exec", "e", "", "Binary file to exec and trace.")
	traceCommand.Flags().BoolVarP(&traceTestBinary, "test", "t", false, "Trace a test binary.")
	traceCommand.Flags().BoolVarP(&traceUseEBPF, "ebpf", "", false, "Trace using eBPF (experimental).")
	traceCommand.Flags().IntVarP(&traceStackDepth, "stack", "s", 0, "Show stack trace with given depth. (Ignored with -ebpf)")
	traceCommand.Flags().String("output", "debug", "Output path for the binary.")
	rootCommand.AddCommand(traceCommand)
}

func traceCmdRun(cmd *cobra.Command, args []string) {
	os.Exit(status(cmd, args))
}

func status(cmd *cobra.Command, args []string) int {
	if headless {
		log.Error("Warning: headless mode not supported with trace")
	}
	if acceptMulti {
		log.Error("Warning: accept multiclient mode not supported with trace")
	}

	var regexp string
	var processArgs []string

	dlvArgs, targetArgs := splitArgs(cmd, args)
	var dlvArgsLen = len(dlvArgs)
	if dlvArgsLen == 1 {
		regexp = args[0]
		dlvArgs = dlvArgs[0:0]
	} else if dlvArgsLen >= 2 {
		regexp = dlvArgs[dlvArgsLen-1]
		dlvArgs = dlvArgs[:dlvArgsLen-1]
	}

	var debugname string
	if traceAttachPid == 0 {
		if dlvArgsLen >= 2 && traceExecFile != "" {
			log.Error("Cannot specify package when using exec.")
			return 1
		}

		debugname = traceExecFile
		if traceExecFile == "" {
			debugexe, ok := buildBinary(cmd, dlvArgs, traceTestBinary)
			if !ok {
				return 1
			}
			debugname = debugexe
			defer gobuild.Remove(debugname)
		}

		processArgs = append([]string{debugname}, targetArgs...)
	}

	// Make a local in-memory connection that client and server use to communicate
	listener, clientConn := service.ListenerPipe()
	defer listener.Close()

	if workingDir == "" {
		workingDir = "."
	}

	// Create and start a debug server
	server := service.NewServer(&service.Config{
		Listener:    listener,
		ProcessArgs: processArgs,
		DebuggerConfig: debugger.Config{
			AttachPid:      traceAttachPid,
			WorkingDir:     workingDir,
			CheckGoVersion: checkGoVersion,
		},
	})
	if err := server.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	client := service.NewClientFromConn(clientConn)
	funcs, err := client.ListFunctions(regexp)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := createTracepoints(client, funcs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	cmds := terminal.DebugCommands(client)
	t := terminal.New(client, nil)
	defer t.Close()
	if traceUseEBPF {
		done := make(chan struct{})
		defer close(done)
		go printTracepointHits(client, done)
	}
	cmds.Call("continue", t)
	return 0
}

func createTracepoints(client *service.RPCClient, funcs []string) error {
	for i := range funcs {
		// use EBPF based tracing
		if traceUseEBPF {
			if err := client.CreateEBPFTracepoint(funcs[i]); err != nil {
				return err
			}
			continue
		}

		// fall back to breakpoint based tracing if we get an error.
		_, err := client.CreateBreakpoint(&api.Breakpoint{
			FunctionName: funcs[i],
			Tracepoint:   true,
			Line:         -1,
			Stacktrace:   traceStackDepth,
			LoadArgs:     &terminal.ShortLoadConfig,
		})
		if err != nil && !isBreakpointExistsErr(err) {
			return err
		}
		addrs, err := client.FunctionReturnLocations(funcs[i])
		if err != nil {
			return err
		}
		for i := range addrs {
			_, err = client.CreateBreakpoint(&api.Breakpoint{
				Addr:        addrs[i],
				TraceReturn: true,
				Stacktrace:  traceStackDepth,
				Line:        -1,
				LoadArgs:    &terminal.ShortLoadConfig,
			})
			if err != nil && !isBreakpointExistsErr(err) {
				return err
			}
		}
	}
	return nil
}

func printTracepointHits(client *service.RPCClient, done chan struct{}) {
	// BPF generates event after entering traced func or
	// before leaving traced func.
	//
	// So we use this map `gFnEntrySeen` to record if traced
	// function is entered or not, then we can differentiate
	// the parameters belongs to input or output ones.
	gFnEntrySeen := map[int]struct{}{}
	for {
		select {
		case <-done:
			return
		default:
		}

		tracepoints, err := client.GetBufferedTracepoints()
		if err != nil {
			panic(err)
		}
		for _, t := range tracepoints {
			var params strings.Builder
			for _, p := range t.InputParams {
				if params.Len() > 0 {
					params.WriteString(", ")
				}
				if p.Kind == reflect.String {
					params.WriteString(fmt.Sprintf("%q", p.Value))
				} else {
					params.WriteString(p.Value)
				}
			}
			_, seen := gFnEntrySeen[t.GoroutineID]
			if seen {
				for _, p := range t.ReturnParams {
					log.Error("=> %#v", p.Value)
				}
				delete(gFnEntrySeen, t.GoroutineID)
			} else {
				gFnEntrySeen[t.GoroutineID] = struct{}{}
				log.Error("> (%d) %s(%s)", t.GoroutineID, t.FunctionName, params.String())
			}
		}
	}
}

func isBreakpointExistsErr(err error) bool {
	return strings.Contains(err.Error(), "Breakpoint exists")
}
