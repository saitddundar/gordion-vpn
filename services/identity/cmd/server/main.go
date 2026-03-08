package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/saitddundar/gordion-vpn/pkg/healthcheck"
	pkglogger "github.com/saitddundar/gordion-vpn/pkg/logger"
	"github.com/saitddundar/gordion-vpn/pkg/middleware"
	identityv1 "github.com/saitddundar/gordion-vpn/pkg/proto/identity/v1"
	"github.com/saitddundar/gordion-vpn/pkg/ratelimit"
	"github.com/saitddundar/gordion-vpn/pkg/tlsutil"
	"github.com/saitddundar/gordion-vpn/pkg/tracing"
	"github.com/saitddundar/gordion-vpn/services/identity/internal/config"
	grpchandler "github.com/saitddundar/gordion-vpn/services/identity/internal/grpc"
	"github.com/saitddundar/gordion-vpn/services/identity/internal/service"
	"github.com/saitddundar/gordion-vpn/services/identity/internal/storage"
)

func main() {
	// Load config
	cfg, err := config.LoadFromEnv("../../configs/identity.dev.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := pkglogger.New(cfg.LogLevel)
	logger.Info("Starting Identity Service...")

	logger.Infof("Connecting to database: %s", cfg.DatabaseURL)
	store, err := storage.New(cfg.DatabaseURL)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer store.Close()
	logger.Info("Database connection established")

	identityService := service.New(store, cfg.JWTSecret, cfg.NetworkSecret, cfg.TokenDuration)
	logger.Info("Identity service initialized")

	handler := grpchandler.NewIdentityHandler(identityService)

	limiter := ratelimit.New(100, time.Minute)

	serverOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			ratelimit.UnaryInterceptor(limiter),
			tracing.ServerInterceptor(logger, "identity"),
			middleware.LoggingInterceptor(logger),
			grpchandler.MetricsInterceptor("identity"),
		),
	}

	// Load TLS if cert files exist
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
	identityv1.RegisterIdentityServiceServer(grpcServer, handler)
	healthcheck.Register(grpcServer, "identity", func() bool {
		return store.Ping() == nil
	})
	reflection.Register(grpcServer)

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	logger.Infof("gRPC server listening on %s", addr)

	// Start Prometheus metrics endpoint
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		logger.Info("Metrics endpoint listening on :9090")
		if err := http.ListenAndServe(":9090", nil); err != nil {
			logger.Fatalf("Failed to start metrics server: %v", err)
		}
	}()

	// Start gRPC server in goroutine
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			logger.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down gracefully (10s timeout)...")

	// Cleanup expired tokens before shutdown
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cleanupCancel()

	if err := identityService.CleanupExpiredTokens(cleanupCtx); err != nil {
		logger.Warnf("Failed to cleanup expired tokens: %v", err)
	}

	// Stop gRPC server with timeout
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	timer := time.NewTimer(10 * time.Second)
	select {
	case <-stopped:
		logger.Info("Server stopped gracefully")
	case <-timer.C:
		logger.Warn("Graceful shutdown timed out, forcing stop")
		grpcServer.Stop()
	}

}
