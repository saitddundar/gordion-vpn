package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const helpWidth = 72

func setupHelp() {
	// Hide noisy built-in commands before rendering
	for _, c := range rootCmd.Commands() {
		if c.Name() == "completion" || c.Name() == "help" {
			c.Hidden = true
		}
	}

	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		renderHelp(cmd)
	})
	for _, sub := range rootCmd.Commands() {
		sub := sub
		sub.SetHelpFunc(func(cmd *cobra.Command, args []string) {
			renderHelp(cmd)
		})
	}
}

// ─── Main renderer ────────────────────────────────────────────────────────────

func renderHelp(cmd *cobra.Command) {
	fmt.Println()

	// ── Title + short description ──────────────────────────────────────
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED")).Render("Gordion VPN")
	ver := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("(" + Version + ")")
	fmt.Printf("  %s %s\n", title, ver)
	fmt.Printf("  %s\n\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Render(cmd.Short),
	)

	// ── Usage ──────────────────────────────────────────────────────────
	usageArg := cmd.UseLine()
	if cmd.HasAvailableSubCommands() {
		usageArg = cmd.CommandPath() + " [command]"
	}
	fmt.Printf("  %s %s\n\n",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A78BFA")).Render("Usage:"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Render(usageArg),
	)

	// ── Commands section ───────────────────────────────────────────────
	if cmd.HasAvailableSubCommands() {
		printBox("Commands", func() {
			maxLen := maxCmdLen(cmd.Commands())
			for _, sub := range cmd.Commands() {
				if sub.Hidden || sub.Name() == "completion" || sub.Name() == "help" {
					continue
				}
				name := lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#34D399")).
					Width(maxLen + 2).
					Render(sub.Name())
				desc := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#9CA3AF")).
					Render(sub.Short)
				fmt.Printf("  %s  %s\n", name, desc)
			}
		})
	}

	// ── Options section ────────────────────────────────────────────────
	if cmd.LocalFlags().HasFlags() {
		printBox("Options", func() {
			printFlagLines(cmd.LocalFlags().FlagUsages())
		})
	}

	// ── Global Options ─────────────────────────────────────────────────
	if cmd.InheritedFlags().HasFlags() {
		printBox("Global Options", func() {
			printFlagLines(cmd.InheritedFlags().FlagUsages())
		})
	}

	// ── Footer (outside any box) ─────────────────────────────────
	if cmd.HasAvailableSubCommands() {
		fmt.Printf("  %s\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563")).
				Render(fmt.Sprintf(`Run "%s [command] --help" for more information.`, cmd.CommandPath())),
		)
	}
	fmt.Println()
}

// ─── Box section ──────────────────────────────────────────────────────────────

// ─ Commands ─────────────────────────────────
//
//	up    Start the agent
//	down  Stop the agent
func printBox(title string, content func()) {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")). // amber
		Bold(true)

	lineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#374151")) // dark gray

	label := labelStyle.Render(" " + title + " ")
	// Calculate remaining dashes after "─ Title "
	// ANSI codes don't count as visible chars, so measure plain text
	labelLen := 2 + len(title) + 2 // " Title " with surrounding dashes
	dashes := helpWidth - labelLen
	if dashes < 4 {
		dashes = 4
	}

	rule := lineStyle.Render("─") + label + lineStyle.Render(strings.Repeat("─", dashes))
	fmt.Printf("\n %s\n", rule)
	content()
}

// ─── Flag lines ───────────────────────────────────────────────────────────────

func printFlagLines(usage string) {
	flagStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#60A5FA")) // blue
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))            // gray

	for _, line := range strings.Split(strings.TrimRight(usage, "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Skip built-in -h/--help — it's universally known, just noise
		if strings.Contains(trimmed, "--help") {
			continue
		}
		// cobra formats: "  -f, --flag string   description"
		idx := strings.Index(trimmed, "   ")
		if idx > 0 {
			flag := strings.TrimSpace(trimmed[:idx])
			desc := strings.TrimSpace(trimmed[idx:])
			fmt.Printf("  %s  %s\n", flagStyle.Render(flag), descStyle.Render(desc))
		} else {
			fmt.Printf("  %s\n", descStyle.Render(trimmed))
		}
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func maxCmdLen(cmds []*cobra.Command) int {
	max := 0
	for _, c := range cmds {
		if !c.Hidden && len(c.Name()) > max {
			max = len(c.Name())
		}
	}
	return max
}
