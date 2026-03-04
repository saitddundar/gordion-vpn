//go:build linux

package gateway

import (
	"fmt"
	"os/exec"
)

func (m *Manager) enableForwarding() error {
	// Enable kernel IP forwarding
	if err := runCmd("sysctl", "-w", "net.ipv4.ip_forward=1"); err != nil {
		return fmt.Errorf("ip_forward failed: %w", err)
	}

	// Add NAT masquerade rule: packets from WG interface are masqueraded
	// as coming from this machine's outgoing IP
	if err := runCmd("iptables", "-t", "nat", "-A", "POSTROUTING",
		"-o", "eth0", "-j", "MASQUERADE"); err != nil {
		return fmt.Errorf("iptables MASQUERADE failed: %w", err)
	}

	// Allow forwarding from WG interface
	if err := runCmd("iptables", "-A", "FORWARD",
		"-i", m.iface, "-j", "ACCEPT"); err != nil {
		return fmt.Errorf("iptables FORWARD in failed: %w", err)
	}

	// Allow return traffic
	if err := runCmd("iptables", "-A", "FORWARD",
		"-o", m.iface, "-m", "state", "--state", "RELATED,ESTABLISHED",
		"-j", "ACCEPT"); err != nil {
		return fmt.Errorf("iptables FORWARD out failed: %w", err)
	}

	return nil
}

func (m *Manager) disableForwarding() error {
	// Remove NAT masquerade rule
	_ = runCmd("iptables", "-t", "nat", "-D", "POSTROUTING",
		"-o", "eth0", "-j", "MASQUERADE")

	// Remove forwarding rules
	_ = runCmd("iptables", "-D", "FORWARD", "-i", m.iface, "-j", "ACCEPT")
	_ = runCmd("iptables", "-D", "FORWARD",
		"-o", m.iface, "-m", "state", "--state", "RELATED,ESTABLISHED",
		"-j", "ACCEPT")

	// Disable IP forwarding
	_ = runCmd("sysctl", "-w", "net.ipv4.ip_forward=0")

	return nil
}

func runCmd(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v failed: %w\noutput: %s", name, args, err, string(out))
	}
	return nil
}
