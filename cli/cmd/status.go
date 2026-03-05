package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/saitddundar/gordion-vpn/cli/internal/state"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Gordion VPN connection status",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := state.Read()
		if err != nil {
			return fmt.Errorf("read state: %w", err)
		}

		if s == nil || !s.IsRunning() {
			printStopped()
			if s != nil && !s.IsRunning() {
				// Stale state file — clean it up
				_ = state.Delete()
			}
			return nil
		}

		printConnected(s)
		return nil
	},
}

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

	printTable(rows)
	fmt.Println()
}

func printTable(rows [][]string) {
	maxKey := 0
	for _, r := range rows {
		if len(r[0]) > maxKey {
			maxKey = len(r[0])
		}
	}

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")). // gray
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
