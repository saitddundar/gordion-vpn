package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type cliState struct {
	PID         int       `json:"pid"`
	VPNAddr     string    `json:"vpn_addr"`
	Token       string    `json:"token"`
	LogFile     string    `json:"log_file"`
	StartedAt   time.Time `json:"started_at"`
	ConfigFile  string    `json:"config_file"`
	IsExitNode  bool      `json:"is_exit_node"`
	UseExitNode bool      `json:"use_exit_node"`
	ExitNodeID  string    `json:"exit_node_id"`
}

func cliStatePath() string {
	return filepath.Join(os.TempDir(), "gordion", "state.json")
}

// It merges with any existing state so the CLI's PID/StartedAt etc. are preserved.
func (a *Agent) updateCLIState(vpnAddr, token string) {
	path := cliStatePath()

	// Read existing state (may have PID, LogFile etc. from gordion up)
	var s cliState
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &s)
	}

	s.VPNAddr = vpnAddr
	s.Token = token
	s.IsExitNode = a.cfg.IsExitNode
	s.UseExitNode = a.cfg.UseExitNode
	s.ExitNodeID = a.cfg.ExitNodeID

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		a.logger.Warnf("updateCLIState: marshal: %v", err)
		return
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		a.logger.Warnf("updateCLIState: write %s: %v", path, err)
	}
}
