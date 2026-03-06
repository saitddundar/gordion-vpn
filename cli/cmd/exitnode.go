package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/saitddundar/gordion-vpn/cli/internal/cliconfig"
	"github.com/saitddundar/gordion-vpn/cli/internal/state"
	discoveryv1 "github.com/saitddundar/gordion-vpn/pkg/proto/discovery/v1"
)

// ─── Parent command ────────────────────────────────────────────────────────────

var exitNodeCmd = &cobra.Command{
	Use:   "exit-node",
	Short: "Manage exit nodes for internet traffic routing",
	Long: styleBold.Render("Exit Nodes") + "\n" +
		styleDim.Render("Route all internet traffic through a peer, masking your IP.\n\n") +
		styleDim.Render("  gordion exit-node list       — show available exit nodes\n") +
		styleDim.Render("  gordion exit-node set <id>   — use a specific exit node\n") +
		styleDim.Render("  gordion exit-node off        — disable exit node routing"),
	// No Run — show help by default
}

// ─── exit-node list ────────────────────────────────────────────────────────────

var exitNodeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available exit nodes in the network",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := cliconfig.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		s, _ := state.Read()
		token := ""
		if s != nil {
			token = s.Token
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := grpc.NewClient(cfg.DiscoveryAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return fmt.Errorf("connect to Discovery (%s): %w", cfg.DiscoveryAddr, err)
		}
		defer conn.Close()

		client := discoveryv1.NewDiscoveryServiceClient(conn)

		if token != "" {
			md := metadata.Pairs("authorization", token)
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		resp, err := client.ListPeers(ctx, &discoveryv1.ListPeersRequest{Limit: 100})
		if err != nil {
			return fmt.Errorf("ListPeers: %w", err)
		}

		// Filter exit nodes only
		var exits []*discoveryv1.Peer
		for _, p := range resp.Peers {
			if p.IsExitNode {
				exits = append(exits, p)
			}
		}

		if len(exits) == 0 {
			fmt.Printf("\n  %s\n\n", styleDim.Render("No exit nodes available in the network."))
			fmt.Printf("  %s\n\n",
				styleDim.Render("To create one: start an agent with  is_exit_node: true  on a VPS."),
			)
			return nil
		}

		printExitNodes(exits, s)
		return nil
	},
}

// ─── exit-node set ─────────────────────────────────────────────────────────────

var exitNodeSetCmd = &cobra.Command{
	Use:   "set [node-id]",
	Short: "Use a specific exit node (empty = auto-select)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := state.Read()
		if err != nil {
			return fmt.Errorf("read state: %w", err)
		}
		if s == nil || !s.IsRunning() {
			return fmt.Errorf("Gordion is not running — start it first with `gordion up`")
		}

		targetID := ""
		if len(args) > 0 {
			targetID = args[0]
		}

		s.UseExitNode = true
		s.ExitNodeID = targetID
		if err := state.Write(s); err != nil {
			return fmt.Errorf("write state: %w", err)
		}

		if targetID == "" {
			printOK("Exit node set to " + styleBold.Render("auto-select"))
		} else {
			printOK("Exit node set to " + styleBold.Render(targetID))
		}
		fmt.Printf("\n  %s\n\n",
			styleDim.Render("Restart the agent for changes to take effect: gordion down && gordion up"),
		)
		return nil
	},
}

// ─── exit-node off ─────────────────────────────────────────────────────────────

var exitNodeOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Disable exit node routing (mesh VPN only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := state.Read()
		if err != nil {
			return fmt.Errorf("read state: %w", err)
		}
		if s == nil || !s.IsRunning() {
			return fmt.Errorf("Gordion is not running — start it first with `gordion up`")
		}

		s.UseExitNode = false
		s.ExitNodeID = ""
		if err := state.Write(s); err != nil {
			return fmt.Errorf("write state: %w", err)
		}

		printOK("Exit node disabled — back to mesh VPN mode")
		fmt.Printf("\n  %s\n\n",
			styleDim.Render("Restart the agent for changes to take effect: gordion down && gordion up"),
		)
		return nil
	},
}

// ─── Display ──────────────────────────────────────────────────────────────────

func printExitNodes(exits []*discoveryv1.Peer, s *state.State) {
	activeID := ""
	if s != nil && s.UseExitNode {
		activeID = s.ExitNodeID // empty = auto
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED")).
		Render(fmt.Sprintf("  %d exit node(s) available\n", len(exits)))

	fmt.Println()
	fmt.Println(header)

	colID := 22
	colIP := 14
	colStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	fmt.Printf("  %-2s  %-*s  %-*s  %s\n",
		"",
		colID, colStyle.Render("NODE ID"),
		colIP, colStyle.Render("VPN ADDRESS"),
		colStyle.Render("LAST SEEN"),
	)
	fmt.Printf("  %s\n", styleDim.Render(strings.Repeat("─", 56)))

	now := time.Now().Unix()
	for _, p := range exits {
		dot, _ := onlineDot(p.LastSeen, now)
		nodeID := truncate(p.NodeId, colID)
		ip := truncate(p.IpAddress, colIP)
		lastSeen := formatLastSeen(p.LastSeen, now)

		active := ""
		if s != nil && s.UseExitNode {
			if activeID == "" || activeID == p.NodeId {
				active = "  " + lipgloss.NewStyle().
					Foreground(lipgloss.Color("#10B981")).
					Bold(true).
					Render("← active")
			}
		}

		fmt.Printf("  %s   %-*s  %-*s  %s%s\n",
			dot,
			colID, nodeID,
			colIP, ip,
			lastSeen,
			active,
		)
	}

	fmt.Println()
	fmt.Printf("  %s\n\n",
		styleDim.Render("Use: gordion exit-node set <node-id>   or   gordion exit-node set  (auto)"),
	)
}

// ─── Init ─────────────────────────────────────────────────────────────────────

func init() {
	exitNodeCmd.AddCommand(exitNodeListCmd)
	exitNodeCmd.AddCommand(exitNodeSetCmd)
	exitNodeCmd.AddCommand(exitNodeOffCmd)
	rootCmd.AddCommand(exitNodeCmd)
}
