package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var cfgFile string
var outputJSON bool // set by --output json

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
}

func Execute() {
	setupHelp()
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, styleError.Render("✗")+" "+err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(
		&cfgFile,
		"config", "c", "",
		"config file (default: configs/agent.dev.yaml)",
	)
	rootCmd.PersistentFlags().BoolVarP(
		&outputJSON,
		"json", "j", false,
		"output as JSON (for scripting / dashboards)",
	)
}

func printOK(msg string) {
	fmt.Println(styleSuccess.Render("✓") + " " + msg)
}

func printErrorExit(msg string) {
	fmt.Fprintln(os.Stderr, styleError.Render("✗")+" "+msg)
	os.Exit(1)
}

func printWarn(msg string) {
	fmt.Println(styleWarn.Render("⚠") + " " + msg)
}
