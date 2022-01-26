package colorize

import (
	"fmt"
	"io"
)

type lineWriter struct {
	w         io.Writer
	lineRange [2]int
	arrowLine int

	curStyle Style
	started  bool
	lineno   int

	colorEscapes map[Style]string
}

func (w *lineWriter) style(style Style) {
	if w.colorEscapes == nil {
		return
	}
	esc := w.colorEscapes[style]
	if esc == "" {
		esc = w.colorEscapes[NormalStyle]
	}
	fmt.Fprintf(w.w, "%s", esc)
}

func (w *lineWriter) inrange() bool {
	lno := w.lineno
	if !w.started {
		lno = w.lineno + 1
	}
	return lno >= w.lineRange[0] && lno < w.lineRange[1]
}

func (w *lineWriter) nl() {
	w.lineno++
	if !w.inrange() || !w.started {
		return
	}
	w.style(ArrowStyle)
	if w.lineno == w.arrowLine {
		fmt.Fprintf(w.w, "=>")
	} else {
		fmt.Fprintf(w.w, "  ")
	}
	w.style(LineNoStyle)
	fmt.Fprintf(w.w, "%4d:\t", w.lineno)
	w.style(w.curStyle)
}

func (w *lineWriter) writeInternal(style Style, data []byte) {
	if !w.inrange() {
		return
	}

	if !w.started {
		w.started = true
		w.curStyle = style
		w.nl()
	} else if w.curStyle != style {
		w.curStyle = style
		w.style(w.curStyle)
	}

	w.w.Write(data)
}

func (w *lineWriter) Write(style Style, data []byte, last bool) {
	cur := 0
	for i := range data {
		if data[i] == '\n' {
			if last && i == len(data)-1 {
				w.writeInternal(style, data[cur:i])
				if w.curStyle != NormalStyle {
					w.style(NormalStyle)
				}
				if w.inrange() {
					w.w.Write([]byte{'\n'})
				}
				last = false
			} else {
				w.writeInternal(style, data[cur:i+1])
				w.nl()
			}
			cur = i + 1
		}
	}
	if cur < len(data) {
		w.writeInternal(style, data[cur:])
	}
	if last {
		if w.curStyle != NormalStyle {
			w.style(NormalStyle)
		}
		if w.inrange() {
			w.w.Write([]byte{'\n'})
		}
	}
}
