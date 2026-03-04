package auth

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	identityv1 "github.com/saitddundar/gordion-vpn/pkg/proto/identity/v1"
	"github.com/saitddundar/gordion-vpn/pkg/tlsutil"
)

type Client struct {
	identity identityv1.IdentityServiceClient
	conn     *grpc.ClientConn
}

func NewClient(identityAddr string, caFile string) (*Client, error) {
	var opts []grpc.DialOption

	if caFile != "" {
		creds, err := tlsutil.ClientCredentials(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS creds: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(identityAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to identity service: %w", err)
	}

	return &Client{
		identity: identityv1.NewIdentityServiceClient(conn),
		conn:     conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) ValidateToken(ctx context.Context, token string) (string, error) {
	resp, err := c.identity.ValidateToken(ctx, &identityv1.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		return "", fmt.Errorf("identity service error: %w", err)
	}

	if !resp.Valid {
		return "", fmt.Errorf("invalid token")
	}

	return resp.NodeId, nil
}
