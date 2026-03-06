package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/saitddundar/gordion-vpn/cli/internal/cliconfig"
	"github.com/saitddundar/gordion-vpn/cli/internal/state"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Connect to Gordion VPN",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Check if already running
		s, err := state.Read()
		if err == nil && s != nil && s.IsRunning() {
			printWarn(fmt.Sprintf("Gordion is already running (PID %d, VPN: %s)", s.PID, s.VPNAddr))
			return nil
		}

		// 2. Load config
		cfg, err := cliconfig.Load(cfgFile)
		if err != nil {
			printErrorExit(err.Error())
		}

		// 3. Find agent binary
		agentBin, err := findAgentBinary()
		if err != nil {
			printErrorExit(err.Error())
		}

		// 4. Build args
		agentArgs := []string{}
		if cfgFile != "" {
			agentArgs = append(agentArgs, "--config", cfgFile)
		}

		// 5. Open log file (stdout + stderr from agent go here)
		logPath := state.DefaultLogPath()
		if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
			return fmt.Errorf("create log dir: %w", err)
		}
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			printWarn("Could not open log file, agent will run without logs: " + err.Error())
			logPath = ""
		}

		// 6. Start agent in background
		fmt.Printf("%s Starting Gordion VPN...\n", styleDim.Render("→"))

		agentCmd := exec.Command(agentBin, agentArgs...)
		if logFile != nil {
			agentCmd.Stdout = logFile
			agentCmd.Stderr = logFile
		}
		agentCmd.Stdin = nil

		// Detach from current process group so it survives terminal close
		setSysProcAttr(agentCmd)

		if err := agentCmd.Start(); err != nil {
			if logFile != nil {
				logFile.Close()
			}
			return fmt.Errorf("failed to start agent: %w\n  binary: %s", err, agentBin)
		}

		// Close our handle to the log file — agent holds its own
		if logFile != nil {
			logFile.Close()
		}

		pid := agentCmd.Process.Pid

		// 7. Write state file
		s = &state.State{
			PID:         pid,
			StartedAt:   time.Now(),
			LogFile:     logPath,
			ConfigFile:  resolvedConfigPath(cfgFile),
			IsExitNode:  cfg.IsExitNode,
			UseExitNode: cfg.UseExitNode,
			ExitNodeID:  cfg.ExitNodeID,
		}
		if err := state.Write(s); err != nil {
			printWarn("Could not write state file: " + err.Error())
		}

		printOK(fmt.Sprintf("Gordion VPN started (PID %d)", pid))

		if logPath != "" {
			fmt.Printf("  %s %s\n",
				styleDim.Render("logs →"),
				styleDim.Render(logPath),
			)
		}

		if cfg.IsExitNode {
			printOK("This node is an " + styleBold.Render("exit node"))
		}
		if cfg.UseExitNode {
			target := cfg.ExitNodeID
			if target == "" {
				target = "auto"
			}
			printOK("Exit node: " + styleBold.Render(target))
		}

		fmt.Printf("\n%s\n",
			styleDim.Render("Run `gordion status` to check connection."),
		)
		return nil
	},
}

func findAgentBinary() (string, error) {
	name := "gordion-agent"
	if runtime.GOOS == "windows" {
		name = "gordion-agent.exe"
	}

	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}

	exe, err := os.Executable()
	if err == nil {
		sibling := filepath.Join(filepath.Dir(exe), name)
		if _, err := os.Stat(sibling); err == nil {
			return sibling, nil
		}
	}

	return "", fmt.Errorf(
		"agent binary %q not found in PATH\n"+
			"  Build it first: cd services/agent && go build -o gordion-agent ./cmd/agent",
		name,
	)
}

func resolvedConfigPath(cfgFile string) string {
	if cfgFile == "" {
		for _, p := range cliconfig.DefaultPaths() {
			if _, err := os.Stat(p); err == nil {
				abs, _ := filepath.Abs(p)
				return abs
			}
		}
		return ""
	}
	abs, _ := filepath.Abs(cfgFile)
	return abs
}

func init() {
	rootCmd.AddCommand(upCmd)
}
