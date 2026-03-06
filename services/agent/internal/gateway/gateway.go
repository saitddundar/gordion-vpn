package gateway

import (
	"fmt"
	"runtime"

	pkglogger "github.com/saitddundar/gordion-vpn/pkg/logger"
)

// handles OS-level routing for exit node functionality.
type Manager struct {
	logger   pkglogger.Logger
	iface    string
	outIface string // detected at runtime (linux/darwin)
	enabled  bool
}

// creates a new gateway Manager.
func New(logger pkglogger.Logger, wgIface string) *Manager {
	if wgIface == "" {
		wgIface = defaultIface()
	}
	return &Manager{
		logger: logger,
		iface:  wgIface,
	}
}

// turns on IP forwarding and NAT so this peer can route traffic.
func (m *Manager) Enable() error {
	if m.enabled {
		return nil
	}

	m.logger.Infof("Gateway: enabling exit node on interface %s", m.iface)

	if err := m.enableForwarding(); err != nil {
		return fmt.Errorf("gateway enable failed: %w", err)
	}

	m.enabled = true
	m.logger.Info("Gateway: exit node active — peers can route internet traffic through this node")
	return nil
}

// removes NAT and IP forwarding rules set by Enable.
func (m *Manager) Disable() error {
	if !m.enabled {
		return nil
	}

	m.logger.Info("Gateway: disabling exit node")

	if err := m.disableForwarding(); err != nil {
		return fmt.Errorf("gateway disable failed: %w", err)
	}

	m.enabled = false
	m.logger.Info("Gateway: exit node disabled")
	return nil
}

func defaultIface() string {
	if runtime.GOOS == "windows" {
		return "gordion"
	}
	return "wg0"
}
