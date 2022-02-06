package terminal

//lint:file-ignore ST1005 errors here can be capitalized

import (
	"fmt"
	"io"
	"net/rpc"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/derekparker/trie"
	"github.com/peterh/liner"

	"github.com/hitzhangjie/dlv/pkg/config"
	"github.com/hitzhangjie/dlv/pkg/locspec"
	"github.com/hitzhangjie/dlv/pkg/log"
	"github.com/hitzhangjie/dlv/pkg/terminal/colorize"
	"github.com/hitzhangjie/dlv/pkg/terminal/starbind"
	"github.com/hitzhangjie/dlv/service"
	"github.com/hitzhangjie/dlv/service/api"
)

// Term represents the terminal running dlv.
type Term struct {
	client       service.Client
	conf         *config.Config
	prompt       string
	line         *liner.State
	cmds         *Commands
	stdout       io.Writer
	displays     []displayEntry
	colorEscapes map[colorize.Style]string

	starlarkEnv *starbind.Env

	substitutePathRulesCache [][2]string

	// quitContinue is set to true by exitCommand to signal that the process
	// should be resumed before quitting.
	quitContinue bool

	longCommandMu         sync.Mutex
	longCommandCancelFlag bool

	quittingMutex sync.Mutex
	quitting      bool
}

type displayEntry struct {
	expr   string
	fmtstr string
}

// New returns a new Term.
func New(client service.Client, conf *config.Config) *Term {
	cmds := DebugCommands(client)
	if conf != nil && conf.Aliases != nil {
		cmds.Merge(conf.Aliases)
	}

	if conf == nil {
		conf = &config.Config{}
	}

	t := &Term{
		client:       client,
		conf:         conf,
		prompt:       "(dlv) ",
		line:         liner.NewLiner(),
		cmds:         cmds,
		stdout:       os.Stdout,
		colorEscapes: buildColorEscapes(conf),
	}

	if client != nil {
		cfg := t.loadConfig()
		client.SetReturnValuesLoadConfig(&cfg)
	}

	t.starlarkEnv = starbind.New(starlarkContext{t})
	return t
}

// Close returns the terminal to its previous mode.
func (t *Term) Close() {
	t.line.Close()
}

func (t *Term) sigintGuard(ch <-chan os.Signal, multiClient bool) {
	for range ch {
		t.longCommandCancel()
		t.starlarkEnv.Cancel()
		state, err := t.client.GetStateNonBlocking()
		if err == nil && state.Recording {
			log.Info("received SIGINT, stopping recording (will not forward signal)")
			err := t.client.StopRecording()
			if err != nil {
				log.Error("%v", err)
			}
			continue
		}
		if err == nil && state.CoreDumping {
			log.Info("received SIGINT, stopping dump")
			err := t.client.CoreDumpCancel()
			if err != nil {
				log.Error("%v", err)
			}
			continue
		}
		if multiClient {
			answer, err := t.line.Prompt("Would you like to [p]ause the target (returning to Delve's prompt) or [q]uit this client (leaving the target running) [p/q]? ")
			if err != nil {
				log.Error("%v", err)
				continue
			}
			answer = strings.TrimSpace(answer)
			switch answer {
			case "p":
				_, err := t.client.Halt()
				if err != nil {
					log.Error("%v", err)
				}
			case "q":
				t.quittingMutex.Lock()
				t.quitting = true
				t.quittingMutex.Unlock()
				err := t.client.Disconnect(false)
				if err != nil {
					log.Error("%v", err)
				} else {
					t.Close()
				}
			default:
				log.Info("only p or q allowed")
			}

		} else {
			log.Info("received SIGINT, stopping process (will not forward signal)")
			_, err := t.client.Halt()
			if err != nil {
				log.Error("%v", err)
			}
		}
	}
}

// Run begins running dlv in the terminal.
func (t *Term) Run() (int, error) {
	defer t.Close()

	multiClient := t.client.IsMulticlient()

	// Send the debugger a halt command on SIGINT
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT)
	go t.sigintGuard(ch, multiClient)

	fns := trie.New()
	cmds := trie.New()
	funcs, _ := t.client.ListFunctions("")
	for _, fn := range funcs {
		fns.Add(fn, nil)
	}
	for _, cmd := range t.cmds.cmds {
		for _, alias := range cmd.aliases {
			cmds.Add(alias, nil)
		}
	}

	t.line.SetCompleter(func(line string) (c []string) {
		cmd := t.cmds.Find(strings.Split(line, " ")[0], noPrefix)
		switch cmd.aliases[0] {
		case "break", "trace", "continue":
			if spc := strings.LastIndex(line, " "); spc > 0 {
				prefix := line[:spc] + " "
				funcs := fns.FuzzySearch(line[spc+1:])
				for _, f := range funcs {
					c = append(c, prefix+f)
				}
			}
		case "nullcmd", "nocmd":
			commands := cmds.FuzzySearch(strings.ToLower(line))
			c = append(c, commands...)
		}
		return
	})

	log.Info("Type 'help' for list of commands.")

	var lastCmd string

	// Ensure that the target process is neither running nor recording by
	// making a blocking call.
	_, _ = t.client.GetState()

	for {
		cmdstr, err := t.promptForInput()
		if err != nil {
			if err == io.EOF {
				log.Error("exit")
				return t.handleExit()
			}
			return 1, fmt.Errorf("Prompt for input failed.")
		}

		// if pressing <enter>, using last command instead
		if strings.TrimSpace(cmdstr) == "" {
			cmdstr = lastCmd
		}

		lastCmd = cmdstr

		if err := t.cmds.Call(cmdstr, t); err != nil {
			if _, ok := err.(ExitRequestError); ok {
				return t.handleExit()
			}
			// The type information gets lost in serialization / de-serialization,
			// so we do a string compare on the error message to see if the process
			// has exited, or if the command actually failed.
			if strings.Contains(err.Error(), "exited") {
				fmt.Fprintln(os.Stderr, err.Error())
			} else {
				t.quittingMutex.Lock()
				quitting := t.quitting
				t.quittingMutex.Unlock()
				if quitting {
					return t.handleExit()
				}
				log.Error("Command failed: %s", err)
			}
		}
	}
}

// Substitutes directory to source file.
//
// Ensures that only directory is substituted, for example:
// substitute from `/dir/subdir`, substitute to `/new`
// for file path `/dir/subdir/file` will return file path `/new/file`.
// for file path `/dir/subdir-2/file` substitution will not be applied.
//
// If more than one substitution rule is defined, the rules are applied
// in the order they are defined, first rule that matches is used for
// substitution.
func (t *Term) substitutePath(path string) string {
	if t.conf == nil {
		return path
	}
	return locspec.SubstitutePath(path, t.substitutePathRules())
}

func (t *Term) substitutePathRules() [][2]string {
	if t.substitutePathRulesCache != nil {
		return t.substitutePathRulesCache
	}
	if t.conf == nil || t.conf.SubstitutePath == nil {
		return nil
	}
	spr := make([][2]string, 0, len(t.conf.SubstitutePath))
	for _, r := range t.conf.SubstitutePath {
		spr = append(spr, [2]string{r.From, r.To})
	}
	t.substitutePathRulesCache = spr
	return spr
}

// formatPath applies path substitution rules and shortens the resulting
// path by replacing the current directory with './'
func (t *Term) formatPath(path string) string {
	path = t.substitutePath(path)
	workingDir, _ := os.Getwd()
	return strings.Replace(path, workingDir, ".", 1)
}

func (t *Term) promptForInput() (string, error) {
	l, err := t.line.Prompt(t.prompt)
	if err != nil {
		return "", err
	}

	l = strings.TrimSuffix(l, "\n")
	if l != "" {
		t.line.AppendHistory(l)
	}

	return l, nil
}

func yesno(line *liner.State, question string) (bool, error) {
	for {
		answer, err := line.Prompt(question)
		if err != nil {
			return false, err
		}
		answer = strings.ToLower(strings.TrimSpace(answer))
		switch answer {
		case "n", "no":
			return false, nil
		case "y", "yes":
			return true, nil
		}
	}
}

func (t *Term) handleExit() (int, error) {
	t.quittingMutex.Lock()
	quitting := t.quitting
	t.quittingMutex.Unlock()
	if quitting {
		return 0, nil
	}

	s, err := t.client.GetState()
	if err != nil {
		if isErrProcessExited(err) {
			if t.client.IsMulticlient() {
				answer, err := yesno(t.line, "Remote process has exited. Would you like to kill the headless instance? [Y/n] ")
				if err != nil {
					return 2, io.EOF
				}
				if answer {
					if err := t.client.Detach(true); err != nil {
						return 1, err
					}
				}
				return 0, err
			}
			return 0, nil
		}
		return 1, err
	}
	if !s.Exited {
		if t.quitContinue {
			err := t.client.Disconnect(true)
			if err != nil {
				return 2, err
			}
			return 0, nil
		}

		doDetach := true
		if t.client.IsMulticlient() {
			answer, err := yesno(t.line, "Would you like to kill the headless instance? [Y/n] ")
			if err != nil {
				return 2, io.EOF
			}
			doDetach = answer
		}

		if doDetach {
			kill := true
			if t.client.AttachedToExistingProcess() {
				answer, err := yesno(t.line, "Would you like to kill the process? [Y/n] ")
				if err != nil {
					return 2, io.EOF
				}
				kill = answer
			}
			if err := t.client.Detach(kill); err != nil {
				return 1, err
			}
		}
	}
	return 0, nil
}

// loadConfig returns an api.LoadConfig with the parameterss specified in
// the configuration file.
func (t *Term) loadConfig() api.LoadConfig {
	r := api.LoadConfig{FollowPointers: true, MaxVariableRecurse: 1, MaxStringLen: 64, MaxArrayValues: 64, MaxStructFields: -1}

	if t.conf != nil && t.conf.MaxStringLen != nil {
		r.MaxStringLen = *t.conf.MaxStringLen
	}
	if t.conf != nil && t.conf.MaxArrayValues != nil {
		r.MaxArrayValues = *t.conf.MaxArrayValues
	}
	if t.conf != nil && t.conf.MaxVariableRecurse != nil {
		r.MaxVariableRecurse = *t.conf.MaxVariableRecurse
	}

	return r
}

func (t *Term) removeDisplay(n int) error {
	if n < 0 || n >= len(t.displays) {
		return fmt.Errorf("%d is out of range", n)
	}
	t.displays[n] = displayEntry{"", ""}
	for i := len(t.displays) - 1; i >= 0; i-- {
		if t.displays[i].expr != "" {
			t.displays = t.displays[:i+1]
			return nil
		}
	}
	t.displays = t.displays[:0]
	return nil
}

func (t *Term) addDisplay(expr, fmtstr string) {
	t.displays = append(t.displays, displayEntry{expr: expr, fmtstr: fmtstr})
}

func (t *Term) printDisplay(i int) {
	expr, fmtstr := t.displays[i].expr, t.displays[i].fmtstr
	val, err := t.client.EvalVariable(api.EvalScope{GoroutineID: -1}, expr, ShortLoadConfig)
	if err != nil {
		if isErrProcessExited(err) {
			return
		}
		log.Error("%d: %s = error %v", i, expr, err)
		return
	}
	log.Info("%d: %s = %s", i, val.Name, val.SinglelineStringFormatted(fmtstr))
}

func (t *Term) printDisplays() {
	for i := range t.displays {
		if t.displays[i].expr != "" {
			t.printDisplay(i)
		}
	}
}

func (t *Term) longCommandCancel() {
	t.longCommandMu.Lock()
	defer t.longCommandMu.Unlock()
	t.longCommandCancelFlag = true
}

func (t *Term) longCommandStart() {
	t.longCommandMu.Lock()
	defer t.longCommandMu.Unlock()
	t.longCommandCancelFlag = false
}

func (t *Term) longCommandCanceled() bool {
	t.longCommandMu.Lock()
	defer t.longCommandMu.Unlock()
	return t.longCommandCancelFlag
}

// isErrProcessExited returns true if `err` is an RPC error equivalent of proc.ErrProcessExited
func isErrProcessExited(err error) bool {
	rpcError, ok := err.(rpc.ServerError)
	return ok && strings.Contains(rpcError.Error(), "has exited with status")
}
