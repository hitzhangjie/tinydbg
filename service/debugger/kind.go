package debugger

// ExecuteKind execution kind of tracee
type ExecuteKind int

const (
	ExecutingExistingFile  = ExecuteKind(iota) // `dlv exec` execute existed executable
	ExecutingGeneratedFile                     // `dlv debug` run go build and debug, or `dlv core` debug coredump
	ExecutingGeneratedTest                     // `dlv test` run go test -c and debug
	ExecutingOther                             // others debugging mode, like `dlv attach`
)
