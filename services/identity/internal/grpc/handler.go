package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identityv1 "github.com/saitddundar/gordion-vpn/pkg/proto/identity/v1"
	"github.com/saitddundar/gordion-vpn/services/identity/internal/service"
)

// implements the identity gRPC service
type IdentityHandler struct {
	identityv1.UnimplementedIdentityServiceServer
	service *service.IdentityService
}

func NewIdentityHandler(svc *service.IdentityService) *IdentityHandler {
	return &IdentityHandler{
		service: svc,
	}
}

func (h *IdentityHandler) RegisterNode(ctx context.Context, req *identityv1.RegisterNodeRequest) (*identityv1.RegisterNodeResponse, error) {

	if req.PublicKey == "" {
		return nil, status.Error(codes.InvalidArgument, "public_key is required")
	}
	if req.Version == "" {
		return nil, status.Error(codes.InvalidArgument, "version is required")
	}
	nodeID, token, expiresAt, err := h.service.RegisterNode(ctx, req.PublicKey, req.Version, req.PeerId, req.NetworkSecret)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register node: %v", err)
	}
	return &identityv1.RegisterNodeResponse{
		NodeId:    nodeID,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (h *IdentityHandler) ValidateToken(ctx context.Context, req *identityv1.ValidateTokenRequest) (*identityv1.ValidateTokenResponse, error) {

	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}
	nodeID, valid, err := h.service.ValidateToken(ctx, req.Token)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to validate token: %v", err)
	}
	return &identityv1.ValidateTokenResponse{
		Valid:  valid,
		NodeId: nodeID,
	}, nil
}

func (h *IdentityHandler) GetPublicKey(ctx context.Context, req *identityv1.GetPublicKeyRequest) (*identityv1.GetPublicKeyResponse, error) {

	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}
	publicKey, err := h.service.GetPublicKey(ctx, req.NodeId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "node not found: %v", err)
	}
	return &identityv1.GetPublicKeyResponse{
		PublicKey: publicKey,
	}, nil
}
