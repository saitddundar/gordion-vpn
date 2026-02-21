package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	pkglogger "github.com/saitddundar/gordion-vpn/pkg/logger"
	discoveryv1 "github.com/saitddundar/gordion-vpn/pkg/proto/discovery/v1"
	"github.com/saitddundar/gordion-vpn/services/discovery/internal/config"
	grpchandler "github.com/saitddundar/gordion-vpn/services/discovery/internal/grpc"
	"github.com/saitddundar/gordion-vpn/services/discovery/internal/matcher"
	"github.com/saitddundar/gordion-vpn/services/discovery/internal/registry"
)

func main() {
	cfg, err := config.Load("../../configs/discovery.dev.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := pkglogger.New(cfg.LogLevel)
	logger.Info("Starting Discovery Service...")

	logger.Infof("Connecting to Redis: %s", cfg.RedisURL)
	reg, err := registry.New(cfg.RedisURL, cfg.HeartbeatTTL)
	if err != nil {
		logger.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer reg.Close()
	logger.Info("Redis connection established")

	m := matcher.New(reg)

	handler := grpchandler.NewDiscoveryHandler(reg, m)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpchandler.MetricsInterceptor("discovery")),
	)
	discoveryv1.RegisterDiscoveryServiceServer(grpcServer, handler)
	reflection.Register(grpcServer)

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	logger.Infof("gRPC server listening on %s", addr)

	// Metrics endpoint
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		logger.Info("Metrics endpoint listening on :9091")
		if err := http.ListenAndServe(":9091", nil); err != nil {
			logger.Fatalf("Failed to start metrics server: %v", err)
		}
	}()

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			logger.Fatalf("Failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down...")
	grpcServer.GracefulStop()
	logger.Info("Server stopped")
}
