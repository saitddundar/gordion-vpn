package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/saitddundar/gordion-vpn/cli/internal/state"
)

type StatusOutput struct {
	Connected   bool   `json:"connected"`
	PID         int    `json:"pid,omitempty"`
	VPNAddr     string `json:"vpn_addr,omitempty"`
	UptimeSec   int64  `json:"uptime_sec,omitempty"`
	IsExitNode  bool   `json:"is_exit_node"`
	UseExitNode bool   `json:"use_exit_node"`
	ExitNodeID  string `json:"exit_node_id,omitempty"`
	LogFile     string `json:"log_file,omitempty"`
	ConfigFile  string `json:"config_file,omitempty"`
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Gordion VPN connection status",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := state.Read()
		if err != nil {
			return fmt.Errorf("read state: %w", err)
		}

		running := s != nil && s.IsRunning()

		// Stale state cleanup
		if s != nil && !running {
			_ = state.Delete()
			s = nil
		}

		if outputJSON {
			return printStatusJSON(s, running)
		}

		if !running {
			printStopped()
		} else {
			printConnected(s)
		}
		return nil
	},
}

// ─── JSON output ─────────────────────────────────────────────────────────────

func printStatusJSON(s *state.State, running bool) error {
	out := StatusOutput{Connected: running}
	if running && s != nil {
		out.PID = s.PID
		out.VPNAddr = s.VPNAddr
		out.UptimeSec = int64(time.Since(s.StartedAt).Seconds())
		out.IsExitNode = s.IsExitNode
		out.UseExitNode = s.UseExitNode
		out.ExitNodeID = s.ExitNodeID
		out.LogFile = s.LogFile
		out.ConfigFile = s.ConfigFile
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// ─── Human output ─────────────────────────────────────────────────────────────

func printStopped() {
	dot := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Render("●")

	fmt.Printf("\n  %s  %s\n\n",
		dot,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6B7280")).Render("Disconnected"),
	)
	fmt.Printf("  %s\n\n",
		styleDim.Render("Run `gordion up` to connect."),
	)
}

func printConnected(s *state.State) {
	dot := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
		Render("●")

	uptime := time.Since(s.StartedAt).Round(time.Second)

	fmt.Printf("\n  %s  %s\n",
		dot,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10B981")).Render("Connected"),
	)

	fmt.Println()

	rows := [][]string{
		{"VPN Address", orDash(s.VPNAddr)},
		{"Uptime", formatDuration(uptime)},
		{"PID", fmt.Sprintf("%d", s.PID)},
	}

	if s.IsExitNode {
		rows = append(rows, []string{"Role", styleSuccess.Render("Exit Node") + styleDim.Render(" (routing internet traffic)")})
	} else if s.UseExitNode {
		target := s.ExitNodeID
		if target == "" {
			target = "auto"
		}
		rows = append(rows, []string{"Exit Node", styleSuccess.Render(target)})
	} else {
		rows = append(rows, []string{"Mode", styleDim.Render("Mesh VPN (no exit node)")})
	}

	if s.ConfigFile != "" {
		rows = append(rows, []string{"Config", styleDim.Render(s.ConfigFile)})
	}
	if s.LogFile != "" {
		rows = append(rows, []string{"Logs", styleDim.Render(s.LogFile)})
	}

	printTable(rows)
	fmt.Println()
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

func printTable(rows [][]string) {
	maxKey := 0
	for _, r := range rows {
		if len(r[0]) > maxKey {
			maxKey = len(r[0])
		}
	}

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Width(maxKey)

	for _, r := range rows {
		fmt.Printf("  %s  %s\n", keyStyle.Render(r[0]), r[1])
	}
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return styleDim.Render("—")
	}
	return styleBold.Render(s)
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
