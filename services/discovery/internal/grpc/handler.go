package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	discoveryv1 "github.com/saitddundar/gordion-vpn/pkg/proto/discovery/v1"
	"github.com/saitddundar/gordion-vpn/services/discovery/internal/matcher"
	"github.com/saitddundar/gordion-vpn/services/discovery/internal/registry"
)

type DiscoveryHandler struct {
	discoveryv1.UnimplementedDiscoveryServiceServer
	registry *registry.Registry
	matcher  *matcher.Matcher
}

func NewDiscoveryHandler(reg *registry.Registry, m *matcher.Matcher) *DiscoveryHandler {
	return &DiscoveryHandler{
		registry: reg,
		matcher:  m,
	}
}

func (h *DiscoveryHandler) RegisterPeer(ctx context.Context, req *discoveryv1.RegisterPeerRequest) (*discoveryv1.RegisterPeerResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}
	if req.IpAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "ip_address is required")
	}

	// TODO: validate token with identity service

	peer := &registry.Peer{
		NodeID:    req.Token[:8], // temporary: token'dan ID çıkar (ilerde identity service'den alınacak)
		PublicKey: "",
		Endpoint:  req.IpAddress + ":" + fmt.Sprint(req.Port),
		Version:   "1.0.0",
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
	peers, err := h.registry.ListOnlinePeers(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list peers failed: %v", err)
	}

	var protoPeers []*discoveryv1.Peer
	for _, p := range peers {
		protoPeers = append(protoPeers, &discoveryv1.Peer{
			NodeId:   p.NodeID,
			LastSeen: p.LastSeen,
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

func (h *DiscoveryHandler) Heartbeat(ctx context.Context, req *discoveryv1.HeartbeatRequest) (*discoveryv1.HeartbeatResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	nodeID := req.Token[:8] // temporary

	if err := h.registry.Heartbeat(ctx, nodeID); err != nil {
		return nil, status.Errorf(codes.NotFound, "heartbeat failed: %v", err)
	}

	return &discoveryv1.HeartbeatResponse{
		Success: true,
		Ttl:     30,
	}, nil
}
