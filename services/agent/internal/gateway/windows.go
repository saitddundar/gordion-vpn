//go:build windows

package gateway

import (
	"fmt"
	"os/exec"
	"strings"
)

const natName = "GordionNAT"

func defaultOutboundIface() (string, error) {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		`(Get-NetRoute -DestinationPrefix '0.0.0.0/0' | Sort-Object RouteMetric | Select-Object -First 1).InterfaceAlias`,
	).Output()
	if err != nil {
		return "", fmt.Errorf("Get-NetRoute: %w", err)
	}
	iface := strings.TrimSpace(string(out))
	if iface == "" {
		return "", fmt.Errorf("no default route found")
	}
	return iface, nil
}

func (m *Manager) enableForwarding() error {
	runCmd("powershell", "-NoProfile", "-Command", //nolint:errcheck
		fmt.Sprintf(`Set-NetIPInterface -InterfaceAlias '%s' -Forwarding Enabled -ErrorAction SilentlyContinue`, m.iface))

	natPS := fmt.Sprintf(`
		if (-not (Get-NetNat -Name '%s' -ErrorAction SilentlyContinue)) {
			New-NetNat -Name '%s' -InternalIPInterfaceAddressPrefix '10.8.0.0/16'
		}`, natName, natName)

	if err := runCmd("powershell", "-NoProfile", "-Command", natPS); err != nil {
		return fmt.Errorf("New-NetNat (run as Administrator): %w", err)
	}

	m.logger.Warn("Gateway: Windows exit node — prefer Linux VPS for production")
	return nil
}

func (m *Manager) disableForwarding() error {
	_ = runCmd("powershell", "-NoProfile", "-Command",
		fmt.Sprintf(`Remove-NetNat -Name '%s' -Confirm:$false -ErrorAction SilentlyContinue`, natName))
	_ = runCmd("powershell", "-NoProfile", "-Command",
		fmt.Sprintf(`Set-NetIPInterface -InterfaceAlias '%s' -Forwarding Disabled -ErrorAction SilentlyContinue`, m.iface))
	return nil
}

func runCmd(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w\n%s", name, err, out)
	}
	return nil
}
