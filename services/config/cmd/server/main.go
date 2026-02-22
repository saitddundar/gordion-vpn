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
	configv1 "github.com/saitddundar/gordion-vpn/pkg/proto/config/v1"
	"github.com/saitddundar/gordion-vpn/pkg/tlsutil"
	"github.com/saitddundar/gordion-vpn/services/config/internal/allocator"
	"github.com/saitddundar/gordion-vpn/services/config/internal/config"
	grpchandler "github.com/saitddundar/gordion-vpn/services/config/internal/grpc"
)

func main() {
	cfg, err := config.Load("../../configs/config.dev.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := pkglogger.New(cfg.LogLevel)
	logger.Info("Starting Config Service...")

	logger.Infof("Connecting to Redis: %s", cfg.RedisURL)
	alloc, err := allocator.New(cfg.RedisURL, cfg.NetworkCIDR)
	if err != nil {
		logger.Fatalf("Failed to initialize allocator: %v", err)
	}
	defer alloc.Close()
	logger.Info("IP allocator initialized")

	handler := grpchandler.NewConfigHandler(alloc, cfg.NetworkCIDR, cfg.MTU, cfg.DNSServers)

	// Create gRPC server with metrics interceptor and optional TLS
	serverOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(grpchandler.MetricsInterceptor("config")),
	}

	certFile := os.Getenv("TLS_CERT")
	keyFile := os.Getenv("TLS_KEY")
	if certFile == "" {
		certFile = "../../certs/server-cert.pem"
	}
	if keyFile == "" {
		keyFile = "../../certs/server-key.pem"
	}

	if _, err := os.Stat(certFile); err == nil {
		creds, err := tlsutil.ServerCredentials(certFile, keyFile)
		if err != nil {
			logger.Fatalf("Failed to load TLS: %v", err)
		}
		serverOpts = append(serverOpts, grpc.Creds(creds))
		logger.Info("TLS enabled")
	} else {
		logger.Warn("TLS disabled (no certs found)")
	}

	grpcServer := grpc.NewServer(serverOpts...)
	configv1.RegisterConfigServiceServer(grpcServer, handler)
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
		logger.Info("Metrics endpoint listening on :9092")
		if err := http.ListenAndServe(":9092", nil); err != nil {
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
