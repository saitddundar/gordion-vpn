//go:build darwin

package gateway

import (
	"fmt"
	"os/exec"
)

func (m *Manager) enableForwarding() error {
	if err := runCmd("sysctl", "-w", "net.inet.ip.forwarding=1"); err != nil {
		return fmt.Errorf("sysctl forwarding failed: %w", err)
	}
	pfRule := fmt.Sprintf("nat on en0 from %s:network to any -> (en0)\n", m.iface)
	if err := runCmd("sh", "-c",
		fmt.Sprintf("echo '%s' | pfctl -f - && pfctl -e", pfRule)); err != nil {
		return fmt.Errorf("pfctl NAT failed: %w", err)
	}
	return nil
}

func (m *Manager) disableForwarding() error {
	_ = runCmd("pfctl", "-d")
	_ = runCmd("sysctl", "-w", "net.inet.ip.forwarding=0")
	return nil
}

func runCmd(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %w\noutput: %s", name, err, string(out))
	}
	return nil
}
