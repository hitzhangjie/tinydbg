//+build linux,!amd64

package native

import (
	"github.com/hitzhangjie/dlv/pkg/elfwriter"
)

func (p *nativeProcess) DumpProcessNotes(notes []elfwriter.Note, threadDone func()) (threadsDone bool, out []elfwriter.Note, err error) {
	return false, notes, nil
}
