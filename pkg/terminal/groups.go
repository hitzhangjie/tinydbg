package terminal

type commandGroup uint8

const (
	otherCmds commandGroup = iota
	breakCmds
	runCmds
	dataCmds
	goroutineCmds
	stackCmds
	sourceCmds
)

type commandGroupDescription struct {
	description string
	group       commandGroup
}

var commandGroupDescriptions = []commandGroupDescription{
	{"Running the program", runCmds},
	{"Manipulating breakpoints", breakCmds},
	{"Inspect program variables and memory", dataCmds},
	{"Inspect the call stack and selecting frames", stackCmds},
	{"Viewing source and disassembly, Listing pkgs, funcs, types", sourceCmds},
	{"Listing and switching between threads and goroutines", goroutineCmds},
	{"Other commands", otherCmds},
}
