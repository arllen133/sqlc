package sqlc

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "github.com/arllen133/sqlc"
	meterName  = "github.com/arllen133/sqlc"
)

// Metrics holds the OpenTelemetry metric instruments
type Metrics struct {
	QueryCount    metric.Int64Counter
	QueryDuration metric.Float64Histogram
	QueryErrors   metric.Int64Counter
}

// ObservabilityConfig holds logging, tracing, and metrics configuration
type ObservabilityConfig struct {
	Logger             *slog.Logger
	Tracer             trace.Tracer
	Meter              metric.Meter
	Metrics            *Metrics
	SlowQueryThreshold time.Duration
	LogQueries         bool // Log all queries (debug mode)
}

// defaultConfig returns a config with no logging/tracing/metrics
func defaultObservabilityConfig() *ObservabilityConfig {
	return &ObservabilityConfig{
		Logger:             nil,
		Tracer:             nil,
		Meter:              nil,
		Metrics:            nil,
		SlowQueryThreshold: 200 * time.Millisecond,
		LogQueries:         false,
	}
}

// SessionOption configures a Session
type SessionOption func(*Session)

// WithLogger sets the logger for the session
func WithLogger(logger *slog.Logger) SessionOption {
	return func(s *Session) {
		s.obs.Logger = logger
	}
}

// WithTracer sets the OpenTelemetry tracer for the session
func WithTracer(tracer trace.Tracer) SessionOption {
	return func(s *Session) {
		s.obs.Tracer = tracer
	}
}

// WithDefaultTracer uses the global OpenTelemetry tracer
func WithDefaultTracer() SessionOption {
	return func(s *Session) {
		s.obs.Tracer = otel.Tracer(tracerName)
	}
}

// WithMeter sets the OpenTelemetry meter for metrics
func WithMeter(meter metric.Meter) SessionOption {
	return func(s *Session) {
		s.obs.Meter = meter
		s.obs.Metrics = initMetrics(meter)
	}
}

// WithDefaultMeter uses the global OpenTelemetry meter
func WithDefaultMeter() SessionOption {
	return func(s *Session) {
		meter := otel.Meter(meterName)
		s.obs.Meter = meter
		s.obs.Metrics = initMetrics(meter)
	}
}

// initMetrics creates all metric instruments
func initMetrics(meter metric.Meter) *Metrics {
	queryCount, _ := meter.Int64Counter("sqlc.query.count",
		metric.WithDescription("Total number of SQL queries executed"),
		metric.WithUnit("{query}"),
	)

	queryDuration, _ := meter.Float64Histogram("sqlc.query.duration",
		metric.WithDescription("Query execution duration in milliseconds"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000),
	)

	queryErrors, _ := meter.Int64Counter("sqlc.query.errors",
		metric.WithDescription("Total number of query errors"),
		metric.WithUnit("{error}"),
	)

	return &Metrics{
		QueryCount:    queryCount,
		QueryDuration: queryDuration,
		QueryErrors:   queryErrors,
	}
}

// WithSlowQueryThreshold sets the slow query threshold for logging
func WithSlowQueryThreshold(d time.Duration) SessionOption {
	return func(s *Session) {
		s.obs.SlowQueryThreshold = d
	}
}

// WithQueryLogging enables logging of all queries
func WithQueryLogging(enabled bool) SessionOption {
	return func(s *Session) {
		s.obs.LogQueries = enabled
	}
}

// spanWrapper wraps a trace.Span to handle nil spans gracefully
type spanWrapper struct {
	span trace.Span
}

func (w spanWrapper) End() {
	if w.span != nil {
		w.span.End()
	}
}

func (w spanWrapper) RecordError(err error) {
	if w.span != nil {
		w.span.RecordError(err)
	}
}

func (w spanWrapper) SetStatus(code codes.Code, description string) {
	if w.span != nil {
		w.span.SetStatus(code, description)
	}
}

func (w spanWrapper) SetAttributes(kv ...attribute.KeyValue) {
	if w.span != nil {
		w.span.SetAttributes(kv...)
	}
}

// startSpan starts a new span if tracing is enabled
func (s *Session) startSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, spanWrapper) {
	if s.obs.Tracer == nil {
		return ctx, spanWrapper{nil}
	}
	ctx, span := s.obs.Tracer.Start(ctx, name, opts...)
	return ctx, spanWrapper{span}
}

// recordMetrics records query metrics if metrics are enabled
func (s *Session) recordMetrics(ctx context.Context, operation string, duration time.Duration, err error) {
	if s.obs.Metrics == nil {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("db.operation", operation),
		attribute.String("db.system", s.dialect.Name()),
	)

	// Record query count
	s.obs.Metrics.QueryCount.Add(ctx, 1, attrs)

	// Record duration
	s.obs.Metrics.QueryDuration.Record(ctx, float64(duration.Milliseconds()), attrs)

	// Record errors
	if err != nil {
		s.obs.Metrics.QueryErrors.Add(ctx, 1, attrs)
	}
}

// logQuery logs a query execution
func (s *Session) logQuery(ctx context.Context, operation, query string, duration time.Duration, err error) {
	if s.obs.Logger == nil {
		return
	}

	attrs := []slog.Attr{
		slog.String("operation", operation),
		slog.Duration("duration", duration),
	}

	if s.obs.LogQueries {
		attrs = append(attrs, slog.String("query", query))
	}

	if err != nil {
		s.obs.Logger.LogAttrs(ctx, slog.LevelError, "query failed", append(attrs, slog.String("error", err.Error()))...)
		return
	}

	if duration > s.obs.SlowQueryThreshold {
		s.obs.Logger.LogAttrs(ctx, slog.LevelWarn, "slow query", attrs...)
		return
	}

	if s.obs.LogQueries {
		s.obs.Logger.LogAttrs(ctx, slog.LevelDebug, "query executed", attrs...)
	}
}
