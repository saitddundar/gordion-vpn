package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/saitddundar/gordion-vpn/pkg/circuitbreaker"
	configv1 "github.com/saitddundar/gordion-vpn/pkg/proto/config/v1"
	discoveryv1 "github.com/saitddundar/gordion-vpn/pkg/proto/discovery/v1"
	identityv1 "github.com/saitddundar/gordion-vpn/pkg/proto/identity/v1"
	"github.com/saitddundar/gordion-vpn/pkg/tlsutil"
	"github.com/saitddundar/gordion-vpn/pkg/tracing"
)

type Client struct {
	identity  identityv1.IdentityServiceClient
	discovery discoveryv1.DiscoveryServiceClient
	config    configv1.ConfigServiceClient

	identityConn  *grpc.ClientConn
	discoveryConn *grpc.ClientConn
	configConn    *grpc.ClientConn

	cbIdentity  *circuitbreaker.CircuitBreaker
	cbDiscovery *circuitbreaker.CircuitBreaker
	cbConfig    *circuitbreaker.CircuitBreaker
}

func newCBConfig(name string) circuitbreaker.Config {
	return circuitbreaker.Config{
		MaxFailures: 5,
		OpenTimeout: 30 * time.Second,
		HalfOpenMax: 1,
		OnStateChange: func(n string, from, to circuitbreaker.State) {
			fmt.Printf("[circuit-breaker] [%s] %s → %s\n", n, from, to)
		},
	}
}

// If caFile is non-empty, TLS is used; otherwise the connection is insecure (dev/test only).
func New(identityAddr, discoveryAddr, configAddr, caFile string) (*Client, error) {
	var transportCreds grpc.DialOption
	if caFile != "" {
		creds, err := tlsutil.ClientCredentials(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS CA cert: %w", err)
		}
		transportCreds = grpc.WithTransportCredentials(creds)
	} else {
		transportCreds = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	opts := []grpc.DialOption{
		transportCreds,
		grpc.WithUnaryInterceptor(tracing.ClientInterceptor()),
	}

	identityConn, err := grpc.NewClient(identityAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("identity connection failed: %w", err)
	}

	discoveryConn, err := grpc.NewClient(discoveryAddr, opts...)
	if err != nil {
		identityConn.Close()
		return nil, fmt.Errorf("discovery connection failed: %w", err)
	}

	configConn, err := grpc.NewClient(configAddr, opts...)
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

		cbIdentity:  circuitbreaker.New("identity", newCBConfig("identity")),
		cbDiscovery: circuitbreaker.New("discovery", newCBConfig("discovery")),
		cbConfig:    circuitbreaker.New("config", newCBConfig("config")),
	}, nil
}

func (c *Client) Close() {
	c.identityConn.Close()
	c.discoveryConn.Close()
	c.configConn.Close()
}

func (c *Client) Register(ctx context.Context, publicKey, peerID string) (nodeID, token string, expiresAt int64, err error) {
	var resp *identityv1.RegisterNodeResponse
	err = c.cbIdentity.Execute(func() error {
		var e error
		resp, e = c.identity.RegisterNode(ctx, &identityv1.RegisterNodeRequest{
			PublicKey: publicKey,
			Version:   "1.0.0",
			PeerId:    peerID,
		})
		return e
	})
	if err != nil {
		return "", "", 0, fmt.Errorf("register failed: %w", err)
	}
	return resp.NodeId, resp.Token, resp.ExpiresAt, nil
}

func (c *Client) GetPeerPublicKey(ctx context.Context, nodeID string) (string, error) {
	var resp *identityv1.GetPublicKeyResponse
	err := c.cbIdentity.Execute(func() error {
		var e error
		resp, e = c.identity.GetPublicKey(ctx, &identityv1.GetPublicKeyRequest{
			NodeId: nodeID,
		})
		return e
	})
	if err != nil {
		return "", fmt.Errorf("get public key failed: %w", err)
	}
	return resp.PublicKey, nil
}

func (c *Client) GetNetworkConfig(ctx context.Context, token string) (*configv1.GetConfigResponse, error) {
	var resp *configv1.GetConfigResponse
	err := c.cbConfig.Execute(func() error {
		var e error
		resp, e = c.config.GetConfig(ctx, &configv1.GetConfigRequest{
			Token: token,
		})
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("get config failed: %w", err)
	}
	return resp, nil
}

func (c *Client) RequestIP(ctx context.Context, token, nodeID string) (ip, subnet, gateway string, err error) {
	var resp *configv1.RequestIPResponse
	err = c.cbConfig.Execute(func() error {
		var e error
		resp, e = c.config.RequestIP(ctx, &configv1.RequestIPRequest{
			Token:  token,
			NodeId: nodeID,
		})
		return e
	})
	if err != nil {
		return "", "", "", fmt.Errorf("request IP failed: %w", err)
	}
	return resp.IpAddress, resp.SubnetMask, resp.Gateway, nil
}

func (c *Client) ReleaseIP(ctx context.Context, token, nodeID, ip string) error {
	return c.cbConfig.Execute(func() error {
		_, e := c.config.ReleaseIP(ctx, &configv1.ReleaseIPRequest{
			Token:     token,
			NodeId:    nodeID,
			IpAddress: ip,
		})
		return e
	})
}

func (c *Client) RegisterPeer(ctx context.Context, token, ip string, port int32, peerID string, p2pAddrs []string) error {
	return c.cbDiscovery.Execute(func() error {
		_, e := c.discovery.RegisterPeer(ctx, &discoveryv1.RegisterPeerRequest{
			Token:     token,
			IpAddress: ip,
			Port:      port,
			Region:    "global",
			Bandwidth: 1000000,
			PeerId:    peerID,
			P2PAddrs:  p2pAddrs,
		})
		return e
	})
}

func (c *Client) DiscoverPeers(ctx context.Context, limit int32) ([]*discoveryv1.Peer, error) {
	var resp *discoveryv1.ListPeersResponse
	err := c.cbDiscovery.Execute(func() error {
		var e error
		resp, e = c.discovery.ListPeers(ctx, &discoveryv1.ListPeersRequest{
			Limit: limit,
		})
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("discover peers failed: %w", err)
	}
	return resp.Peers, nil
}

func (c *Client) Heartbeat(ctx context.Context, token string) error {
	return c.cbDiscovery.Execute(func() error {
		_, e := c.discovery.Heartbeat(ctx, &discoveryv1.HeartbeatRequest{
			Token: token,
		})
		return e
	})
}
