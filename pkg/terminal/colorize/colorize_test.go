package colorize_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"

	"github.com/hitzhangjie/dlv/pkg/terminal/colorize"
)

var src = `package main

// Vehical defines the vehical behavior
type Vehical interface{
	// Run vehical can run in a speed
	Run()
}

// BMWS1000RR defines the motocycle bmw s1000rr
type BMWS1000RR struct {
}

// Run bwm s1000rr run
func (a *BMWS1000RR) Run() {
	println("I can run at 300km/h")
}

func main() {
	var vehical = &BMWS1000RR{}
	vehical.Run()
}
`

const terminalHighlightEscapeCode string = "\033[%2dm"

const (
	ansiBlack     = 30
	ansiRed       = 31
	ansiGreen     = 32
	ansiYellow    = 33
	ansiBlue      = 34
	ansiMagenta   = 35
	ansiCyan      = 36
	ansiWhite     = 37
	ansiBrBlack   = 90
	ansiBrRed     = 91
	ansiBrGreen   = 92
	ansiBrYellow  = 93
	ansiBrBlue    = 94
	ansiBrMagenta = 95
	ansiBrCyan    = 96
	ansiBrWhite   = 97
)

func colorizeCode(code int) string {
	return fmt.Sprintf(terminalHighlightEscapeCode, code)
}

var colors = map[colorize.Style]string{
	colorize.KeywordStyle: colorizeCode(ansiYellow),
	colorize.ArrowStyle:   colorizeCode(ansiBlue),
	colorize.CommentStyle: colorizeCode(ansiGreen),
	colorize.LineNoStyle:  colorizeCode(ansiBrWhite),
	colorize.NormalStyle:  colorizeCode(ansiBrWhite),
	colorize.NumberStyle:  colorizeCode(ansiBrCyan),
	colorize.StringStyle:  colorizeCode(ansiBrBlue),
}

func TestPrint(t *testing.T) {
	p := gomonkey.ApplyFunc(os.ReadFile, func(name string) ([]byte, error) {
		return []byte(src), nil
	})
	defer p.Reset()
	colorize.Print(os.Stdout, "main.go", bytes.NewBufferString(src), 1, 30, 10, colors)
}
