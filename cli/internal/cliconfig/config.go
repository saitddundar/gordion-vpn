package cliconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	IdentityAddr  string `yaml:"identity_addr"`
	DiscoveryAddr string `yaml:"discovery_addr"`
	ConfigAddr    string `yaml:"config_addr"`
	TLSCACert     string `yaml:"tls_ca_cert"`
	LogLevel      string `yaml:"log_level"`
	DryRun        *bool  `yaml:"dry_run"`
	WireGuardPort int    `yaml:"wireguard_port"`
	P2PPort       int    `yaml:"p2p_port"`
	IsExitNode    bool   `yaml:"is_exit_node"`
	UseExitNode   bool   `yaml:"use_exit_node"`
	ExitNodeID    string `yaml:"exit_node_id"`
	ExitNodeDNS   string `yaml:"exit_node_dns"`
}

func DefaultPaths() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"configs/agent.dev.yaml",
		"../configs/agent.dev.yaml",
		filepath.Join(home, ".gordion", "config.yaml"),
		"/etc/gordion/config.yaml",
	}
}

func Load(cfgFile string) (*Config, error) {
	candidates := []string{cfgFile}
	if cfgFile == "" {
		candidates = DefaultPaths()
	}

	for _, path := range candidates {
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read config %s: %w", path, err)
		}

		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parse config %s: %w", path, err)
		}

		cfg.applyDefaults()
		return &cfg, nil
	}

	return nil, fmt.Errorf("no config file found (tried: %v)\nRun with --config path/to/agent.yaml", candidates)
}

func (c *Config) applyDefaults() {
	if c.IdentityAddr == "" {
		c.IdentityAddr = "localhost:8001"
	}
	if c.DiscoveryAddr == "" {
		c.DiscoveryAddr = "localhost:8002"
	}
	if c.ConfigAddr == "" {
		c.ConfigAddr = "localhost:8003"
	}
	if c.WireGuardPort == 0 {
		c.WireGuardPort = 51820
	}
	if c.P2PPort == 0 {
		c.P2PPort = 4001
	}
	if c.ExitNodeDNS == "" {
		c.ExitNodeDNS = "1.1.1.1, 1.0.0.1"
	}
}
