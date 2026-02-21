package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	configv1 "github.com/saitddundar/gordion-vpn/pkg/proto/config/v1"
	"github.com/saitddundar/gordion-vpn/services/config/internal/allocator"
)

type ConfigHandler struct {
	configv1.UnimplementedConfigServiceServer
	allocator   *allocator.Allocator
	networkCIDR string
	mtu         int32
	dnsServers  []string
}

func NewConfigHandler(alloc *allocator.Allocator, networkCIDR string, mtu int, dnsServers []string) *ConfigHandler {
	return &ConfigHandler{
		allocator:   alloc,
		networkCIDR: networkCIDR,
		mtu:         int32(mtu),
		dnsServers:  dnsServers,
	}
}

func (h *ConfigHandler) GetConfig(ctx context.Context, req *configv1.GetConfigRequest) (*configv1.GetConfigResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	// TODO: validate token with identity service

	return &configv1.GetConfigResponse{
		NetworkCidr: h.networkCIDR,
		Mtu:         h.mtu,
		DnsServers:  h.dnsServers,
	}, nil
}

func (h *ConfigHandler) RequestIP(ctx context.Context, req *configv1.RequestIPRequest) (*configv1.RequestIPResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}
	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	ip, err := h.allocator.AllocateIP(ctx, req.NodeId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "IP allocation failed: %v", err)
	}

	return &configv1.RequestIPResponse{
		IpAddress:  ip,
		SubnetMask: h.allocator.SubnetMask(),
		Gateway:    h.allocator.Gateway(),
	}, nil
}

func (h *ConfigHandler) ReleaseIP(ctx context.Context, req *configv1.ReleaseIPRequest) (*configv1.ReleaseIPResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}
	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	if err := h.allocator.ReleaseIP(ctx, req.NodeId, req.IpAddress); err != nil {
		return nil, status.Errorf(codes.Internal, "IP release failed: %v", err)
	}

	return &configv1.ReleaseIPResponse{
		Success: true,
		Message: "IP released",
	}, nil
}
