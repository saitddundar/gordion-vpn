package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config base struct - her servis extend edebilir
type Config struct {
	LogLevel    string `yaml:"log_level"`
	Environment string `yaml:"environment"` // dev, prod
}

// Load loads config from YAML file
func Load(path string, cfg interface{}) error {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	return nil
}

// LoadWithDefaults loads config with environment variable override
func LoadWithDefaults(path string, cfg interface{}) error {
	// Try to load from file
	if err := Load(path, cfg); err != nil {
		// If file doesn't exist, use defaults
		if !os.IsNotExist(err) {
			return err
		}
	}

	// TODO: Override with env vars

	return nil
}
