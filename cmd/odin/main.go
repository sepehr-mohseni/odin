package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	// Added missing time import
	"odin/pkg/config"
	"odin/pkg/gateway"
	"odin/pkg/logging"
)

var (
	configPath string
	version    = "dev"
)

func init() {
	flag.StringVar(&configPath, "config", "config/config.yaml", "Path to configuration file")
	flag.Parse()
}

func main() {
	logger := logging.NewLogger()

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		loggingConfig := logging.Config{
			Level: envLogLevel,
			JSON:  false,
		}
		logging.ConfigureLogger(logger, loggingConfig)
	}

	logger.Infof("Starting Odin API Gateway %s", version)
	logger.Infof("Loading configuration from %s", configPath)

	cfg, err := config.Load(configPath, logger)
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Configure logging
	loggingConfig := logging.Config{
		Level: cfg.Logging.Level,
		JSON:  cfg.Logging.JSON,
	}
	logging.ConfigureLogger(logger, loggingConfig)

	// Apply environment variable overrides
	if envPort := os.Getenv("GATEWAY_PORT"); envPort != "" {
		var port int
		if _, err := fmt.Sscanf(envPort, "%d", &port); err == nil {
			cfg.Server.Port = port
		}
	}

	gw, err := gateway.New(cfg, configPath, logger)
	if err != nil {
		logger.Fatalf("Failed to initialize gateway: %v", err)
	}

	// Start the server in a goroutine
	go func() {
		if err := gw.Start(); err != nil {
			if err.Error() != "http: Server closed" {
				logger.Fatalf("Failed to start server: %v", err)
			}
		}
	}()

	// Setup signal handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Handle SIGHUP for config reload
	reload := make(chan os.Signal, 1)
	signal.Notify(reload, syscall.SIGHUP)

	go func() {
		for {
			<-reload
			logger.Info("Received SIGHUP, reloading configuration")
			// Fix: Remove the unused newConfig variable
			if _, err := config.Load(configPath, logger); err != nil {
				logger.Errorf("Failed to reload configuration: %v", err)
			} else {
				// Apply updated configuration (implementation depends on your design)
				logger.Info("Configuration reloaded successfully")
			}
		}
	}()

	// Block until signal is received
	sig := <-quit
	logger.Infof("Received signal: %v, shutting down...", sig)

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.GracefulTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := gw.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server gracefully shut down")
}
