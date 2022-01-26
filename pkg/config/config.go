package config

const (
	configDir  string = "dlv"
	configFile string = "config.yml"
)

// SubstitutePathRule describes a rule for substitution of path to source code file.
type SubstitutePathRule struct {
	// Directory path will be substituted if it matches `From`.
	From string
	// Path to which substitution is performed.
	To string
}

// SubstitutePathRules is a slice of source code path substitution rules.
type SubstitutePathRules []SubstitutePathRule

// Config defines all configuration options available to be set through the config file.
type Config struct {
	// Commands aliases.
	Aliases map[string][]string `yaml:"aliases"`
	// Source code path substitution rules.
	SubstitutePath SubstitutePathRules `yaml:"substitute-path"`

	// MaxStringLen is the maximum string length that the commands print,
	// locals, args and vars should read (in verbose mode).
	MaxStringLen *int `yaml:"max-string-len,omitempty"`
	// MaxArrayValues is the maximum number of array items that the commands
	// print, locals, args and vars should read (in verbose mode).
	MaxArrayValues *int `yaml:"max-array-values,omitempty"`
	// MaxVariableRecurse is output evaluation depth of nested struct members, array and
	// slice items and dereference pointers
	MaxVariableRecurse *int `yaml:"max-variable-recurse,omitempty"`
	// DisassembleFlavor allow user to specify output syntax flavor of assembly, one of
	// this list "intel"(default), "gnu", "go"
	DisassembleFlavor *string `yaml:"disassemble-flavor,omitempty"`

	// If ShowLocationExpr is true whatis will print the DWARF location
	// expression for its argument.
	ShowLocationExpr bool `yaml:"show-location-expr"`

	// Source list line-number color, as a terminal escape sequence.
	// For historic reasons, this can also be an integer color code.
	SourceListLineColor interface{} `yaml:"source-list-line-color"`

	// Source list arrow color, as a terminal escape sequence.
	SourceListArrowColor string `yaml:"source-list-arrow-color"`

	// Source list keyword color, as a terminal escape sequence.
	SourceListKeywordColor string `yaml:"source-list-keyword-color"`

	// Source list string color, as a terminal escape sequence.
	SourceListStringColor string `yaml:"source-list-string-color"`

	// Source list number color, as a terminal escape sequence.
	SourceListNumberColor string `yaml:"source-list-number-color"`

	// Source list comment color, as a terminal escape sequence.
	SourceListCommentColor string `yaml:"source-list-comment-color"`

	// number of lines to list above and below cursor when printfile() is
	// called (i.e. when execution stops, listCommand is used, etc)
	SourceListLineCount *int `yaml:"source-list-line-count,omitempty"`
}

func (c *Config) GetSourceListLineCount() int {
	n := 5 // default value
	lcp := c.SourceListLineCount
	if lcp != nil && *lcp >= 0 {
		n = *lcp
	}
	return n
}
