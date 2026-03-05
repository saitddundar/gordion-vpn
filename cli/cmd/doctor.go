package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/saitddundar/gordion-vpn/cli/internal/cliconfig"
	"github.com/saitddundar/gordion-vpn/cli/internal/state"
	configv1 "github.com/saitddundar/gordion-vpn/pkg/proto/config/v1"
	discoveryv1 "github.com/saitddundar/gordion-vpn/pkg/proto/discovery/v1"
	identityv1 "github.com/saitddundar/gordion-vpn/pkg/proto/identity/v1"
)

type CheckResult struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run network and connectivity diagnostics",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := cliconfig.Load(cfgFile)
		if err != nil {
			printErrorExit(err.Error())
		}

		results := runChecks(cfg)

		if outputJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(results)
		}

		printDoctorResults(results)
		return nil
	},
}

func runChecks(cfg *cliconfig.Config) []CheckResult {
	var checks []CheckResult

	// 1. Agent running
	s, _ := state.Read()
	if s != nil && s.IsRunning() {
		checks = append(checks, CheckResult{"Agent process", true, fmt.Sprintf("Running (PID %d)", s.PID)})
	} else {
		checks = append(checks, CheckResult{"Agent process", false, "Not running — start with `gordion up`"})
	}

	// 2. Identity Service reachable
	checks = append(checks, grpcCheck("Identity Service", cfg.IdentityAddr, func(conn *grpc.ClientConn) error {
		cl := identityv1.NewIdentityServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := cl.ValidateToken(ctx, &identityv1.ValidateTokenRequest{Token: "probe"})
		// Any response (even Unauthenticated) means the service is up
		if err != nil {
			st, _ := err.(interface {
				GRPCStatus() interface{ Code() uint32 }
			})
			_ = st
			// grpc status Unauthenticated = service is alive
			if isGRPCAlive(err) {
				return nil
			}
		}
		return err
	}))

	// 3. Discovery Service reachable
	checks = append(checks, grpcCheck("Discovery Service", cfg.DiscoveryAddr, func(conn *grpc.ClientConn) error {
		cl := discoveryv1.NewDiscoveryServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := cl.ListPeers(ctx, &discoveryv1.ListPeersRequest{Limit: 1})
		if isGRPCAlive(err) {
			return nil
		}
		return err
	}))

	// 4. Config Service reachable
	checks = append(checks, grpcCheck("Config Service", cfg.ConfigAddr, func(conn *grpc.ClientConn) error {
		cl := configv1.NewConfigServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := cl.GetConfig(ctx, &configv1.GetConfigRequest{Token: "probe"})
		if isGRPCAlive(err) {
			return nil
		}
		return err
	}))

	// 5. WireGuard port available
	wpPort := fmt.Sprintf(":%d", cfg.WireGuardPort)
	if s != nil && s.IsRunning() {
		checks = append(checks, CheckResult{
			"WireGuard port",
			true,
			fmt.Sprintf("Port %d in use by agent (expected)", cfg.WireGuardPort),
		})
	} else {
		ln, err := net.ListenPacket("udp", wpPort)
		if err != nil {
			checks = append(checks, CheckResult{"WireGuard port", false, fmt.Sprintf("Port %d unavailable: %v", cfg.WireGuardPort, err)})
		} else {
			ln.Close()
			checks = append(checks, CheckResult{"WireGuard port", true, fmt.Sprintf("Port %d is free", cfg.WireGuardPort)})
		}
	}

	// 6. P2P port available
	p2pPort := fmt.Sprintf(":%d", cfg.P2PPort)
	ln, err := net.Listen("tcp", p2pPort)
	if err != nil && s == nil {
		checks = append(checks, CheckResult{"P2P port", false, fmt.Sprintf("Port %d unavailable: %v", cfg.P2PPort, err)})
	} else {
		if ln != nil {
			ln.Close()
		}
		checks = append(checks, CheckResult{"P2P port", true, fmt.Sprintf("Port %d OK", cfg.P2PPort)})
	}

	// 7. WireGuard binary present (if not dry-run)
	wgBin := "wg"
	if runtime.GOOS == "windows" {
		wgBin = "wg.exe"
	}
	if _, err := exec.LookPath(wgBin); err != nil {
		checks = append(checks, CheckResult{"WireGuard binary", false, "Not found in PATH (required for real tunnel)"})
	} else {
		checks = append(checks, CheckResult{"WireGuard binary", true, "Found in PATH"})
	}

	return checks
}

func grpcCheck(name, addr string, fn func(conn *grpc.ClientConn) error) CheckResult {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return CheckResult{name, false, fmt.Sprintf("cannot connect to %s: %v", addr, err)}
	}
	defer conn.Close()

	if err := fn(conn); err != nil {
		return CheckResult{name, false, fmt.Sprintf("unreachable (%s): %v", addr, err)}
	}
	return CheckResult{name, true, fmt.Sprintf("OK (%s)", addr)}
}

func isGRPCAlive(err error) bool {
	if err == nil {
		return true
	}
	s := err.Error()
	for _, code := range []string{"Unauthenticated", "InvalidArgument", "PermissionDenied", "NotFound"} {
		if len(s) >= len(code) {
			for i := 0; i <= len(s)-len(code); i++ {
				if s[i:i+len(code)] == code {
					return true
				}
			}
		}
	}
	return false
}

func printDoctorResults(results []CheckResult) {
	fmt.Println()
	fmt.Printf(" %s\n\n",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED")).Render("─ Diagnostics ──────────────────────────────────────────────────────"),
	)

	allOK := true
	for _, r := range results {
		var icon, nameStr, msgStr string
		if r.OK {
			icon = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("✓")
			nameStr = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E5E7EB")).Render(r.Name)
			msgStr = styleDim.Render(r.Message)
		} else {
			icon = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render("✗")
			nameStr = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EF4444")).Render(r.Name)
			msgStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Render(r.Message)
			allOK = false
		}
		fmt.Printf("  %s  %-22s  %s\n", icon, nameStr, msgStr)
	}

	fmt.Println()
	if allOK {
		printOK("All checks passed")
	} else {
		printWarn("Some checks failed — see above for details")
	}
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
