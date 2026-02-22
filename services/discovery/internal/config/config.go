package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GRPCPort     int    `yaml:"grpc_port"`
	RedisURL     string `yaml:"redis_url"`
	LogLevel     string `yaml:"log_level"`
	HeartbeatTTL int    `yaml:"heartbeat_ttl"` //second type
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

	// Override with env vars if present
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		cfg.RedisURL = redisURL
	}

	return cfg, nil
}

func (c *Config) setDefaults() {
	if c.GRPCPort == 0 {
		c.GRPCPort = 8002
	}
	if c.RedisURL == "" {
		c.RedisURL = "localhost:6379"
	}
	if c.LogLevel == "" {
		c.LogLevel = "debug"
	}
	if c.HeartbeatTTL == 0 {
		c.HeartbeatTTL = 30
	}
}
