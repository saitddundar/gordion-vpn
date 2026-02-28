package main

import (
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

	"github.com/saitddundar/gordion-vpn/pkg/auth"
	"github.com/saitddundar/gordion-vpn/pkg/healthcheck"
	pkglogger "github.com/saitddundar/gordion-vpn/pkg/logger"
	"github.com/saitddundar/gordion-vpn/pkg/middleware"
	discoveryv1 "github.com/saitddundar/gordion-vpn/pkg/proto/discovery/v1"
	"github.com/saitddundar/gordion-vpn/pkg/ratelimit"
	"github.com/saitddundar/gordion-vpn/pkg/tlsutil"
	"github.com/saitddundar/gordion-vpn/pkg/tracing"
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

	limiter := ratelimit.New(100, time.Minute)

	serverOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			ratelimit.UnaryInterceptor(limiter),
			tracing.ServerInterceptor(logger, "discovery"),
			middleware.LoggingInterceptor(logger),
			grpchandler.MetricsInterceptor("discovery"),
		),
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
	healthcheck.Register(grpcServer, "discovery", func() bool {
		return reg.Ping() == nil
	})
	reflection.Register(grpcServer)

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	logger.Infof("gRPC server listening on %s", addr)

	// Metrics endpoint (non-fatal if port already in use)
	go func() {
		metricsPort := os.Getenv("METRICS_PORT")
		if metricsPort == "" {
			metricsPort = "9091"
		}
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		logger.Infof("Metrics endpoint listening on :%s", metricsPort)
		if err := http.ListenAndServe(":"+metricsPort, mux); err != nil {
			logger.Warnf("Metrics server error (non-fatal): %v", err)
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

	logger.Info("Shutting down gracefully (10s timeout)...")
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
