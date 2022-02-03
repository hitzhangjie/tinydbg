package debugger

// Config provides the configuration to start a Debugger.
//
// Only one of ProcessArgs or AttachPid should be specified. If ProcessArgs is
// provided, a new process will be launched. Otherwise, the debugger will try
// to attach to an existing process with AttachPid.
type Config struct {
	// WorkingDir is working directory of the new process. This field is used
	// only when launching a new process.
	WorkingDir string

	// AttachPid is the PID of an existing process to which the debugger should
	// attach.
	AttachPid int

	// CoreFile specifies the path to the core dump to open.
	CoreFile string

	// Foreground lets target process access stdin.
	Foreground bool

	// CheckGoVersion is true if the debugger should check the version of Go
	// used to compile the executable and refuse to work on incompatible
	// versions.
	CheckGoVersion bool

	// Packages contains the packages that we are debugging.
	Packages []string

	// ExecuteKind contains the kind of the executed program.
	ExecuteKind ExecuteKind

	// DisableASLR disables ASLR
	DisableASLR bool
}
