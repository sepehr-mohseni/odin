package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/sirupsen/logrus"
)

// Config represents tracing configuration
type Config struct {
	Enabled        bool    `yaml:"enabled"`
	ServiceName    string  `yaml:"serviceName"`
	ServiceVersion string  `yaml:"serviceVersion"`
	Environment    string  `yaml:"environment"`
	Endpoint       string  `yaml:"endpoint"`
	SampleRate     float64 `yaml:"sampleRate"`
	Insecure       bool    `yaml:"insecure"`
}

// SetDefaults sets default values for tracing configuration
func (c *Config) SetDefaults() {
	if c.ServiceName == "" {
		c.ServiceName = "odin-gateway"
	}
	if c.ServiceVersion == "" {
		c.ServiceVersion = "1.0.0"
	}
	if c.Environment == "" {
		c.Environment = "development"
	}
	if c.Endpoint == "" {
		c.Endpoint = "http://localhost:4318/v1/traces"
	}
	if c.SampleRate == 0 {
		c.SampleRate = 1.0 // 100% sampling by default
	}
}

// Manager handles distributed tracing operations
type Manager struct {
	tracer   oteltrace.Tracer
	provider *trace.TracerProvider
	config   Config
	logger   *logrus.Logger
}

// NewManager creates a new tracing manager
func NewManager(config Config, logger *logrus.Logger) (*Manager, error) {
	if !config.Enabled {
		return &Manager{
			tracer: otel.Tracer("noop"),
			config: config,
			logger: logger,
		}, nil
	}

	config.SetDefaults()

	// Create OTLP exporter
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(config.Endpoint),
	}
	if config.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(config.SampleRate)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(provider)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := provider.Tracer(config.ServiceName)

	logger.WithFields(logrus.Fields{
		"service_name":    config.ServiceName,
		"service_version": config.ServiceVersion,
		"environment":     config.Environment,
		"endpoint":        config.Endpoint,
		"sample_rate":     config.SampleRate,
	}).Info("Distributed tracing initialized")

	return &Manager{
		tracer:   tracer,
		provider: provider,
		config:   config,
		logger:   logger,
	}, nil
}

// StartSpan starts a new span with the given name and options
func (m *Manager) StartSpan(ctx context.Context, spanName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	if !m.config.Enabled {
		return ctx, oteltrace.SpanFromContext(ctx)
	}
	return m.tracer.Start(ctx, spanName, opts...)
}

// StartHTTPSpan starts a span for HTTP requests
func (m *Manager) StartHTTPSpan(ctx context.Context, method, path, service string) (context.Context, oteltrace.Span) {
	spanName := fmt.Sprintf("%s %s", method, path)
	ctx, span := m.StartSpan(ctx, spanName,
		oteltrace.WithAttributes(
			semconv.HTTPMethodKey.String(method),
			semconv.HTTPRouteKey.String(path),
			attribute.String("service.name", service),
			attribute.String("component", "http"),
		),
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
	)
	return ctx, span
}

// StartClientSpan starts a span for client requests (outgoing)
func (m *Manager) StartClientSpan(ctx context.Context, operation, target string) (context.Context, oteltrace.Span) {
	ctx, span := m.StartSpan(ctx, operation,
		oteltrace.WithAttributes(
			attribute.String("target", target),
			attribute.String("component", "client"),
		),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
	)
	return ctx, span
}

// StartPluginSpan starts a span for plugin operations
func (m *Manager) StartPluginSpan(ctx context.Context, pluginName, hookType string) (context.Context, oteltrace.Span) {
	spanName := fmt.Sprintf("plugin.%s.%s", pluginName, hookType)
	ctx, span := m.StartSpan(ctx, spanName,
		oteltrace.WithAttributes(
			attribute.String("plugin.name", pluginName),
			attribute.String("plugin.hook", hookType),
			attribute.String("component", "plugin"),
		),
		oteltrace.WithSpanKind(oteltrace.SpanKindInternal),
	)
	return ctx, span
}

// AddSpanEvent adds an event to the current span
func (m *Manager) AddSpanEvent(span oteltrace.Span, name string, attrs ...attribute.KeyValue) {
	if !m.config.Enabled {
		return
	}
	span.AddEvent(name, oteltrace.WithAttributes(attrs...))
}

// SetSpanAttributes sets attributes on a span
func (m *Manager) SetSpanAttributes(span oteltrace.Span, attrs ...attribute.KeyValue) {
	if !m.config.Enabled {
		return
	}
	span.SetAttributes(attrs...)
}

// RecordError records an error on a span
func (m *Manager) RecordError(span oteltrace.Span, err error, attrs ...attribute.KeyValue) {
	if !m.config.Enabled {
		return
	}
	span.RecordError(err, oteltrace.WithAttributes(attrs...))
	span.SetStatus(codes.Error, err.Error())
}

// FinishSpan finishes a span with optional attributes
func (m *Manager) FinishSpan(span oteltrace.Span, err error, attrs ...attribute.KeyValue) {
	if !m.config.Enabled {
		return
	}

	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}

	if err != nil {
		m.RecordError(span, err)
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.End()
}

// Shutdown shuts down the tracer provider
func (m *Manager) Shutdown(ctx context.Context) error {
	if !m.config.Enabled || m.provider == nil {
		return nil
	}

	// Allow time for final spans to be exported
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := m.provider.Shutdown(shutdownCtx); err != nil {
		m.logger.WithError(err).Error("Failed to shutdown tracing provider")
		return err
	}

	m.logger.Info("Tracing provider shut down successfully")
	return nil
}

// GetTracer returns the underlying tracer
func (m *Manager) GetTracer() oteltrace.Tracer {
	return m.tracer
}

// InjectHeaders injects tracing headers into a map
func (m *Manager) InjectHeaders(ctx context.Context, headers map[string]string) {
	if !m.config.Enabled {
		return
	}
	otel.GetTextMapPropagator().Inject(ctx, &headerCarrier{headers: headers})
}

// ExtractHeaders extracts tracing context from headers
func (m *Manager) ExtractHeaders(ctx context.Context, headers map[string][]string) context.Context {
	if !m.config.Enabled {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, &multiHeaderCarrier{headers: headers})
}

// headerCarrier implements TextMapCarrier for single-value headers
type headerCarrier struct {
	headers map[string]string
}

func (c *headerCarrier) Get(key string) string {
	return c.headers[key]
}

func (c *headerCarrier) Set(key, value string) {
	c.headers[key] = value
}

func (c *headerCarrier) Keys() []string {
	keys := make([]string, 0, len(c.headers))
	for k := range c.headers {
		keys = append(keys, k)
	}
	return keys
}

// multiHeaderCarrier implements TextMapCarrier for multi-value headers
type multiHeaderCarrier struct {
	headers map[string][]string
}

func (c *multiHeaderCarrier) Get(key string) string {
	values := c.headers[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (c *multiHeaderCarrier) Set(key, value string) {
	c.headers[key] = []string{value}
}

func (c *multiHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c.headers))
	for k := range c.headers {
		keys = append(keys, k)
	}
	return keys
}
