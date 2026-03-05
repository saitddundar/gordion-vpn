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

var peersLimit int32

var peersCmd = &cobra.Command{
	Use:   "peers",
	Short: "List peers in the Gordion network",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := cliconfig.Load(cfgFile)
		if err != nil {
			printErrorExit(err.Error())
		}

		// Get token from state file (agent sets it; if not available we try without)
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

		// Attach token if we have one
		if token != "" {
			md := metadata.Pairs("authorization", token)
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		resp, err := client.ListPeers(ctx, &discoveryv1.ListPeersRequest{
			Limit: peersLimit,
		})
		if err != nil {
			return fmt.Errorf("ListPeers: %w\n  (Is the agent running and connected?)", err)
		}

		if len(resp.Peers) == 0 {
			fmt.Printf("\n  %s\n\n", styleDim.Render("No peers found in the network."))
			return nil
		}

		printPeers(resp.Peers)
		return nil
	},
}

func printPeers(peers []*discoveryv1.Peer) {

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED")).
		Render(fmt.Sprintf("  %d peer(s) in network\n", len(peers)))
	fmt.Println()
	fmt.Println(header)

	colID := 20
	colIP := 14
	colRegion := 10

	colStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	fmt.Printf("  %-2s  %-*s  %-*s  %-*s  %s\n",
		"",
		colID, colStyle.Render("NODE ID"),
		colIP, colStyle.Render("VPN ADDRESS"),
		colRegion, colStyle.Render("REGION"),
		colStyle.Render("LAST SEEN"),
	)
	fmt.Printf("  %s\n", styleDim.Render(strings.Repeat("─", 64)))

	now := time.Now().Unix()
	for _, p := range peers {
		dot, dotColor := onlineDot(p.LastSeen, now)
		_ = dotColor

		nodeID := truncate(p.NodeId, colID)
		ip := truncate(p.IpAddress, colIP)
		region := truncate(p.Region, colRegion)
		lastSeen := formatLastSeen(p.LastSeen, now)

		suffix := ""
		if p.IsExitNode {
			suffix = "  " + lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true).
				Render("[EXIT NODE]")
		}

		fmt.Printf("  %s   %-*s  %-*s  %-*s  %s%s\n",
			dot,
			colID, nodeID,
			colIP, ip,
			colRegion, region,
			lastSeen,
			suffix,
		)
	}
	fmt.Println()
}

func onlineDot(lastSeen, now int64) (string, string) {
	age := now - lastSeen
	if age < 60 { // online: heartbeat within last minute
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("●"), "#10B981"
	}
	if age < 300 { // idle: 1-5 min
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Render("●"), "#F59E0B"
	}
	// offline
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("○"), "#6B7280"
}

func formatLastSeen(lastSeen, now int64) string {
	age := now - lastSeen
	if age < 5 {
		return styleDim.Render("just now")
	}
	if age < 60 {
		return styleDim.Render(fmt.Sprintf("%ds ago", age))
	}
	if age < 3600 {
		return styleDim.Render(fmt.Sprintf("%dm ago", age/60))
	}
	return styleDim.Render(fmt.Sprintf("%dh ago", age/3600))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func init() {
	peersCmd.Flags().Int32VarP(&peersLimit, "limit", "n", 50, "Max number of peers to show")
	rootCmd.AddCommand(peersCmd)
}
