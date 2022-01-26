// Package locspec implements code to parse a string into a specific
// location specification.
//
// Location spec examples:
//
//  locStr ::= <filename>:<line> | <function>[:<line>] | /<regex>/ | (+|-)<offset> | <line> | *<address>
//  * <filename> can be the full path of a file or just a suffix
//  * <function> ::= <package>.<receiver type>.<name> | <package>.(*<receiver type>).<name> | <receiver type>.<name> | <package>.<name> | (*<receiver type>).<name> | <name>
//    <function> must be unambiguous
//  * /<regex>/ will return a location for each function matched by regex
//  * +<offset> returns a location for the line that is <offset> lines after the current line
//  * -<offset> returns a location for the line that is <offset> lines before the current line
//  * <line> returns a location for a line in the current file
//  * *<address> returns the location corresponding to the specified address
package locspec

// NormalLocationSpec represents a basic location spec.
// This can be a `file:line` or `func:line`.
type NormalLocationSpec struct {
	Base       string
	FuncBase   *FuncLocationSpec
	LineOffset int
}

// RegexLocationSpec represents a regular expression
// location expression such as `/^myfunc$/`.
type RegexLocationSpec struct {
	FuncRegex string
}

// AddrLocationSpec represents an address when used
// as a location spec.
type AddrLocationSpec struct {
	AddrExpr string
}

// OffsetLocationSpec represents a location spec that
// is an offset of the current location (file:line).
type OffsetLocationSpec struct {
	Offset int
}

// LineLocationSpec represents a line number in the current file.
// This can be a `line`.
type LineLocationSpec struct {
	Line int
}

// FuncLocationSpec represents a function in the target program.
type FuncLocationSpec struct {
	PackageName           string
	AbsolutePackage       bool
	ReceiverName          string
	PackageOrReceiverName string
	BaseName              string
}
