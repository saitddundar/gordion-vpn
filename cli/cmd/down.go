package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/saitddundar/gordion-vpn/cli/internal/state"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop the Gordion VPN agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := state.Read()
		if err != nil {
			return fmt.Errorf("read state: %w", err)
		}
		if s == nil {
			printWarn("Gordion is not running (no state file found)")
			return nil
		}
		if !s.IsRunning() {
			printWarn(fmt.Sprintf("Gordion process (PID %d) is not running — cleaning up state file", s.PID))
			_ = state.Delete()
			return nil
		}

		fmt.Printf("%s Stopping Gordion VPN agent (PID %d)...\n",
			styleDim.Render("→"), s.PID)

		proc, err := os.FindProcess(s.PID)
		if err != nil {
			return fmt.Errorf("find process: %w", err)
		}

		// Send SIGTERM (graceful shutdown) — agent handles this via context cancel
		if err := sendInterrupt(proc); err != nil {
			return fmt.Errorf("send stop signal: %w", err)
		}

		// Wait up to 10 seconds for graceful exit
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if !s.IsRunning() {
				break
			}
			time.Sleep(300 * time.Millisecond)
		}

		if s.IsRunning() {
			// Force kill if graceful shutdown timed out
			_ = proc.Kill()
			printWarn("Agent did not stop gracefully — force killed")
		}

		_ = state.Delete()
		printOK("Gordion VPN stopped")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
