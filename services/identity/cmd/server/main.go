package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pkglogger "github.com/saitddundar/gordion-vpn/pkg/logger"
	identityv1 "github.com/saitddundar/gordion-vpn/pkg/proto/identity/v1"
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

	identityService := service.New(store, cfg.JWTSecret, cfg.TokenDuration)
	logger.Info("Identity service initialized")

	handler := grpchandler.NewIdentityHandler(identityService)

	grpcServer := grpc.NewServer()
	identityv1.RegisterIdentityServiceServer(grpcServer, handler)

	reflection.Register(grpcServer)

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	logger.Infof("gRPC server listening on %s", addr)

	// Start server in goroutine
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			logger.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down gracefully...")

	// Cleanup expired tokens before shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := identityService.CleanupExpiredTokens(ctx); err != nil {
		logger.Warnf("Failed to cleanup expired tokens: %v", err)
	}

	// Stop gRPC server
	grpcServer.GracefulStop()
	logger.Info("Server stopped")
}
