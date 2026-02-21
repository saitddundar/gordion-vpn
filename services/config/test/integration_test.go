package test

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	configv1 "github.com/saitddundar/gordion-vpn/pkg/proto/config/v1"
)

func TestConfigService(t *testing.T) {
	conn, err := grpc.Dial("localhost:8003", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := configv1.NewConfigServiceClient(conn)
	ctx := context.Background()

	// Test 1: Get Config
	t.Run("GetConfig", func(t *testing.T) {
		resp, err := client.GetConfig(ctx, &configv1.GetConfigRequest{
			Token: "test_token_123",
		})
		if err != nil {
			t.Fatalf("GetConfig failed: %v", err)
		}
		if resp.NetworkCidr == "" {
			t.Fatal("Expected network CIDR")
		}
		t.Logf("[ok] Network: %s, MTU: %d, DNS: %v", resp.NetworkCidr, resp.Mtu, resp.DnsServers)
	})

	// Test 2: Request IP
	t.Run("RequestIP", func(t *testing.T) {
		resp, err := client.RequestIP(ctx, &configv1.RequestIPRequest{
			Token:  "test_token_123",
			NodeId: "test-node-001",
		})
		if err != nil {
			t.Fatalf("RequestIP failed: %v", err)
		}
		if resp.IpAddress == "" {
			t.Fatal("Expected IP address")
		}
		t.Logf("[ok] IP: %s, Subnet: %s, Gateway: %s", resp.IpAddress, resp.SubnetMask, resp.Gateway)

		// Test 3: Request same node again (should get same IP)
		resp2, err := client.RequestIP(ctx, &configv1.RequestIPRequest{
			Token:  "test_token_123",
			NodeId: "test-node-001",
		})
		if err != nil {
			t.Fatalf("RequestIP (same node) failed: %v", err)
		}
		if resp2.IpAddress != resp.IpAddress {
			t.Fatalf("Expected same IP, got %s vs %s", resp.IpAddress, resp2.IpAddress)
		}
		t.Logf("[ok] Same node got same IP: %s", resp2.IpAddress)
	})

	// Test 4: Release IP
	t.Run("ReleaseIP", func(t *testing.T) {
		resp, err := client.ReleaseIP(ctx, &configv1.ReleaseIPRequest{
			Token:     "test_token_123",
			NodeId:    "test-node-001",
			IpAddress: "10.8.0.2",
		})
		if err != nil {
			t.Fatalf("ReleaseIP failed: %v", err)
		}
		if !resp.Success {
			t.Fatal("ReleaseIP not successful")
		}
		t.Logf("[ok] IP released: %s", resp.Message)
	})
}
