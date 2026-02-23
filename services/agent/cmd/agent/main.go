package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	pkglogger "github.com/saitddundar/gordion-vpn/pkg/logger"
	"github.com/saitddundar/gordion-vpn/services/agent/internal/agent"
	"github.com/saitddundar/gordion-vpn/services/agent/internal/config"
)

func main() {
	cfg, err := config.Load("../../configs/agent.dev.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := pkglogger.New(cfg.LogLevel)
	logger.Info("Starting Gordion VPN Agent...")

	a, err := agent.New(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to create agent: %v", err)
	}

	ctx := context.Background()
	if err := a.Start(ctx); err != nil {
		logger.Fatalf("Agent start failed: %v", err)
	}

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.Stop()
}
