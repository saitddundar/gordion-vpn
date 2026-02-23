package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	configv1 "github.com/saitddundar/gordion-vpn/pkg/proto/config/v1"
	discoveryv1 "github.com/saitddundar/gordion-vpn/pkg/proto/discovery/v1"
	identityv1 "github.com/saitddundar/gordion-vpn/pkg/proto/identity/v1"
)

type Client struct {
	identity  identityv1.IdentityServiceClient
	discovery discoveryv1.DiscoveryServiceClient
	config    configv1.ConfigServiceClient

	identityConn  *grpc.ClientConn
	discoveryConn *grpc.ClientConn
	configConn    *grpc.ClientConn
}

func New(identityAddr, discoveryAddr, configAddr string) (*Client, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	identityConn, err := grpc.Dial(identityAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("identity connection failed: %w", err)
	}

	discoveryConn, err := grpc.Dial(discoveryAddr, opts...)
	if err != nil {
		identityConn.Close()
		return nil, fmt.Errorf("discovery connection failed: %w", err)
	}

	configConn, err := grpc.Dial(configAddr, opts...)
	if err != nil {
		identityConn.Close()
		discoveryConn.Close()
		return nil, fmt.Errorf("config connection failed: %w", err)
	}

	return &Client{
		identity:      identityv1.NewIdentityServiceClient(identityConn),
		discovery:     discoveryv1.NewDiscoveryServiceClient(discoveryConn),
		config:        configv1.NewConfigServiceClient(configConn),
		identityConn:  identityConn,
		discoveryConn: discoveryConn,
		configConn:    configConn,
	}, nil
}

func (c *Client) Close() {
	c.identityConn.Close()
	c.discoveryConn.Close()
	c.configConn.Close()
}

// sends public key to Identity Service and gets back a token
func (c *Client) Register(ctx context.Context, publicKey string) (nodeID, token string, err error) {
	resp, err := c.identity.RegisterNode(ctx, &identityv1.RegisterNodeRequest{
		PublicKey: publicKey,
		Version:   "1.0.0",
	})
	if err != nil {
		return "", "", fmt.Errorf("register failed: %w", err)
	}
	return resp.NodeId, resp.Token, nil
}

// fetches network settings (CIDR, MTU, DNS) from Config Service
func (c *Client) GetNetworkConfig(ctx context.Context, token string) (*configv1.GetConfigResponse, error) {
	resp, err := c.config.GetConfig(ctx, &configv1.GetConfigRequest{
		Token: token,
	})
	if err != nil {
		return nil, fmt.Errorf("get config failed: %w", err)
	}
	return resp, nil
}

// gets a VPN IP address assigned to this node
func (c *Client) RequestIP(ctx context.Context, token, nodeID string) (ip, subnet, gateway string, err error) {
	resp, err := c.config.RequestIP(ctx, &configv1.RequestIPRequest{
		Token:  token,
		NodeId: nodeID,
	})
	if err != nil {
		return "", "", "", fmt.Errorf("request IP failed: %w", err)
	}
	return resp.IpAddress, resp.SubnetMask, resp.Gateway, nil
}

// returns the IP address when disconnecting
func (c *Client) ReleaseIP(ctx context.Context, token, nodeID, ip string) error {
	_, err := c.config.ReleaseIP(ctx, &configv1.ReleaseIPRequest{
		Token:     token,
		NodeId:    nodeID,
		IpAddress: ip,
	})
	if err != nil {
		return fmt.Errorf("release IP failed: %w", err)
	}
	return nil
}

// announces this node to the Discovery Service
func (c *Client) RegisterPeer(ctx context.Context, token, ip string, port int32) error {
	_, err := c.discovery.RegisterPeer(ctx, &discoveryv1.RegisterPeerRequest{
		Token:     token,
		IpAddress: ip,
		Port:      port,
	})
	if err != nil {
		return fmt.Errorf("register peer failed: %w", err)
	}
	return nil
}

// fetches the list of online peers
func (c *Client) DiscoverPeers(ctx context.Context, limit int32) ([]*discoveryv1.Peer, error) {
	resp, err := c.discovery.ListPeers(ctx, &discoveryv1.ListPeersRequest{
		Limit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("discover peers failed: %w", err)
	}
	return resp.Peers, nil
}

// sends a keepalive signal to Discovery Service
func (c *Client) Heartbeat(ctx context.Context, token string) error {
	_, err := c.discovery.Heartbeat(ctx, &discoveryv1.HeartbeatRequest{
		Token: token,
	})
	if err != nil {
		return fmt.Errorf("heartbeat failed: %w", err)
	}
	return nil
}
