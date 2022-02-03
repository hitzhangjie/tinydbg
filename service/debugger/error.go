package debugger

import "errors"

var (
	// ErrCanNotRestart is returned when the target cannot be restarted.
	// This is returned for targets that have been attached to, or when
	// debugging core files.
	ErrCanNotRestart = errors.New("can not restart this target")

	// ErrNotRecording is returned when StopRecording is called while the
	// debugger is not recording the target.
	ErrNotRecording = errors.New("debugger is not recording")

	// ErrCoreDumpInProgress is returned when a core dump is already in progress.
	ErrCoreDumpInProgress = errors.New("core dump in progress")

	// ErrCoreDumpNotSupported is returned when core dumping is not supported
	ErrCoreDumpNotSupported = errors.New("core dumping not supported")
)
