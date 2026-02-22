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
	"github.com/saitddundar/gordion-vpn/pkg/auth"
	discoveryv1 "github.com/saitddundar/gordion-vpn/pkg/proto/discovery/v1"
	"github.com/saitddundar/gordion-vpn/pkg/tlsutil"
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

	// Connect to Identity Service for token validation
	var authClient *auth.Client
	identityAddr := os.Getenv("IDENTITY_ADDR")
	if identityAddr == "" {
		identityAddr = "localhost:8001"
	}
	authClient, err = auth.NewClient(identityAddr, "")
	if err != nil {
		logger.Warnf("Auth client disabled: %v", err)
	} else {
		defer authClient.Close()
		logger.Infof("Auth client connected to Identity Service at %s", identityAddr)
	}

	handler := grpchandler.NewDiscoveryHandler(reg, m, authClient)
	
	// Create gRPC server with metrics interceptor and optional TLS
	serverOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(grpchandler.MetricsInterceptor("discovery")),
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
