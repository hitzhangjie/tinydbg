package loclist

import "github.com/hitzhangjie/dlv/pkg/dwarf/godwarf"

// Reader represents a loclist reader.
type Reader interface {
	Find(off int, staticBase, base, pc uint64, debugAddr *godwarf.DebugAddr) (*Entry, error)
	Empty() bool
}
