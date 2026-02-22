package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	pkglogger "github.com/saitddundar/gordion-vpn/pkg/logger"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

func generateRequestID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func LoggingInterceptor(logger pkglogger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		requestID := generateRequestID()
		ctx = context.WithValue(ctx, RequestIDKey, requestID)

		start := time.Now()

		logger.Infof("[%s] --> %s", requestID, info.FullMethod)

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		if err != nil {
			code := status.Code(err)
			logger.Errorf("[%s] <-- %s | %s | %v | %s", requestID, info.FullMethod, code, duration, err)
		} else {
			logger.Infof("[%s] <-- %s | OK | %v", requestID, info.FullMethod, duration)
		}

		return resp, err
	}
}

func RequestIDFromContext(ctx context.Context) string {
	id, ok := ctx.Value(RequestIDKey).(string)
	if !ok {
		return "unknown"
	}
	return id
}
