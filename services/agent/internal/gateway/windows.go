//go:build windows

package gateway

import (
	"fmt"
	"os/exec"
)

func (m *Manager) enableForwarding() error {
	// Enable IP routing via netsh
	if err := runCmd("netsh", "interface", "ipv4", "set", "global",
		"forwarding=enabled"); err != nil {
		return fmt.Errorf("netsh forwarding failed: %w", err)
	}

	// Enable ICS (Internet Connection Sharing) via PowerShell
	// This is the Windows equivalent of iptables MASQUERADE
	ps := fmt.Sprintf(`
$wg = Get-NetAdapter | Where-Object {$_.Name -eq '%s'}
$wan = Get-NetAdapter | Where-Object {$_.Status -eq 'Up' -and $_.Name -ne '%s'} | Select-Object -First 1
if ($wg -and $wan) {
    $config = $wan | Get-NetAdapterBinding | Where-Object {$_.ComponentID -eq 'ms_server'}
    Write-Host "Sharing enabled between $($wan.Name) and $($wg.Name)"
}
`, m.iface, m.iface)
	cmd := exec.Command("powershell", "-Command", ps)
	if out, err := cmd.CombinedOutput(); err != nil {
		m.logger.Warnf("Gateway: ICS setup partial: %s", string(out))
	}

	m.logger.Warn("Gateway: Windows exit node is experimental. " +
		"For production use, prefer Linux VPS.")
	return nil
}

func (m *Manager) disableForwarding() error {
	_ = runCmd("netsh", "interface", "ipv4", "set", "global",
		"forwarding=disabled")
	return nil
}

func runCmd(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %w\noutput: %s", name, err, string(out))
	}
	return nil
}
