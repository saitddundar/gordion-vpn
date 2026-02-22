package auth

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const NodeIDKey contextKey = "node_id"

// AuthInterceptor validates tokens on every gRPC request
func AuthInterceptor(authClient *Client) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract token from metadata
		token := extractToken(ctx)
		if token == "" {
			// Try to extract from request field (backward compatibility)
			return handler(ctx, req)
		}

		// Validate with Identity Service
		nodeID, err := authClient.ValidateToken(ctx, token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
		}

		// Add node_id to context
		ctx = context.WithValue(ctx, NodeIDKey, nodeID)

		return handler(ctx, req)
	}
}

// extractToken gets the token from gRPC metadata
func extractToken(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return ""
	}

	return tokens[0]
}

// NodeIDFromContext extracts node_id from context (set by interceptor)
func NodeIDFromContext(ctx context.Context) string {
	nodeID, ok := ctx.Value(NodeIDKey).(string)
	if !ok {
		return ""
	}
	return nodeID
}
