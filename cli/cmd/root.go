package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// Global flag: config file path
var cfgFile string

// Styles — defined once, used across all commands
var (
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")) // purple

	styleSuccess = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#10B981")) // green

	styleError = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#EF4444")) // red

	styleWarn = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")) // amber

	styleDim = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")) // gray

	styleBold = lipgloss.NewStyle().Bold(true)
)

var rootCmd = &cobra.Command{
	Use:   "gordion",
	Short: "Gordion VPN — self-hosted P2P mesh VPN",
	Long: styleTitle.Render("Gordion VPN") + "\n" +
		styleDim.Render("A self-hosted, peer-to-peer mesh VPN with exit node support.\n") +
		styleDim.Render("https://github.com/saitddundar/gordion-vpn"),
	// No Run — show help by default
}

// Execute is called from main.go
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, styleError.Render("Error: "+err.Error()))
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config", "",
		"config file (default: configs/agent.dev.yaml)",
	)
}

// Helper: print a success line
func printOK(msg string) {
	fmt.Println(styleSuccess.Render("✓") + " " + msg)
}

// Helper: print an error line and exit
func printErrorExit(msg string) {
	fmt.Fprintln(os.Stderr, styleError.Render("✗")+" "+msg)
	os.Exit(1)
}

// Helper: print a warning line
func printWarn(msg string) {
	fmt.Println(styleWarn.Render("⚠") + " " + msg)
}
