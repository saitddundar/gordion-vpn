package grpc

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/saitddundar/gordion-vpn/pkg/auth"
	configv1 "github.com/saitddundar/gordion-vpn/pkg/proto/config/v1"
	"github.com/saitddundar/gordion-vpn/services/config/internal/allocator"
)

type ConfigHandler struct {
	configv1.UnimplementedConfigServiceServer
	allocator  *allocator.Allocator
	authClient *auth.Client

	// cfgMu protects hot-reloadable config fields below
	cfgMu         sync.RWMutex
	networkCIDR   string
	mtu           int32
	dnsServers    []string
	configVersion int32
}

func NewConfigHandler(alloc *allocator.Allocator, authClient *auth.Client, networkCIDR string, mtu int, dnsServers []string) *ConfigHandler {
	return &ConfigHandler{
		allocator:     alloc,
		authClient:    authClient,
		networkCIDR:   networkCIDR,
		mtu:           int32(mtu),
		dnsServers:    dnsServers,
		configVersion: 1,
	}
}

func (h *ConfigHandler) GetConfig(ctx context.Context, req *configv1.GetConfigRequest) (*configv1.GetConfigResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	// Validate token with Identity Service
	if _, err := h.resolveNodeID(ctx, req.Token); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "auth failed: %v", err)
	}

	h.cfgMu.RLock()
	version := h.configVersion
	cidr := h.networkCIDR
	mtu := h.mtu
	dns := h.dnsServers
	h.cfgMu.RUnlock()

	// Version check: if client already has latest, skip sending full config
	if req.ConfigVersion > 0 && req.ConfigVersion >= version {
		return &configv1.GetConfigResponse{
			ConfigVersion: version,
			UpToDate:      true,
		}, nil
	}

	return &configv1.GetConfigResponse{
		NetworkCidr:   cidr,
		Mtu:           mtu,
		DnsServers:    dns,
		ConfigVersion: version,
		UpToDate:      false,
	}, nil
}

// hot-swaps network config (called on SIGHUP)
func (h *ConfigHandler) ReloadConfig(networkCIDR string, mtu int, dnsServers []string) {
	h.cfgMu.Lock()
	defer h.cfgMu.Unlock()
	h.networkCIDR = networkCIDR
	h.mtu = int32(mtu)
	h.dnsServers = dnsServers
	h.configVersion++
}

func (h *ConfigHandler) RequestIP(ctx context.Context, req *configv1.RequestIPRequest) (*configv1.RequestIPResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}
	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	// Validate token with Identity Service
	if _, err := h.resolveNodeID(ctx, req.Token); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "auth failed: %v", err)
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

	// Validate token with Identity Service
	if _, err := h.resolveNodeID(ctx, req.Token); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "auth failed: %v", err)
	}

	if err := h.allocator.ReleaseIP(ctx, req.NodeId, req.IpAddress); err != nil {
		return nil, status.Errorf(codes.Internal, "IP release failed: %v", err)
	}

	return &configv1.ReleaseIPResponse{
		Success: true,
		Message: "IP released",
	}, nil
}

// validates token via Identity Service or falls back
func (h *ConfigHandler) resolveNodeID(ctx context.Context, token string) (string, error) {
	if h.authClient != nil {
		return h.authClient.ValidateToken(ctx, token)
	}
	// Fallback: no auth client configured (dev mode)
	if len(token) < 8 {
		return "", fmt.Errorf("token too short")
	}
	return token[:8], nil
}
