// Package sqlc provides a type-safe ORM library using generics and code generation.
// This file implements observability functionality, including logging, tracing, and metrics.
//
// Observability is a critical component of database operations in production environments.
// sqlc provides comprehensive observability support:
//   - Structured logging: Records all SQL queries, execution time, error messages
//   - Distributed tracing: Traces database operations via OpenTelemetry
//   - Performance metrics: Collects query count, latency distribution, error rate, etc.
//   - Slow query detection: Automatically identifies and logs slow queries
//
// Usage example:
//
//	// Basic logging configuration
//	session := sqlc.NewSession(db, sqlc.MySQL{},
//	    sqlc.WithLogger(slog.Default()),
//	    sqlc.WithQueryLogging(true),
//	)
//
//	// Complete observability configuration
//	session := sqlc.NewSession(db, sqlc.PostgreSQL{},
//	    sqlc.WithLogger(slog.Default()),
//	    sqlc.WithDefaultTracer(),
//	    sqlc.WithDefaultMeter(),
//	    sqlc.WithSlowQueryThreshold(100*time.Millisecond),
//	    sqlc.WithQueryLogging(true),
//	)
//
// Observability data usage:
//   - Performance analysis: Identify slow queries and performance bottlenecks
//   - Troubleshooting: Locate issues through trace chains
//   - Capacity planning: Analyze database load through metrics
//   - Security audit: Record all database operations
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
	// tracerName is the name of the OpenTelemetry tracer.
	// Used to identify the source of trace data for filtering and analysis in tracing systems.
	tracerName = "github.com/arllen133/sqlc"

	// meterName is the name of the OpenTelemetry meter.
	// Used to identify the source of metric data for filtering and analysis in monitoring systems.
	meterName = "github.com/arllen133/sqlc"
)

// Metrics contains all OpenTelemetry metric instruments.
// These metrics are used to monitor the performance and health of database operations.
//
// Metric types:
//   - QueryCount: Counter, records total number of queries
//   - QueryDuration: Histogram, records query latency distribution
//   - QueryErrors: Counter, records total number of query errors
//
// Usage scenarios:
//   - Monitor database load and throughput
//   - Analyze query performance and latency distribution
//   - Track error rates and anomalies
//   - Set up alerts and automated responses
type Metrics struct {
	// QueryCount records the total number of SQL queries executed.
	// Grouped by operation type (select, insert, update, delete) and database type.
	//
	// Metric attributes:
	//   - db.operation: Operation type (select, exec, query, etc.)
	//   - db.system: Database type (mysql, postgres, sqlite3)
	//
	// Usage:
	//   - Monitor query throughput
	//   - Analyze operation type distribution
	//   - Capacity planning and performance optimization
	QueryCount metric.Int64Counter

	// QueryDuration records the distribution of query execution time.
	// Using histogram allows analyzing latency percentiles (P50, P95, P99).
	//
	// Metric attributes:
	//   - db.operation: Operation type
	//   - db.system: Database type
	//
	// Unit: milliseconds (ms)
	//
	// Predefined bucket boundaries: 1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000
	//
	// Usage:
	//   - Identify slow queries
	//   - Analyze performance trends
	//   - Set SLAs and alert thresholds
	QueryDuration metric.Float64Histogram

	// QueryErrors records the total number of query errors.
	// Grouped by operation type and database type.
	//
	// Metric attributes:
	//   - db.operation: Operation type
	//   - db.system: Database type
	//
	// Usage:
	//   - Monitor error rates
	//   - Identify anomaly patterns
	//   - Set up error alerts
	QueryErrors metric.Int64Counter
}

// ObservabilityConfig holds configuration for logging, tracing, and metrics.
// This configuration controls the observability behavior of Session.
//
// Configuration items:
//   - Logger: Structured logger (slog.Logger)
//   - Tracer: OpenTelemetry tracer
//   - Meter: OpenTelemetry metrics collector
//   - Metrics: Initialized metric instruments
//   - SlowQueryThreshold: Slow query threshold
//   - LogQueries: Whether to log all queries
//
// Usage example:
//
//	// Custom configuration
//	config := &ObservabilityConfig{
//	    Logger:             slog.Default(),
//	    Tracer:             otel.Tracer("my-app"),
//	    SlowQueryThreshold: 200 * time.Millisecond,
//	    LogQueries:         true,
//	}
type ObservabilityConfig struct {
	// Logger is the structured logger for recording query logs.
	// If nil, no logs are recorded.
	//
	// Log levels:
	//   - Debug: All queries (requires LogQueries = true)
	//   - Warn: Slow queries
	//   - Error: Failed queries
	//
	// Log fields:
	//   - operation: Operation type
	//   - duration: Execution duration
	//   - query: SQL statement (requires LogQueries = true)
	//   - error: Error message (if failed)
	Logger *slog.Logger

	// Tracer is the OpenTelemetry tracer for creating distributed trace spans.
	// If nil, no trace data is created.
	//
	// Span attributes:
	//   - db.statement: SQL statement
	//   - db.operation: Operation type
	//   - db.system: Database type
	//
	// Usage:
	//   - Trace request flow through the system
	//   - Analyze database operation proportion in overall requests
	//   - Identify performance bottlenecks
	Tracer trace.Tracer

	// Meter is the OpenTelemetry metrics collector.
	// If nil, no metric data is collected.
	//
	// Usage:
	//   - Create counters and histograms
	//   - Collect performance metrics
	Meter metric.Meter

	// Metrics contains initialized metric instruments.
	// Automatically initialized via WithMeter() or WithDefaultMeter().
	Metrics *Metrics

	// SlowQueryThreshold defines the threshold for slow queries.
	// Queries exceeding this threshold are logged at warning level.
	//
	// Default: 200 milliseconds
	//
	// Usage:
	//   - Identify queries needing optimization
	//   - Set performance baselines
	SlowQueryThreshold time.Duration

	// LogQueries controls whether to log all queries.
	// If true, all queries are logged at Debug level.
	//
	// Default: false
	//
	// Note:
	//   - Enabling this generates a large volume of logs, use only for debugging
	//   - For production, recommend disabling or using sampling
	//   - Slow queries and error queries are always logged
	LogQueries bool
}

// defaultObservabilityConfig returns the default observability configuration.
// Default configuration doesn't enable any observability features, need to explicitly enable via options.
//
// Default values:
//   - Logger: nil (no logging)
//   - Tracer: nil (no tracing)
//   - Meter: nil (no metrics)
//   - Metrics: nil
//   - SlowQueryThreshold: 200 milliseconds
//   - LogQueries: false
//
// Returns:
//   - *ObservabilityConfig: Default configuration instance
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

// SessionOption defines a function for configuring Session.
// Uses functional options pattern to provide flexible configuration.
//
// Advantages:
//   - Optional configuration: Unneeded configurations can be omitted
//   - Composable: Multiple options can be combined
//   - Extensible: Easy to add new configuration options
//   - Readable: Configuration intent is clear
//
// Example:
//
//	session := sqlc.NewSession(db, sqlc.MySQL{},
//	    sqlc.WithLogger(logger),
//	    sqlc.WithQueryLogging(true),
//	    sqlc.WithSlowQueryThreshold(100*time.Millisecond),
//	)
type SessionOption func(*Session)

// WithLogger sets the logger for the session.
// When enabled, query execution status, slow queries, and errors are logged.
//
// Parameter:
//   - logger: slog.Logger instance, cannot be nil
//
// Usage example:
//
//	// Use default logger
//	session := sqlc.NewSession(db, sqlc.MySQL{},
//	    sqlc.WithLogger(slog.Default()),
//	)
//
//	// Use custom logger
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
//	    Level: slog.LevelDebug,
//	}))
//	session := sqlc.NewSession(db, sqlc.MySQL{},
//	    sqlc.WithLogger(logger),
//	)
//
// Note:
//   - Only sets the logger, doesn't automatically log all queries
//   - Use with WithQueryLogging(true) to log all queries
//   - Slow queries and error queries are always logged
func WithLogger(logger *slog.Logger) SessionOption {
	return func(s *Session) {
		s.obs.Logger = logger
	}
}

// WithTracer sets the OpenTelemetry tracer.
// When enabled, all database operations create trace spans.
//
// Parameter:
//   - tracer: trace.Tracer instance
//
// Usage example:
//
//	// Use custom tracer
//	tracer := otel.Tracer("my-service")
//	session := sqlc.NewSession(db, sqlc.MySQL{},
//	    sqlc.WithTracer(tracer),
//	)
//
// Trace data includes:
//   - Span name: Operation type (e.g., "sqlc.Query", "sqlc.Exec")
//   - Span attributes: SQL statement, operation type, database type
//   - Span status: Success or failure
//   - Span events: Error information (if failed)
//
// Note:
//   - Requires OpenTelemetry SDK configuration to take effect
//   - Recommend using WithDefaultTracer() for simpler configuration
func WithTracer(tracer trace.Tracer) SessionOption {
	return func(s *Session) {
		s.obs.Tracer = tracer
	}
}

// WithDefaultTracer creates a tracer using the global OpenTelemetry TracerProvider.
// This is the simplest way to enable tracing.
//
// Usage example:
//
//	// Initialize OpenTelemetry (must be done at program startup)
//	tp := tracesdk.NewTracerProvider(...)
//	otel.SetTracerProvider(tp)
//
//	// Enable tracing when creating Session
//	session := sqlc.NewSession(db, sqlc.MySQL{},
//	    sqlc.WithDefaultTracer(),
//	)
//
// Note:
//   - Requires global TracerProvider to be configured first
//   - Tracer name is "github.com/arllen133/sqlc"
func WithDefaultTracer() SessionOption {
	return func(s *Session) {
		s.obs.Tracer = otel.Tracer(tracerName)
	}
}

// WithMeter sets the OpenTelemetry metrics collector.
// When enabled, collects query count, latency, errors, and other metrics.
//
// Parameter:
//   - meter: metric.Meter instance
//
// Usage example:
//
//	// Use custom meter
//	meter := otel.Meter("my-service")
//	session := sqlc.NewSession(db, sqlc.MySQL{},
//	    sqlc.WithMeter(meter),
//	)
//
// Collected metrics:
//   - sqlc.query.count: Total query count (grouped by operation type)
//   - sqlc.query.duration: Query latency distribution (milliseconds)
//   - sqlc.query.errors: Query error count (grouped by operation type)
//
// Note:
//   - Requires OpenTelemetry SDK configuration to take effect
//   - Recommend using WithDefaultMeter() for simpler configuration
func WithMeter(meter metric.Meter) SessionOption {
	return func(s *Session) {
		s.obs.Meter = meter
		s.obs.Metrics = initMetrics(meter)
	}
}

// WithDefaultMeter creates a metrics collector using the global OpenTelemetry MeterProvider.
// This is the simplest way to enable metrics collection.
//
// Usage example:
//
//	// Initialize OpenTelemetry (must be done at program startup)
//	mp := metricsdk.NewMeterProvider(...)
//	otel.SetMeterProvider(mp)
//
//	// Enable metrics collection when creating Session
//	session := sqlc.NewSession(db, sqlc.MySQL{},
//	    sqlc.WithDefaultMeter(),
//	)
//
// Collected metrics:
//
//   - sqlc.query.count: Total query count
//
//     Unit: {query}
//
//     Attributes: db.operation, db.system
//
//   - sqlc.query.duration: Query latency distribution
//
//     Unit: ms
//
//     Attributes: db.operation, db.system
//
//     Bucket boundaries: 1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000
//
//   - sqlc.query.errors: Query error count
//
//     Unit: {error}
//
//     Attributes: db.operation, db.system
//
// Note:
//   - Requires global MeterProvider to be configured first
//   - Metric names are prefixed with "sqlc."
func WithDefaultMeter() SessionOption {
	return func(s *Session) {
		meter := otel.Meter(meterName)
		s.obs.Meter = meter
		s.obs.Metrics = initMetrics(meter)
	}
}

// initMetrics initializes all metric instruments.
// Creates query counter, latency histogram, and error counter.
//
// Parameter:
//   - meter: OpenTelemetry meter instance
//
// Returns:
//   - *Metrics: Initialized metric instruments collection
//
// Created metrics:
//   - sqlc.query.count (Int64Counter): Query counter
//   - sqlc.query.duration (Float64Histogram): Latency histogram
//   - sqlc.query.errors (Int64Counter): Error counter
//
// Note:
//   - If metric creation fails, errors are ignored (uses no-op implementation)
//   - This ensures program continues to run even if metrics initialization fails
func initMetrics(meter metric.Meter) *Metrics {
	// Create query counter
	// Records total queries executed, grouped by operation type and database type
	queryCount, _ := meter.Int64Counter("sqlc.query.count",
		metric.WithDescription("Total number of SQL queries executed"),
		metric.WithUnit("{query}"),
	)

	// Create latency histogram
	// Records query execution time distribution for performance analysis
	// Predefined bucket boundaries cover range from 1ms to 5s
	queryDuration, _ := meter.Float64Histogram("sqlc.query.duration",
		metric.WithDescription("Query execution duration in milliseconds"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000),
	)

	// Create error counter
	// Records total query errors for monitoring error rates
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

// WithSlowQueryThreshold sets the slow query threshold.
// Queries exceeding this threshold are logged at warning level.
//
// Parameter:
//   - d: Slow query threshold (time.Duration)
//
// Usage example:
//
//	// Set 100ms as slow query threshold
//	session := sqlc.NewSession(db, sqlc.MySQL{},
//	    sqlc.WithLogger(slog.Default()),
//	    sqlc.WithSlowQueryThreshold(100*time.Millisecond),
//	)
//
//	// Set 1s as slow query threshold
//	session := sqlc.NewSession(db, sqlc.PostgreSQL{},
//	    sqlc.WithLogger(slog.Default()),
//	    sqlc.WithSlowQueryThreshold(time.Second),
//	)
//
// Note:
//   - Requires Logger configuration to log slow queries
//   - Slow queries are logged at Warn level
//   - Default threshold is 200ms
func WithSlowQueryThreshold(d time.Duration) SessionOption {
	return func(s *Session) {
		s.obs.SlowQueryThreshold = d
	}
}

// WithQueryLogging controls whether to log all queries.
// When enabled, all queries are logged at Debug level.
//
// Parameter:
//   - enabled: true to enable, false to disable
//
// Usage example:
//
//	// Enable query logging (debug mode)
//	session := sqlc.NewSession(db, sqlc.MySQL{},
//	    sqlc.WithLogger(slog.Default()),
//	    sqlc.WithQueryLogging(true),
//	)
//
//	// Disable query logging (production mode)
//	session := sqlc.NewSession(db, sqlc.MySQL{},
//	    sqlc.WithLogger(slog.Default()),
//	    sqlc.WithQueryLogging(false),
//	)
//
// Log levels:
//   - Debug: All queries (enabled = true)
//   - Warn: Slow queries (always logged)
//   - Error: Failed queries (always logged)
//
// Note:
//   - Enabling this generates a large volume of logs, may impact performance
//   - For production, recommend disabling or using sampling
//   - Requires Logger configuration to take effect
func WithQueryLogging(enabled bool) SessionOption {
	return func(s *Session) {
		s.obs.LogQueries = enabled
	}
}

// spanWrapper wraps trace.Span to handle nil spans gracefully.
// When tracing is not enabled (Tracer is nil), uses nil span to avoid null pointer errors.
//
// Design pattern: Null Object Pattern
//   - When Tracer is not configured, returns nil span
//   - All method calls on nil span are no-ops
//   - Avoids checking for nil at each call site
type spanWrapper struct {
	span trace.Span
}

// End ends the span.
// If span is nil, this is a no-op.
func (w spanWrapper) End() {
	if w.span != nil {
		w.span.End()
	}
}

// RecordError records an error to the span.
// If span is nil, this is a no-op.
//
// Parameter:
//   - err: Error to record
func (w spanWrapper) RecordError(err error) {
	if w.span != nil {
		w.span.RecordError(err)
	}
}

// SetStatus sets the span status.
// If span is nil, this is a no-op.
//
// Parameters:
//   - code: Status code (Ok, Error, Unset)
//   - description: Status description
func (w spanWrapper) SetStatus(code codes.Code, description string) {
	if w.span != nil {
		w.span.SetStatus(code, description)
	}
}

// SetAttributes sets span attributes.
// If span is nil, this is a no-op.
//
// Parameter:
//   - kv: Attribute key-value pairs
func (w spanWrapper) SetAttributes(kv ...attribute.KeyValue) {
	if w.span != nil {
		w.span.SetAttributes(kv...)
	}
}

// startSpan starts a new trace span.
// If tracing is not enabled (Tracer is nil), returns nil span wrapper.
//
// Parameters:
//   - ctx: Context for propagating trace information
//   - name: Span name (e.g., "sqlc.Query", "sqlc.Exec")
//   - opts: Optional span start options
//
// Returns:
//   - context.Context: Context containing new span
//   - spanWrapper: Span wrapper (may be nil)
//
// Usage example (internal use):
//
//	ctx, span := s.startSpan(ctx, "sqlc.Query")
//	defer span.End()
//
//	// Execute database operations...
//
//	if err != nil {
//	    span.RecordError(err)
//	    span.SetStatus(codes.Error, err.Error())
//	}
func (s *Session) startSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, spanWrapper) {
	// Check if tracer is configured
	if s.obs.Tracer == nil {
		// Not configured, return nil span wrapper
		return ctx, spanWrapper{nil}
	}

	// Start new span
	ctx, span := s.obs.Tracer.Start(ctx, name, opts...)
	return ctx, spanWrapper{span}
}

// recordMetrics records query metrics.
// If metrics are not enabled (Metrics is nil), this is a no-op.
//
// Parameters:
//   - ctx: Context for metric recording
//   - operation: Operation type (select, exec, query, etc.)
//   - duration: Query execution duration
//   - err: Query error (if any)
//
// Recorded metrics:
//   - sqlc.query.count: Increment by 1
//   - sqlc.query.duration: Record latency
//   - sqlc.query.errors: If error exists, increment by 1
//
// Metric attributes:
//   - db.operation: Operation type
//   - db.system: Database type
//
// Usage example (internal use):
//
//	start := time.Now()
//	err := executeQuery()
//	duration := time.Since(start)
//
//	s.recordMetrics(ctx, "select", duration, err)
func (s *Session) recordMetrics(ctx context.Context, operation string, duration time.Duration, err error) {
	// Check if metrics are configured
	if s.obs.Metrics == nil {
		return
	}

	// Prepare metric attributes
	attrs := metric.WithAttributes(
		attribute.String("db.operation", operation),
		attribute.String("db.system", s.dialect.Name()),
	)

	// Record query count (increment by 1 for each query)
	s.obs.Metrics.QueryCount.Add(ctx, 1, attrs)

	// Record query latency (milliseconds)
	s.obs.Metrics.QueryDuration.Record(ctx, float64(duration.Milliseconds()), attrs)

	// If error exists, record error count
	if err != nil {
		s.obs.Metrics.QueryErrors.Add(ctx, 1, attrs)
	}
}

// logQuery logs a query execution.
// If logging is not enabled (Logger is nil), this is a no-op.
//
// Parameters:
//   - ctx: Context for logging
//   - operation: Operation type (select, exec, query, etc.)
//   - query: SQL query statement
//   - duration: Query execution duration
//   - err: Query error (if any)
//
// Log levels:
//   - Error: Query failed (includes error message)
//   - Warn: Slow query (exceeds SlowQueryThreshold)
//   - Debug: All queries (requires LogQueries = true)
//
// Log fields:
//   - operation: Operation type
//   - duration: Execution duration
//   - query: SQL statement (requires LogQueries = true)
//   - error: Error message (if failed)
//
// Usage example (internal use):
//
//	start := time.Now()
//	err := executeQuery()
//	duration := time.Since(start)
//
//	s.logQuery(ctx, "select", query, duration, err)
func (s *Session) logQuery(ctx context.Context, operation, query string, duration time.Duration, err error) {
	// Check if logger is configured
	if s.obs.Logger == nil {
		return
	}

	// Prepare base log attributes
	attrs := []slog.Attr{
		slog.String("operation", operation),
		slog.Duration("duration", duration),
	}

	// If query logging is enabled, add SQL statement
	if s.obs.LogQueries {
		attrs = append(attrs, slog.String("query", query))
	}

	// Error case: Log at Error level
	if err != nil {
		s.obs.Logger.LogAttrs(ctx, slog.LevelError, "query failed",
			append(attrs, slog.String("error", err.Error()))...)
		return
	}

	// Slow query: Log at Warn level
	if duration > s.obs.SlowQueryThreshold {
		s.obs.Logger.LogAttrs(ctx, slog.LevelWarn, "slow query", attrs...)
		return
	}

	// Normal query: Log at Debug level (requires LogQueries = true)
	if s.obs.LogQueries {
		s.obs.Logger.LogAttrs(ctx, slog.LevelDebug, "query executed", attrs...)
	}
}
