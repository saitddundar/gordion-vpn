package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/saitddundar/gordion-vpn/pkg/metrics"
)

func MetricsInterceptor(serviceName string) grpc.UnaryServerInterceptor {

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		metrics.ActiveConnections.WithLabelValues(serviceName).Inc()
		defer metrics.ActiveConnections.WithLabelValues(serviceName).Dec()

		resp, err := handler(ctx, req)

		duration := time.Since(start).Seconds()
		statusLabel := "success"
		if err != nil {
			statusLabel = status.Code(err).String()
		}
		metrics.RequestsTotal.WithLabelValues(serviceName, info.FullMethod, statusLabel).Inc()
		metrics.RequestDuration.WithLabelValues(serviceName, info.FullMethod).Observe(duration)
		return resp, err
	}
}
