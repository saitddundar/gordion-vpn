package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel    string `yaml:"log_level"`
	Environment string `yaml:"environment"`
}

func Load(path string, cfg interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	return nil
}

func LoadWithDefaults(path string, cfg interface{}) error {
	if err := Load(path, cfg); err != nil && !os.IsNotExist(err) {
		return err
	}
	if c, ok := cfg.(*Config); ok {
		if v := os.Getenv("LOG_LEVEL"); v != "" {
			c.LogLevel = v
		}
		if v := os.Getenv("ENVIRONMENT"); v != "" {
			c.Environment = v
		}
	}
	return nil
}
