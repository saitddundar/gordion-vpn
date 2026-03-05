package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/saitddundar/gordion-vpn/cli/internal/state"
)

// DefaultLogPath returns the expected log file location for the gordion agent.
func DefaultLogPath() string {
	return filepath.Join(os.TempDir(), "gordion", "agent.log")
}

var logsFollow bool
var logsLines int

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View Gordion VPN logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		logPath := DefaultLogPath()

		// If agent wrote a custom log path into state, prefer that
		s, _ := state.Read()
		if s != nil && s.LogFile != "" {
			logPath = s.LogFile
		}

		f, err := os.Open(logPath)
		if err != nil {
			if os.IsNotExist(err) {
				printWarn("No log file found at: " + logPath)
				fmt.Printf("  %s\n\n",
					styleDim.Render("Start the agent with `gordion up` to generate logs."),
				)
				return nil
			}
			return fmt.Errorf("open log file: %w", err)
		}
		defer f.Close()

		if logsFollow {
			return tailFollow(f)
		}
		return printLast(f, logsLines)
	},
}

// printLast prints the last N lines of the log file.
func printLast(f *os.File, n int) error {
	// Read all lines, keep last n
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	if len(lines) == 0 {
		fmt.Println(styleDim.Render("  (log file is empty)"))
		return nil
	}

	start := len(lines) - n
	if start < 0 {
		start = 0
	}

	fmt.Println()
	for _, line := range lines[start:] {
		fmt.Println(formatLogLine(line))
	}
	fmt.Println()
	return nil
}

// tailFollow blocks and streams new log lines as they are written (like tail -f).
func tailFollow(f *os.File) error {
	fmt.Printf("  %s  %s\n\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("●"),
		styleDim.Render("Streaming logs... (Ctrl+C to stop)"),
	)

	// Seek to end, then poll for new content
	f.Seek(0, io.SeekEnd)
	reader := bufio.NewReader(f)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			return err
		}
		fmt.Print(formatLogLine(line))
	}
}

// formatLogLine applies color based on log level keywords.
func formatLogLine(line string) string {
	switch {
	case containsAny(line, "ERROR", "error", "ERR"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render("  " + line)
	case containsAny(line, "WARN", "warn", "WRN"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Render("  " + line)
	case containsAny(line, "INFO", "info", "INF"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA")).Render("  " + line)
	case containsAny(line, "DEBUG", "debug", "DBG"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("  " + line)
	default:
		return "  " + line
	}
}

func containsAny(s string, keywords ...string) bool {
	for _, k := range keywords {
		if len(s) >= len(k) {
			for i := 0; i <= len(s)-len(k); i++ {
				if s[i:i+len(k)] == k {
					return true
				}
			}
		}
	}
	return false
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Stream logs in real time (like tail -f)")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "Number of recent lines to show")
	rootCmd.AddCommand(logsCmd)
}
