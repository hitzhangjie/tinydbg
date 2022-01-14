package log

import (
	"fmt"
	"log"
	"runtime"
	"strings"
)

var (
	verbose bool
)

const (
	// LVerbose 显示详细日志信息
	LVerbose = 1 << 10
)

func init() {
	// clear the datetime prefix in default logger
	// `go: x &^ y` is equal to `c: x & ~y`
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
}

// SetFlags 设置log选项
func SetFlags(flags int) {
	log.SetFlags(log.Flags() | flags)
	verbose = (flags & LVerbose) != 0
}

// Debug print logging info at level DEBUG, if flag verbose true, filename and lineno will be logged.
func Debug(format string, vals ...interface{}) {
	fn, _ := callerAddress(3)
	if verbose {
		log.Printf("%s[Debug][%s] %s%s", ColorDebug, fn, fmt.Sprintf(format, vals...), ColorReset)
	}
}

// Info print logging info at level INFO, if flag verbose true, filename and lineno will be logged.
func Info(format string, vals ...interface{}) {
	fn, _ := callerAddress(3)
	if verbose {
		log.Printf("%s[Info][%s] %s%s", ColorGreen, fn, fmt.Sprintf(format, vals...), ColorReset)
	} else {
		log.Printf("%s%s%s", ColorGreen, fmt.Sprintf(format, vals...), ColorReset)
	}
}

// Warn print logging info at level WARN, if flag verbose true, filename and lineno will be logged.
func Warn(format string, vals ...interface{}) {
	fn, _ := callerAddress(3)
	if verbose {
		log.Printf("%s[Error][%s] %s%s", ColorOrange, fn, fmt.Sprintf(format, vals...), ColorReset)
	} else {
		log.Printf("%s%s%s", ColorOrange, fmt.Sprintf(format, vals...), ColorReset)
	}
}

// Error print logging info at level ERROR, if flag verbose true, filename and lineno will be logged.
func Error(format string, vals ...interface{}) {
	fn, _ := callerAddress(3)
	if verbose {
		log.Printf("%s[Error][%s] %s%s", ColorRed, fn, fmt.Sprintf(format, vals...), ColorReset)
	} else {
		log.Printf("%s%s%s", ColorRed, fmt.Sprintf(format, vals...), ColorReset)
	}
}

// callerAddress skip N level to get the caller's filename and lineno, if no caller return error.
func callerAddress(skip int) (string, error) {

	fpcs := make([]uintptr, 1)
	// Skip N levels to get the caller
	n := runtime.Callers(skip, fpcs)
	if n == 0 {
		return "", fmt.Errorf("MSG: NO CALLER")
	}

	caller := runtime.FuncForPC(fpcs[0] - 1)
	if caller == nil {
		return "", fmt.Errorf("MSG: CALLER IS NIL")
	}

	// Print the file name and line number
	fileName, lineNo := caller.FileLine(fpcs[0] - 1)
	baseName := fileName[strings.LastIndex(fileName, "/")+1:]

	return fmt.Sprintf("%s:%d", baseName, lineNo), nil
}
