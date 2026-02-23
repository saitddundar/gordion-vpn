package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pkglogger "github.com/saitddundar/gordion-vpn/pkg/logger"
)

type contextKey string

const TraceIDKey contextKey = "trace_id"
const metadataKey = "x-trace-id"

func generateTraceID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func TraceIDFromContext(ctx context.Context) string {
	id, ok := ctx.Value(TraceIDKey).(string)
	if !ok {
		return ""
	}
	return id
}

// extracts or creates trace_id for incoming requests
func ServerInterceptor(logger pkglogger.Logger, serviceName string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Try to extract trace_id from incoming metadata
		traceID := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get(metadataKey); len(vals) > 0 {
				traceID = vals[0]
			}
		}

		// Generate new trace_id if not found
		if traceID == "" {
			traceID = generateTraceID()
		}

		ctx = context.WithValue(ctx, TraceIDKey, traceID)

		logger.Infof("[%s] [%s] --> %s", traceID, serviceName, info.FullMethod)

		resp, err := handler(ctx, req)

		if err != nil {
			logger.Errorf("[%s] [%s] <-- %s | ERROR: %v", traceID, serviceName, info.FullMethod, err)
		} else {
			logger.Infof("[%s] [%s] <-- %s | OK", traceID, serviceName, info.FullMethod)
		}

		return resp, err
	}
}

// propagates trace_id to outgoing requests
func ClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		traceID, ok := ctx.Value(TraceIDKey).(string)
		if ok && traceID != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, metadataKey, traceID)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
