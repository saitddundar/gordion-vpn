package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	IdentityAddr  string `yaml:"identity_addr"`
	DiscoveryAddr string `yaml:"discovery_addr"`
	ConfigAddr    string `yaml:"config_addr"`
	LogLevel      string `yaml:"log_level"`
	Heartbeat     int    `yaml:"heartbeat_interval"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config read error: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config parse error: %w", err)
	}

	cfg.setDefaults()
	cfg.overrideFromEnv()

	return cfg, nil
}

func (c *Config) setDefaults() {
	if c.IdentityAddr == "" {
		c.IdentityAddr = "localhost:8001"
	}
	if c.DiscoveryAddr == "" {
		c.DiscoveryAddr = "localhost:8002"
	}
	if c.ConfigAddr == "" {
		c.ConfigAddr = "localhost:8003"
	}
	if c.LogLevel == "" {
		c.LogLevel = "debug"
	}
	if c.Heartbeat == 0 {
		c.Heartbeat = 25
	}
}

func (c *Config) overrideFromEnv() {
	if v := os.Getenv("IDENTITY_ADDR"); v != "" {
		c.IdentityAddr = v
	}
	if v := os.Getenv("DISCOVERY_ADDR"); v != "" {
		c.DiscoveryAddr = v
	}
	if v := os.Getenv("CONFIG_ADDR"); v != "" {
		c.ConfigAddr = v
	}
}
