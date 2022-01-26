package config

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

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
