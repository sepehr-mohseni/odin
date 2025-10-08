package main

import (
	"context"
	"time"

	"odin/pkg/plugins"

	"github.com/sirupsen/logrus"
)

// RequestLoggerPlugin logs incoming requests
type RequestLoggerPlugin struct {
	logger         *logrus.Logger
	logLevel       string
	includeHeaders bool
}

// Ensure our plugin implements the interface at compile time
var _ plugins.Plugin = (*RequestLoggerPlugin)(nil)

// Plugin instance that will be exported
var Plugin = &RequestLoggerPlugin{}

func (p *RequestLoggerPlugin) Name() string {
	return "request_logger"
}

func (p *RequestLoggerPlugin) Version() string {
	return "1.0.0"
}

func (p *RequestLoggerPlugin) Initialize(config map[string]interface{}) error {
	p.logger = logrus.New()

	if level, ok := config["log_level"].(string); ok {
		if lvl, err := logrus.ParseLevel(level); err == nil {
			p.logger.SetLevel(lvl)
		}
		p.logLevel = level
	} else {
		p.logLevel = "info"
	}

	if include, ok := config["include_headers"].(bool); ok {
		p.includeHeaders = include
	}

	p.logger.WithFields(logrus.Fields{
		"plugin":          p.Name(),
		"version":         p.Version(),
		"log_level":       p.logLevel,
		"include_headers": p.includeHeaders,
	}).Info("Request Logger Plugin initialized")

	return nil
}

func (p *RequestLoggerPlugin) PreRequest(ctx context.Context, pluginCtx *plugins.PluginContext) error {
	fields := logrus.Fields{
		"request_id": pluginCtx.RequestID,
		"method":     pluginCtx.Method,
		"path":       pluginCtx.Path,
		"user_id":    pluginCtx.UserID,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	if p.includeHeaders && pluginCtx.Headers != nil {
		safeHeaders := make(map[string][]string)
		for key, values := range pluginCtx.Headers {
			if key != "Authorization" && key != "Cookie" && key != "X-Api-Key" {
				safeHeaders[key] = values
			}
		}
		fields["headers"] = safeHeaders
	}

	pluginCtx.Metadata["request_start_time"] = time.Now()
	pluginCtx.Metadata["plugin_logger"] = p.Name()

	p.logger.WithFields(fields).Info("Incoming request")
	return nil
}

func (p *RequestLoggerPlugin) PostRequest(ctx context.Context, pluginCtx *plugins.PluginContext) error {
	return nil
}

func (p *RequestLoggerPlugin) PreResponse(ctx context.Context, pluginCtx *plugins.PluginContext) error {
	return nil
}

func (p *RequestLoggerPlugin) PostResponse(ctx context.Context, pluginCtx *plugins.PluginContext) error {
	var duration time.Duration
	if startTime, ok := pluginCtx.Metadata["request_start_time"].(time.Time); ok {
		duration = time.Since(startTime)
	}

	fields := logrus.Fields{
		"request_id":  pluginCtx.RequestID,
		"method":      pluginCtx.Method,
		"path":        pluginCtx.Path,
		"user_id":     pluginCtx.UserID,
		"duration":    duration.String(),
		"duration_ms": duration.Nanoseconds() / 1000000,
	}

	p.logger.WithFields(fields).Info("Request completed")
	return nil
}

func (p *RequestLoggerPlugin) Cleanup() error {
	p.logger.WithFields(logrus.Fields{
		"plugin":  p.Name(),
		"version": p.Version(),
	}).Info("Request Logger Plugin cleanup")
	return nil
}

func main() {}
