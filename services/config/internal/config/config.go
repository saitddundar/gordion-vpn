package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GRPCPort    int      `yaml:"grpc_port"`
	NetworkCIDR string   `yaml:"network_cidr"` // network: 10.8.0.0/16
	MTU         int      `yaml:"mtu"`
	DNSServers  []string `yaml:"dns_servers"`
	RedisURL    string   `yaml:"redis_url"`
	LogLevel    string   `yaml:"log_level"`
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
	return cfg, nil
}

func (c *Config) setDefaults() {
	if c.GRPCPort == 0 {
		c.GRPCPort = 8003
	}
	if c.NetworkCIDR == "" {
		c.NetworkCIDR = "10.8.0.0/16"
	}
	if c.MTU == 0 {
		c.MTU = 1420
	}
	if len(c.DNSServers) == 0 {
		c.DNSServers = []string{"1.1.1.1", "8.8.8.8"}
	}
	if c.RedisURL == "" {
		c.RedisURL = "localhost:6379"
	}
	if c.LogLevel == "" {
		c.LogLevel = "debug"
	}
}
