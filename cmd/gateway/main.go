package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"odin/pkg/config"
	"odin/pkg/gateway"
	"odin/pkg/logging"

	"github.com/sirupsen/logrus"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "Path to configuration file")
	flag.Parse()

	log := logging.NewLogger()
	log.Info("Starting API Gateway")

	cfg, err := config.Load(*configPath, log)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Configure logging
	loggingConfig := logging.Config{
		Level: cfg.Logging.Level,
		JSON:  cfg.Logging.JSON,
	}
	logging.ConfigureLogger(log, loggingConfig)

	for _, svc := range cfg.Services {
		log.WithFields(logrus.Fields{
			"name":           svc.Name,
			"base_path":      svc.BasePath,
			"targets":        svc.Targets,
			"authentication": svc.Authentication,
		}).Info("Loaded service configuration")

		if svc.Aggregation != nil {
			for _, dep := range svc.Aggregation.Dependencies {
				log.WithFields(logrus.Fields{
					"service":    svc.Name,
					"depends_on": dep.Service,
					"path":       dep.Path,
				}).Info("Service has dependency")
			}
		}
	}

	gw, err := gateway.New(cfg, *configPath, log)
	if err != nil {
		log.Fatalf("Failed to initialize gateway: %v", err)
	}

	go func() {
		if err := gw.Start(); err != nil {
			if err.Error() != "http: Server closed" {
				log.Errorf("Server error: %v", err)
			}
		}
	}()

	log.Infof("API Gateway started on port %d", cfg.Server.Port)
	log.Infof("Admin interface available at http://localhost:%d/admin", cfg.Server.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.GracefulTimeout)
	defer cancel()

	if err := gw.Shutdown(ctx); err != nil {
		log.Errorf("Server forced to shutdown: %v", err)
	}

	log.Info("Server stopped")
}
