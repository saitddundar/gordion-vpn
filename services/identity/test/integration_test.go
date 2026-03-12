package test

import (
	"context"
	"os"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	identityv1 "github.com/saitddundar/gordion-vpn/pkg/proto/identity/v1"
)

func getNetworkSecret() string {
	if s := os.Getenv("NETWORK_SECRET"); s != "" {
		return s
	}
	return "gordion_secret_key" // matches configs/identity.dev.yaml default
}

func TestIdentityService(t *testing.T) {
	// Connect to running service
	conn, err := grpc.NewClient("localhost:8001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := identityv1.NewIdentityServiceClient(conn)
	ctx := context.Background()

	// Test: Register Node
	t.Run("RegisterNode", func(t *testing.T) {
		resp, err := client.RegisterNode(ctx, &identityv1.RegisterNodeRequest{
			PublicKey:     "test_public_key_abc123",
			Version:       "1.0.0",
			NetworkSecret: getNetworkSecret(),
		})
		if err != nil {
			t.Fatalf("RegisterNode failed: %v", err)
		}

		t.Logf("[✓] Node registered successfully!")
		t.Logf("   Node ID: %s", resp.NodeId)
		t.Logf("   Token: %s...", resp.Token[:50])
		t.Logf("   Expires At: %d", resp.ExpiresAt)

		// Test: Validate Token
		validateResp, err := client.ValidateToken(ctx, &identityv1.ValidateTokenRequest{
			Token: resp.Token,
		})
		if err != nil {
			t.Fatalf("ValidateToken failed: %v", err)
		}

		if !validateResp.Valid {
			t.Fatal(" [X] Token should be valid!")
		}

		t.Logf(" [✓] Token validated!")
		t.Logf("   Valid: %v", validateResp.Valid)
		t.Logf("   Node ID: %s", validateResp.NodeId)

		// Test: Get Public Key
		keyResp, err := client.GetPublicKey(ctx, &identityv1.GetPublicKeyRequest{
			NodeId: resp.NodeId,
		})
		if err != nil {
			t.Fatalf("GetPublicKey failed: %v", err)
		}

		if keyResp.PublicKey != "test_public_key_abc123" {
			t.Fatal(" [X] Public key mismatch!")
		}

		t.Logf(" [✓] Public key retrieved!")
		t.Logf("   Public Key: %s", keyResp.PublicKey)
	})
}
