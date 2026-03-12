package test

import (
	"context"
	"os"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	discoveryv1 "github.com/saitddundar/gordion-vpn/pkg/proto/discovery/v1"
	identityv1 "github.com/saitddundar/gordion-vpn/pkg/proto/identity/v1"
)

func getNetworkSecret() string {
	if s := os.Getenv("NETWORK_SECRET"); s != "" {
		return s
	}
	return "gordion_secret_key"
}

// getTestToken registers a node with Identity Service and returns a valid JWT.
func getTestToken(t *testing.T) string {
	t.Helper()

	conn, err := grpc.NewClient("localhost:8001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("connect to Identity: %v", err)
	}
	defer conn.Close()

	client := identityv1.NewIdentityServiceClient(conn)
	resp, err := client.RegisterNode(context.Background(), &identityv1.RegisterNodeRequest{
		PublicKey:     "dGVzdHB1YmxpY2tleWZvcmRpc2NvdmVyeXRlc3Q=", // base64 test key
		Version:       "test",
		PeerId:        "discovery-test-peer",
		NetworkSecret: getNetworkSecret(),
	})
	if err != nil {
		t.Fatalf("RegisterNode (for token): %v", err)
	}
	return resp.Token
}

func TestDiscoveryService(t *testing.T) {
	// Step 0: get a real JWT from Identity Service
	token := getTestToken(t)

	conn, err := grpc.NewClient("localhost:8002", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := discoveryv1.NewDiscoveryServiceClient(conn)
	ctx := context.Background()

	// Test 1: Register Peer (token in request body)
	t.Run("RegisterPeer", func(t *testing.T) {
		resp, err := client.RegisterPeer(ctx, &discoveryv1.RegisterPeerRequest{
			Token:     token,
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

	// Test 2: List Peers (token in gRPC metadata)
	t.Run("ListPeers", func(t *testing.T) {
		md := metadata.Pairs("authorization", token)
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

	// Test 3: Heartbeat (token in request body)
	t.Run("Heartbeat", func(t *testing.T) {
		resp, err := client.Heartbeat(ctx, &discoveryv1.HeartbeatRequest{
			Token:     token,
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
