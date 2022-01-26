package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v2"
)

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

var loadConfigOnce sync.Once

// LoadConfig attempts to populate a Config object from the config.yml file.
func LoadConfig() (*Config, error) {
	var cfg *Config
	var err error
	loadConfigOnce.Do(func() {
		cfg, err = loadConfig()
	})
	return cfg, err
}

func loadConfig() (*Config, error) {
	if err := createConfigPath(); err != nil {
		return &Config{}, fmt.Errorf("could not create config directory: %v", err)
	}
	fullConfigFile, err := GetConfigFilePath(configFile)
	if err != nil {
		return &Config{}, fmt.Errorf("unable to get config file path: %v", err)
	}

	f, err := os.Open(fullConfigFile)
	if err != nil {
		f, err = createDefaultConfig(fullConfigFile)
		if err != nil {
			return &Config{}, fmt.Errorf("error creating default config file: %v", err)
		}
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return &Config{}, fmt.Errorf("unable to read config data: %v", err)
	}

	var c Config
	if err = yaml.Unmarshal(data, &c); err != nil {
		return &Config{}, fmt.Errorf("unable to decode config file: %v", err)
	}
	return &c, nil
}

// SaveConfig will marshal and save the config struct
// to disk.
func SaveConfig(conf *Config) error {
	fullConfigFile, err := GetConfigFilePath(configFile)
	if err != nil {
		return err
	}

	out, err := yaml.Marshal(*conf)
	if err != nil {
		return err
	}

	f, err := os.Create(fullConfigFile)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(out)
	return err
}

func createDefaultConfig(path string) (*os.File, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("unable to create config file: %v", err)
	}

	if err = writeDefaultConfig(f); err != nil {
		return nil, fmt.Errorf("unable to write default configuration: %v", err)
	}

	f.Seek(0, io.SeekStart)
	return f, nil
}

func writeDefaultConfig(f *os.File) error {
	_, err := f.WriteString(
		`# Configuration file for the delve debugger.

# This is the default configuration file. Available options are provided, but disabled.
# Delete the leading hash mark to enable an item.

# Uncomment the following line and set your preferred ANSI color for source
# line numbers in the (list) command. The default is 34 (dark blue). See
# https://en.wikipedia.org/wiki/ANSI_escape_code#3/4_bit
# source-list-line-color: "\x1b[34m"

# Uncomment the following lines to change the colors used by syntax highlighting.
# source-list-keyword-color: "\x1b[0m"
# source-list-string-color: "\x1b[92m"
# source-list-number-color: "\x1b[0m"
# source-list-comment-color: "\x1b[95m"
# source-list-arrow-color: "\x1b[93m"

# Uncomment to change the number of lines printed above and below cursor when
# listing source code.
# source-list-line-count: 5

# Provided aliases will be added to the default aliases for a given command.
aliases:
  # command: ["alias1", "alias2"]

# Define sources path substitution rules. Can be used to rewrite a source path stored
# in program's debug information, if the sources were moved to a different place
# between compilation and debugging.
# Note that substitution rules will not be used for paths passed to "break" and "trace"
# commands.
substitute-path:
  # - {from: path, to: path}
  
# Maximum number of elements loaded from an array.
# max-array-values: 64

# Maximum loaded string length.
# max-string-len: 64

# Output evaluation.
# max-variable-recurse: 1

# Uncomment the following line to make the whatis command also print the DWARF location expression of its argument.
# show-location-expr: true

# Allow user to specify output syntax flavor of assembly, one of this list "intel"(default), "gnu", "go".
# disassemble-flavor: intel

# List of directories to use when searching for separate debug info files.
debug-info-directories: ["/usr/lib/debug/.build-id"]
`)
	return err
}

// createConfigPath creates the directory structure at which all config files are saved.
func createConfigPath() error {
	path, err := GetConfigFilePath("")
	if err != nil {
		return err
	}
	return os.MkdirAll(path, 0700)
}

// GetConfigFilePath gets the full path to the given config file name.
func GetConfigFilePath(file string) (string, error) {
	home := getUserHomeDir()
	fp := filepath.Join(home, ".config", configDir, file)
	return fp, nil
}

func getUserHomeDir() string {
	userHomeDir := "."
	usr, err := user.Current()
	if err == nil {
		userHomeDir = usr.HomeDir
	}
	return userHomeDir
}
