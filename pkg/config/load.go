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
func LoadConfig() (cfg *Config, err error) {
	loadConfigOnce.Do(func() {
		cfg, err = loadConfig()
	})
	return cfg, err
}

func loadConfig() (*Config, error) {
	if err := createConfigPath(); err != nil {
		return nil, fmt.Errorf("could not create config directory: %v", err)
	}

	fp, err := GetConfigFilePath(configFile)
	if err != nil {
		return nil, fmt.Errorf("unable to get config file path: %v", err)
	}

	f, err := os.Open(fp)
	if err != nil {
		f, err = createDefaultConfig(fp)
		if err != nil {
			return nil, fmt.Errorf("error creating default config file: %v", err)
		}
	}
	defer f.Close()

	dat, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("unable to read config data: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(dat, &cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config file: %v", err)
	}
	return &cfg, nil
}
