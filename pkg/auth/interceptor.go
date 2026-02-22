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

func AuthInterceptor(authClient *Client) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		token := extractToken(ctx)
		if token == "" {
			return handler(ctx, req)
		}

		nodeID, err := authClient.ValidateToken(ctx, token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
		}

		ctx = context.WithValue(ctx, NodeIDKey, nodeID)
		return handler(ctx, req)
	}
}

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

func NodeIDFromContext(ctx context.Context) string {
	nodeID, ok := ctx.Value(NodeIDKey).(string)
	if !ok {
		return ""
	}
	return nodeID
}
