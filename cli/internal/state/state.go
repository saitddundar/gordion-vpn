package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type State struct {
	PID         int       `json:"pid"`
	VPNAddr     string    `json:"vpn_addr"` // e.g. "10.8.0.2/24"
	StartedAt   time.Time `json:"started_at"`
	ConfigFile  string    `json:"config_file"`
	IsExitNode  bool      `json:"is_exit_node"`
	UseExitNode bool      `json:"use_exit_node"`
	ExitNodeID  string    `json:"exit_node_id"`
}

func Path() string {
	// On Linux: /run/user/<uid>/gordion/state.json
	// On others: os.TempDir()/gordion/state.json
	dir := filepath.Join(os.TempDir(), "gordion")
	return filepath.Join(dir, "state.json")
}

func Write(s *State) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func Read() (*State, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse state file: %w", err)
	}
	return &s, nil
}

func Delete() error {
	err := os.Remove(Path())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *State) IsRunning() bool {
	if s == nil || s.PID == 0 {
		return false
	}
	proc, err := os.FindProcess(s.PID)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds; sending signal 0 checks existence.
	return processExists(proc)
}
