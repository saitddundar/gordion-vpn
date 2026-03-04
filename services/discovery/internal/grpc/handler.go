package grpc

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/saitddundar/gordion-vpn/pkg/auth"
	discoveryv1 "github.com/saitddundar/gordion-vpn/pkg/proto/discovery/v1"
	"github.com/saitddundar/gordion-vpn/services/discovery/internal/matcher"
	"github.com/saitddundar/gordion-vpn/services/discovery/internal/registry"
)

type DiscoveryHandler struct {
	discoveryv1.UnimplementedDiscoveryServiceServer
	registry   *registry.Registry
	matcher    *matcher.Matcher
	authClient *auth.Client
}

func NewDiscoveryHandler(reg *registry.Registry, m *matcher.Matcher, authClient *auth.Client) *DiscoveryHandler {
	return &DiscoveryHandler{
		registry:   reg,
		matcher:    m,
		authClient: authClient,
	}
}

func (h *DiscoveryHandler) RegisterPeer(ctx context.Context, req *discoveryv1.RegisterPeerRequest) (*discoveryv1.RegisterPeerResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}
	if req.IpAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "ip_address is required")
	}

	// Validate token with Identity Service
	nodeID, err := h.resolveNodeID(ctx, req.Token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "auth failed: %v", err)
	}

	peer := &registry.Peer{
		NodeID:     nodeID,
		PublicKey:  "",
		Endpoint:   req.IpAddress + ":" + fmt.Sprint(req.Port),
		Version:    "1.0.0",
		PeerID:     req.PeerId,
		P2PAddrs:   req.P2PAddrs,
		IsExitNode: req.IsExitNode,
	}

	if err := h.registry.Register(ctx, peer); err != nil {
		return nil, status.Errorf(codes.Internal, "register failed: %v", err)
	}

	return &discoveryv1.RegisterPeerResponse{
		Success: true,
		Message: "peer registered",
	}, nil
}

func (h *DiscoveryHandler) ListPeers(ctx context.Context, req *discoveryv1.ListPeersRequest) (*discoveryv1.ListPeersResponse, error) {
	token := tokenFromMetadata(ctx)
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "authorization token required")
	}
	if _, err := h.resolveNodeID(ctx, token); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "auth failed: %v", err)
	}

	peers, err := h.registry.ListOnlinePeers(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list peers failed: %v", err)
	}

	var protoPeers []*discoveryv1.Peer
	for _, p := range peers {
		// Extract IP and Port from Endpoint (e.g., "10.0.0.5:51820")
		var ipAddress string
		var port int32
		parts := strings.Split(p.Endpoint, ":")
		if len(parts) == 2 {
			ipAddress = parts[0]
			if parsedPort, err := strconv.Atoi(parts[1]); err == nil {
				port = int32(parsedPort)
			}
		}

		protoPeers = append(protoPeers, &discoveryv1.Peer{
			NodeId:     p.NodeID,
			IpAddress:  ipAddress,
			Port:       port,
			Region:     p.Version,
			PeerId:     p.PeerID,
			P2PAddrs:   p.P2PAddrs,
			IsExitNode: p.IsExitNode,
			LastSeen:   p.LastSeen,
		})
	}

	// Apply limit
	limit := int(req.Limit)
	if limit <= 0 || limit > len(protoPeers) {
		limit = len(protoPeers)
	}

	return &discoveryv1.ListPeersResponse{
		Peers: protoPeers[:limit],
	}, nil
}

// tokenFromMetadata extracts the JWT token from gRPC metadata "authorization" key.
func tokenFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get("authorization")
	if len(vals) == 0 {
		return ""
	}
	return strings.TrimPrefix(vals[0], "Bearer ")
}

func (h *DiscoveryHandler) Heartbeat(ctx context.Context, req *discoveryv1.HeartbeatRequest) (*discoveryv1.HeartbeatResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	// Validate token with Identity Service
	nodeID, err := h.resolveNodeID(ctx, req.Token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "auth failed: %v", err)
	}

	if err := h.registry.Heartbeat(ctx, nodeID); err != nil {
		return nil, status.Errorf(codes.NotFound, "heartbeat failed: %v", err)
	}

	return &discoveryv1.HeartbeatResponse{
		Success: true,
		Ttl:     30,
	}, nil
}

// resolveNodeID validates token via Identity Service or falls back to token prefix
func (h *DiscoveryHandler) resolveNodeID(ctx context.Context, token string) (string, error) {
	if h.authClient != nil {
		return h.authClient.ValidateToken(ctx, token)
	}
	// Fallback: no auth client configured (dev mode)
	if len(token) < 8 {
		return "", fmt.Errorf("token too short")
	}
	return token[:8], nil
}
