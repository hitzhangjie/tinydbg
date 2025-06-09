package logflags

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

// which loggers are enabled.
var (
	logAny             = false
	logDebugger        = false
	logDebugLineErrors = false
	logRPCEnabled      = false
	logFnCall          = false
	logStack           = false

	logOut io.WriteCloser
)

// Setup sets debugger flags.
//
// If logDest is not empty logs will be redirected to the file specified by logDest,
// it could be one of the following:
// - file descriptor
// - file path
func Setup(enableLog bool, enableLoggers, logDest string) error {
	if logDest != "" {
		n, err := strconv.Atoi(logDest)
		if err == nil {
			// logDest is a file descriptor
			logOut = os.NewFile(uintptr(n), "delve-logs")
		} else {
			// logDest is a file path
			fh, err := os.Create(logDest)
			if err != nil {
				return fmt.Errorf("could not create log file: %v", err)
			}
			logOut = fh
		}
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	if !enableLog {
		log.SetOutput(io.Discard)
		if enableLoggers != "" {
			return errors.New("--log-output specified without --log")
		}
		return nil
	}
	if enableLoggers == "" {
		enableLoggers = "debugger"
	}
	logAny = true
	v := strings.Split(enableLoggers, ",")
	for _, l := range v {
		// If adding another value, do make sure to
		// update "Help about logging flags" in commands.go.
		switch l {
		case "debugger":
			logDebugger = true
		case "debuglineerr":
			logDebugLineErrors = true
		case "rpc":
			logRPCEnabled = true
		case "fncall":
			logFnCall = true
		case "stack":
			logStack = true
		default:
			fmt.Fprintf(os.Stderr, "Warning: unknown log output value %q, run 'dlv help log' for usage.\n", l)
		}
	}
	return nil
}

// Close closes the logger output.
func Close() {
	if logOut != nil {
		logOut.Close()
	}
}

// LogAny returns true if any logging is enabled.
func LogAny() bool {
	return logAny
}

// LogDebugger returns true if the debugger package should log.
func LogDebugger() bool {
	return logDebugger
}

// LogDebuggerLogger returns a logger for the debugger package.
func LogDebuggerLogger() Logger {
	return makeLogger(logDebugger, "layer", "debugger")
}

// LogDebugLineErrors returns true if pkg/dwarf/line should log its recoverable
// errors.
func LogDebugLineErrors() bool {
	return logDebugLineErrors
}

// LogDebugLineLogger returns a logger for the dwarf/line package.
func LogDebugLineLogger() Logger {
	return makeLogger(logDebugLineErrors, "layer", "dwarf-line")
}

// LogRPC returns true if LogRPC messages should be logged.
func LogRPC() bool {
	return logRPCEnabled
}

// RPCLogger returns a logger for RPC messages.
func RPCLogger() Logger {
	return rpcLogger(logRPCEnabled)
}

// rpcLogger returns a logger for RPC messages set to a specific minimal log level.
func rpcLogger(flag bool) Logger {
	return makeLogger(flag, "layer", "rpc")
}

// FnCall returns true if the function call protocol should be logged.
func FnCall() bool {
	return logFnCall
}

func FnCallLogger() Logger {
	return makeLogger(logFnCall, "layer", "proc", "kind", "fncall")
}

// Stack returns true if the stacktracer should be logged.
func Stack() bool {
	return logStack
}

func StackLogger() Logger {
	return makeLogger(logStack, "layer", "core", "kind", "stack")
}

// WriteAPIListeningMessage writes the "API server listening" message in headless mode.
func WriteAPIListeningMessage(addr net.Addr) {
	writeListeningMessage("API", addr)
}

func writeListeningMessage(server string, addr net.Addr) {
	msg := fmt.Sprintf("%s server listening at: %s", server, addr)
	if logOut != nil {
		fmt.Fprintln(logOut, msg)
	} else {
		fmt.Println(msg)
	}
	tcpAddr, _ := addr.(*net.TCPAddr)
	if tcpAddr == nil || tcpAddr.IP.IsLoopback() {
		return
	}
	logger := rpcLogger(true)
	logger.Warnf("Listening for remote connections (connections are not authenticated nor encrypted)")
}

func WriteError(msg string) {
	if logOut != nil {
		fmt.Fprintln(logOut, msg)
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
}

func WriteCgoFlagsWarning() {
	makeLogger(true, "layer", "dlv").Warn("CGO_CFLAGS already set, Cgo code could be optimized.")
}
