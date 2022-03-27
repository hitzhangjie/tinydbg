package proc

import (
	"errors"
	"fmt"
)

var (
	// ErrCouldNotDetermineRelocation is an error returned when Delve could not determine the base address of a
	// position independent executable.
	ErrCouldNotDetermineRelocation = errors.New("could not determine the base address of a PIE")

	// ErrNoDebugInfoFound is returned when Delve cannot open the debug_info
	// section or find an external debug info file.
	ErrNoDebugInfoFound = errors.New("could not open debug info")
)

// ErrFunctionNotFound is returned when failing to find the
// function named 'FuncName' within the binary.
type ErrFunctionNotFound struct {
	FuncName string
}

func (err *ErrFunctionNotFound) Error() string {
	return fmt.Sprintf("could not find function %s\n", err.FuncName)
}
