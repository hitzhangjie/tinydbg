package terminal

import (
	"fmt"

	"github.com/hitzhangjie/dlv/pkg/config"
	"github.com/hitzhangjie/dlv/pkg/terminal/colorize"
)

const (
	terminalHighlightEscapeCode string = "\033[%2dm"
	terminalResetEscapeCode     string = "\033[0m"
)

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

func buildColorEscapes(conf *config.Config) map[colorize.Style]string {
	colorEscapes := make(map[colorize.Style]string)
	colorEscapes[colorize.NormalStyle] = terminalResetEscapeCode
	wd := func(s string, defaultCode int) string {
		if s == "" {
			return fmt.Sprintf(terminalHighlightEscapeCode, defaultCode)
		}
		return s
	}
	colorEscapes[colorize.KeywordStyle] = conf.SourceListKeywordColor
	colorEscapes[colorize.StringStyle] = wd(conf.SourceListStringColor, ansiGreen)
	colorEscapes[colorize.NumberStyle] = conf.SourceListNumberColor
	colorEscapes[colorize.CommentStyle] = wd(conf.SourceListCommentColor, ansiBrMagenta)
	colorEscapes[colorize.ArrowStyle] = wd(conf.SourceListArrowColor, ansiYellow)
	switch x := conf.SourceListLineColor.(type) {
	case string:
		colorEscapes[colorize.LineNoStyle] = x
	case int:
		if (x > ansiWhite && x < ansiBrBlack) || x < ansiBlack || x > ansiBrWhite {
			x = ansiBlue
		}
		colorEscapes[colorize.LineNoStyle] = fmt.Sprintf(terminalHighlightEscapeCode, x)
	case nil:
		colorEscapes[colorize.LineNoStyle] = fmt.Sprintf(terminalHighlightEscapeCode, ansiBlue)
	}
	return colorEscapes
}
