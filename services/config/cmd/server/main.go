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

	"github.com/saitddundar/gordion-vpn/pkg/auth"
	"github.com/saitddundar/gordion-vpn/pkg/healthcheck"
	pkglogger "github.com/saitddundar/gordion-vpn/pkg/logger"
	"github.com/saitddundar/gordion-vpn/pkg/middleware"
	configv1 "github.com/saitddundar/gordion-vpn/pkg/proto/config/v1"
	"github.com/saitddundar/gordion-vpn/pkg/ratelimit"
	"github.com/saitddundar/gordion-vpn/pkg/tlsutil"
	"github.com/saitddundar/gordion-vpn/pkg/tracing"
	"github.com/saitddundar/gordion-vpn/services/config/internal/allocator"
	"github.com/saitddundar/gordion-vpn/services/config/internal/config"
	grpchandler "github.com/saitddundar/gordion-vpn/services/config/internal/grpc"
)

const configPath = "../../configs/config.dev.yaml"

func main() {
	cfg, err := config.Load(configPath)
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

	handler := grpchandler.NewConfigHandler(alloc, authClient, cfg.NetworkCIDR, cfg.MTU, cfg.DNSServers)

	limiter := ratelimit.New(100, time.Minute)

	serverOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			ratelimit.UnaryInterceptor(limiter),
			tracing.ServerInterceptor(logger, "config"),
			middleware.LoggingInterceptor(logger),
			grpchandler.MetricsInterceptor("config"),
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
	configv1.RegisterConfigServiceServer(grpcServer, handler)
	healthcheck.Register(grpcServer, "config", func() bool {
		return alloc.Ping() == nil
	})
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

	// SIGHUP → reload config, SIGINT/SIGTERM → shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for sig := range sigCh {
		if sig == syscall.SIGHUP {
			logger.Info("Received SIGHUP, reloading config...")
			newCfg, err := config.Load(configPath)
			if err != nil {
				logger.Errorf("Config reload failed: %v", err)
				continue
			}
			handler.ReloadConfig(newCfg.NetworkCIDR, newCfg.MTU, newCfg.DNSServers)
			logger.Infof("Config reloaded (version bumped)")
			continue
		}

		// SIGINT or SIGTERM → graceful shutdown with timeout
		logger.Info("Shutting down gracefully (10s timeout)...")
		stopped := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(stopped)
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		select {
		case <-stopped:
			logger.Info("Server stopped gracefully")
		case <-ctx.Done():
			logger.Warn("Graceful shutdown timed out, forcing stop")
			grpcServer.Stop()
		}
		cancel()
		return
	}
}
