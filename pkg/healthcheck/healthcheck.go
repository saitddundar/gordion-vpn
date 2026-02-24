package healthcheck

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type Server struct {
	grpc_health_v1.UnimplementedHealthServer
	serviceName string
	checker     func() bool
}

func New(serviceName string, checker func() bool) *Server {
	return &Server{
		serviceName: serviceName,
		checker:     checker,
	}
}

// implements the gRPC health check RPC
func (s *Server) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	// If a specific service is requested, match it
	if req.Service != "" && req.Service != s.serviceName {
		return nil, status.Error(codes.NotFound, "unknown service")
	}

	if s.checker != nil && !s.checker() {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		}, nil
	}

	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

// adds health check to a gRPC server
func Register(server *grpc.Server, serviceName string, checker func() bool) {
	hs := New(serviceName, checker)
	grpc_health_v1.RegisterHealthServer(server, hs)
}
