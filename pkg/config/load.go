package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"gopkg.in/yaml.v2"
)

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
