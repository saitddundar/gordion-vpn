package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// service config
type Config struct {
	GRPCPort int `yaml:"grpc_port"`

	DatabaseURL string `yaml:"database_url"`

	LogLevel string `yaml:"log_level"`

	JWTSecret     string `yaml:"jwt_secret"`
	TokenDuration int    `yaml:"token_duration_hours"`
}

// Load loads configuration from file
func Load(path string) (*Config, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// LoadFromEnv loads config with environment variable overrides
func LoadFromEnv(path string) (*Config, error) {
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	// Override with env vars if present
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.DatabaseURL = dbURL
	}

	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.JWTSecret = jwtSecret
	}

	return cfg, nil
}

// Validate validates configuration
func (c *Config) Validate() error {
	if c.GRPCPort == 0 {
		return fmt.Errorf("grpc_port is required")
	}

	if c.DatabaseURL == "" {
		return fmt.Errorf("database_url is required")
	}

	if c.JWTSecret == "" {
		return fmt.Errorf("jwt_secret is required")
	}

	if c.TokenDuration == 0 {
		c.TokenDuration = 24 // default 24 hours
	}

	return nil
}
