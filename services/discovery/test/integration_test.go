package test

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	discoveryv1 "github.com/saitddundar/gordion-vpn/pkg/proto/discovery/v1"
)

func TestDiscoveryService(t *testing.T) {
	conn, err := grpc.Dial("localhost:8002", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := discoveryv1.NewDiscoveryServiceClient(conn)
	ctx := context.Background()

	// Test 1: Register Peer
	t.Run("RegisterPeer", func(t *testing.T) {
		resp, err := client.RegisterPeer(ctx, &discoveryv1.RegisterPeerRequest{
			Token:     "testtoken123_abc",
			IpAddress: "192.168.1.100",
			Port:      51820,
			Region:    "eu-west",
			Bandwidth: 1000000,
		})
		if err != nil {
			t.Fatalf("RegisterPeer failed: %v", err)
		}
		if !resp.Success {
			t.Fatalf("RegisterPeer not successful: %s", resp.Message)
		}
		t.Logf("[ok] Peer registered: %s", resp.Message)
	})

	// Test 2: List Peers (requires auth token in metadata)
	t.Run("ListPeers", func(t *testing.T) {
		md := metadata.Pairs("authorization", "testtoken123_abc")
		ctxWithToken := metadata.NewOutgoingContext(ctx, md)
		resp, err := client.ListPeers(ctxWithToken, &discoveryv1.ListPeersRequest{
			Limit: 10,
		})
		if err != nil {
			t.Fatalf("ListPeers failed: %v", err)
		}
		if len(resp.Peers) == 0 {
			t.Fatal("Expected at least 1 peer")
		}
		t.Logf("[ok] Found %d peers", len(resp.Peers))
		for _, p := range resp.Peers {
			t.Logf("   Peer: %s (last_seen: %d)", p.NodeId, p.LastSeen)
		}
	})

	// Test 3: Heartbeat
	t.Run("Heartbeat", func(t *testing.T) {
		resp, err := client.Heartbeat(ctx, &discoveryv1.HeartbeatRequest{
			Token:     "testtoken123_abc",
			Bandwidth: 500000,
		})
		if err != nil {
			t.Fatalf("Heartbeat failed: %v", err)
		}
		if !resp.Success {
			t.Fatal("Heartbeat not successful")
		}
		t.Logf("[ok] Heartbeat success, TTL: %d seconds", resp.Ttl)
	})
}
