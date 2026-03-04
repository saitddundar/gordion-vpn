package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	IdentityAddr     string `yaml:"identity_addr"`
	DiscoveryAddr    string `yaml:"discovery_addr"`
	ConfigAddr       string `yaml:"config_addr"`
	LogLevel         string `yaml:"log_level"`
	Heartbeat        int    `yaml:"heartbeat_interval"`
	PeerSyncInterval int    `yaml:"peer_sync_interval"`
	DryRun           *bool  `yaml:"dry_run"`
	WireGuardPort    int    `yaml:"wireguard_port"`
	P2PPort          int    `yaml:"p2p_port"`
	TLSCACert        string `yaml:"tls_ca_cert"`

	// Exit node configuration
	IsExitNode  bool   `yaml:"is_exit_node"`  // this peer routes internet traffic for others
	UseExitNode bool   `yaml:"use_exit_node"` // route all internet traffic through an exit node
	ExitNodeID  string `yaml:"exit_node_id"`  // specific exit node to use (empty = auto-select)
	ExitNodeDNS string `yaml:"exit_node_dns"` // DNS server to use when exit node is active (prevents DNS leaks)
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
	if c.DryRun == nil {
		t := true
		c.DryRun = &t
	}
	if c.WireGuardPort == 0 {
		c.WireGuardPort = 51820
	}
	if c.P2PPort == 0 {
		c.P2PPort = 4001
	}
	if c.PeerSyncInterval == 0 {
		c.PeerSyncInterval = 60
	}
	if c.ExitNodeDNS == "" {
		c.ExitNodeDNS = "1.1.1.1, 1.0.0.1" // Cloudflare — will route through the tunnel
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
	if v := os.Getenv("P2P_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.P2PPort = port
		}
	}
	if v := os.Getenv("WIREGUARD_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.WireGuardPort = port
		}
	}
	if v := os.Getenv("TLS_CA_CERT"); v != "" {
		c.TLSCACert = v
	}
	if v := os.Getenv("IS_EXIT_NODE"); v == "true" {
		c.IsExitNode = true
	}
	if v := os.Getenv("USE_EXIT_NODE"); v == "true" {
		c.UseExitNode = true
	}
	if v := os.Getenv("EXIT_NODE_ID"); v != "" {
		c.ExitNodeID = v
	}
	if v := os.Getenv("EXIT_NODE_DNS"); v != "" {
		c.ExitNodeDNS = v
	}
}
