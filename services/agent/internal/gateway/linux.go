//go:build linux

package gateway

import (
	"fmt"
	"os/exec"
	"strings"
)

func defaultOutboundIface() (string, error) {
	out, err := exec.Command("ip", "route", "get", "8.8.8.8").Output()
	if err != nil {
		return "", fmt.Errorf("ip route get: %w", err)
	}
	fields := strings.Fields(string(out))
	for i, f := range fields {
		if f == "dev" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}
	return "", fmt.Errorf("no dev field in: %s", strings.TrimSpace(string(out)))
}

func (m *Manager) enableForwarding() error {
	iface, err := defaultOutboundIface()
	if err != nil {
		return fmt.Errorf("detect outbound iface: %w", err)
	}
	m.logger.Infof("Gateway: outbound interface: %s", iface)

	if err := runCmd("sysctl", "-w", "net.ipv4.ip_forward=1"); err != nil {
		return err
	}
	if err := runCmd("iptables", "-t", "nat", "-A", "POSTROUTING", "-o", iface, "-j", "MASQUERADE"); err != nil {
		return err
	}
	if err := runCmd("iptables", "-A", "FORWARD", "-i", m.iface, "-j", "ACCEPT"); err != nil {
		return err
	}
	if err := runCmd("iptables", "-A", "FORWARD", "-o", m.iface, "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT"); err != nil {
		return err
	}
	m.outIface = iface
	return nil
}

func (m *Manager) disableForwarding() error {
	iface := m.outIface
	if iface == "" {
		iface, _ = defaultOutboundIface()
	}
	_ = runCmd("iptables", "-t", "nat", "-D", "POSTROUTING", "-o", iface, "-j", "MASQUERADE")
	_ = runCmd("iptables", "-D", "FORWARD", "-i", m.iface, "-j", "ACCEPT")
	_ = runCmd("iptables", "-D", "FORWARD", "-o", m.iface, "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT")
	_ = runCmd("sysctl", "-w", "net.ipv4.ip_forward=0")
	return nil
}

func runCmd(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w\n%s", name, err, out)
	}
	return nil
}
