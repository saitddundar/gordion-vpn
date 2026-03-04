package test

import (
	"context"
	"testing"
	"time"

	"github.com/saitddundar/gordion-vpn/services/agent/internal/client"
	"github.com/saitddundar/gordion-vpn/services/agent/internal/wireguard"
)

func TestAgentFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := client.New("localhost:8001", "localhost:8002", "localhost:8003", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	// Step 1: Generate keypair
	t.Run("KeyGeneration", func(t *testing.T) {
		kp, err := wireguard.GenerateKeyPair()
		if err != nil {
			t.Fatalf("KeyPair generation failed: %v", err)
		}
		if kp.PrivateKey == "" || kp.PublicKey == "" {
			t.Fatal("Empty keys generated")
		}
		t.Logf("[ok] PrivateKey: %s...", kp.PrivateKey[:16])
		t.Logf("[ok] PublicKey: %s...", kp.PublicKey[:16])
	})

	// Step 2: Register with Identity
	var nodeID, token string
	t.Run("Register", func(t *testing.T) {
		kp, _ := wireguard.GenerateKeyPair()
		var expiresAt int64
		nodeID, token, expiresAt, err = c.Register(ctx, kp.PublicKey, "test-peer-id")
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}
		if nodeID == "" || token == "" {
			t.Fatal("Empty nodeID or token")
		}
		t.Logf("[ok] NodeID: %s", nodeID)
		t.Logf("[ok] Token: %s...", token[:16])
		t.Logf("[ok] ExpiresAt: %d", expiresAt)
	})

	// Step 3: Get network config
	t.Run("GetNetworkConfig", func(t *testing.T) {
		cfg, err := c.GetNetworkConfig(ctx, token)
		if err != nil {
			t.Fatalf("GetNetworkConfig failed: %v", err)
		}
		if cfg.NetworkCidr == "" {
			t.Fatal("Empty CIDR")
		}
		t.Logf("[ok] CIDR: %s, MTU: %d, DNS: %v", cfg.NetworkCidr, cfg.Mtu, cfg.DnsServers)
	})

	// Step 4: Request IP
	var vpnIP string
	t.Run("RequestIP", func(t *testing.T) {
		ip, subnet, gw, err := c.RequestIP(ctx, token, nodeID)
		if err != nil {
			t.Fatalf("RequestIP failed: %v", err)
		}
		vpnIP = ip
		t.Logf("[ok] IP: %s, Subnet: %s, Gateway: %s", ip, subnet, gw)
	})

	// Step 5: Register as peer
	t.Run("RegisterPeer", func(t *testing.T) {
		err := c.RegisterPeer(ctx, token, vpnIP, 51820, "test-peer-id", []string{"/ip4/127.0.0.1/tcp/4001/p2p/test-peer-id"}, false)
		if err != nil {
			t.Fatalf("RegisterPeer failed: %v", err)
		}
		t.Log("[ok] Peer registered")
	})

	// Step 6: Discover peers
	t.Run("DiscoverPeers", func(t *testing.T) {
		peers, err := c.DiscoverPeers(ctx, token, 10)
		if err != nil {
			t.Fatalf("DiscoverPeers failed: %v", err)
		}
		t.Logf("[ok] Found %d peers", len(peers))
	})

	// Step 7: Heartbeat
	t.Run("Heartbeat", func(t *testing.T) {
		err := c.Heartbeat(ctx, token)
		if err != nil {
			t.Fatalf("Heartbeat failed: %v", err)
		}
		t.Log("[ok] Heartbeat sent")
	})

	// Step 8: Release IP
	t.Run("ReleaseIP", func(t *testing.T) {
		err := c.ReleaseIP(ctx, token, nodeID, vpnIP)
		if err != nil {
			t.Fatalf("ReleaseIP failed: %v", err)
		}
		t.Log("[ok] IP released")
	})
}
