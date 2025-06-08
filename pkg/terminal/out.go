package terminal

import (
	"bufio"
	"fmt"
	"io"
)

// Print prints to out the text read from reader, between lines startLine and endLine.
func Print(out io.Writer, reader io.Reader, startLine, endLine, arrowLine int) error {
	scanner := bufio.NewScanner(reader)
	lineno := 0

	for scanner.Scan() {
		lineno++
		if lineno < startLine {
			continue
		}
		if lineno >= endLine {
			break
		}

		// Print line number and arrow
		if lineno == arrowLine {
			fmt.Fprintf(out, "=>")
		} else {
			fmt.Fprintf(out, "  ")
		}
		fmt.Fprintf(out, "%4d:\t%s\n", lineno, scanner.Text())
	}

	return scanner.Err()
}
