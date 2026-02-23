package wireguard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	pkglogger "github.com/saitddundar/gordion-vpn/pkg/logger"
)

const configTemplate = `[Interface]
PrivateKey = {{ .PrivateKey }}
Address = {{ .Address }}
MTU = {{ .MTU }}
{{ if .DNS }}DNS = {{ .DNS }}{{ end }}

{{ range .Peers }}[Peer]
PublicKey = {{ .PublicKey }}
Endpoint = {{ .Endpoint }}
AllowedIPs = {{ .AllowedIPs }}
PersistentKeepalive = 25

{{ end }}`

type PeerConfig struct {
	PublicKey  string
	Endpoint   string
	AllowedIPs string
}

type Config struct {
	PrivateKey string
	Address    string
	MTU        int32
	DNS        string
	Peers      []PeerConfig
}

type Manager struct {
	logger    pkglogger.Logger
	iface     string
	configDir string
	dryRun    bool
}

func NewManager(logger pkglogger.Logger, dryRun bool) *Manager {
	iface := "wg0"
	if runtime.GOOS == "windows" {
		iface = "gordion"
	}

	configDir := os.TempDir()

	return &Manager{
		logger:    logger,
		iface:     iface,
		configDir: configDir,
		dryRun:    dryRun,
	}
}

func (m *Manager) Configure(cfg *Config) error {
	configPath := filepath.Join(m.configDir, m.iface+".conf")

	tmpl, err := template.New("wg").Parse(configTemplate)
	if err != nil {
		return fmt.Errorf("template parse error: %w", err)
	}

	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("config file create error: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, cfg); err != nil {
		return fmt.Errorf("template execute error: %w", err)
	}

	m.logger.Infof("WireGuard config written to %s", configPath)

	if m.dryRun {
		m.logger.Warn("Dry run mode - skipping tunnel setup")
		return nil
	}

	return m.up(configPath)
}

func (m *Manager) AddPeer(peer PeerConfig) error {
	if m.dryRun {
		m.logger.Infof("[dry-run] Add peer: %s @ %s", peer.PublicKey[:8], peer.Endpoint)
		return nil
	}

	args := []string{
		"set", m.iface,
		"peer", peer.PublicKey,
		"endpoint", peer.Endpoint,
		"allowed-ips", peer.AllowedIPs,
		"persistent-keepalive", "25",
	}

	return m.runWG(args...)
}

func (m *Manager) RemovePeer(publicKey string) error {
	if m.dryRun {
		m.logger.Infof("[dry-run] Remove peer: %s", publicKey[:8])
		return nil
	}

	return m.runWG("set", m.iface, "peer", publicKey, "remove")
}

func (m *Manager) Down() error {
	if m.dryRun {
		m.logger.Info("[dry-run] Tunnel down")
		return nil
	}

	configPath := filepath.Join(m.configDir, m.iface+".conf")

	switch runtime.GOOS {
	case "linux", "darwin":
		return m.runCmd("wg-quick", "down", configPath)
	case "windows":
		return m.runCmd("wireguard", "/uninstalltunnelservice", m.iface)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func (m *Manager) up(configPath string) error {
	switch runtime.GOOS {
	case "linux", "darwin":
		return m.runCmd("wg-quick", "up", configPath)
	case "windows":
		return m.runCmd("wireguard", "/installtunnelservice", configPath)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func (m *Manager) runWG(args ...string) error {
	return m.runCmd("wg", args...)
}

func (m *Manager) runCmd(name string, args ...string) error {
	m.logger.Debugf("exec: %s %s", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %w\noutput: %s", name, err, string(output))
	}
	return nil
}
