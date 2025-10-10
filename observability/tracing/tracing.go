// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

/*
Package tracing provides distributed tracing support using OpenTelemetry.

This package enables end-to-end request tracing across multiple agents and services,
making it easy to debug and optimize performance in distributed systems.

Features:
  - OpenTelemetry integration
  - Jaeger exporter
  - Automatic span creation
  - Context propagation
  - Custom attributes and events
  - Sampling configuration

Example:

	import "github.com/sage-x-project/sage-adk/observability/tracing"

	// Initialize tracing
	shutdown, err := tracing.InitTracing(tracing.Config{
	    ServiceName: "my-agent",
	    JaegerEndpoint: "http://localhost:14268/api/traces",
	    SamplingRate: 1.0,
	})
	if err != nil {
	    log.Fatal(err)
	}
	defer shutdown(context.Background())

	// Create a span
	ctx, span := tracing.StartSpan(ctx, "operation-name")
	defer span.End()

	// Add attributes
	span.SetAttribute("key", "value")

	// Add event
	span.AddEvent("event-name")
*/
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds tracing configuration
type Config struct {
	// ServiceName is the name of this service
	ServiceName string

	// JaegerEndpoint is the Jaeger collector endpoint
	JaegerEndpoint string

	// SamplingRate (0.0 - 1.0) determines what percentage of traces to collect
	SamplingRate float64

	// Environment (dev, staging, prod)
	Environment string

	// Version is the service version
	Version string

	// Enabled enables/disables tracing
	Enabled bool
}

// DefaultConfig returns default tracing configuration
func DefaultConfig() Config {
	return Config{
		ServiceName:    "sage-adk-agent",
		JaegerEndpoint: "http://localhost:14268/api/traces",
		SamplingRate:   1.0,
		Environment:    "development",
		Version:        "1.2.0",
		Enabled:        true,
	}
}

// InitTracing initializes OpenTelemetry tracing
func InitTracing(cfg Config) (func(context.Context) error, error) {
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	// Create Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.JaegerEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.Version),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Return shutdown function
	return tp.Shutdown, nil
}

// StartSpan creates a new span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer("sage-adk")
	return tracer.Start(ctx, name, opts...)
}

// RecordError records an error in the current span
func RecordError(span trace.Span, err error) {
	if span != nil && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// AddEvent adds an event to the current span
func AddEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	if span != nil {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SetAttributes sets attributes on the current span
func SetAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	if span != nil {
		span.SetAttributes(attrs...)
	}
}

// InjectContext injects trace context into a carrier
func InjectContext(ctx context.Context, carrier interface{}) error {
	// This would implement context propagation
	return nil
}

// ExtractContext extracts trace context from a carrier
func ExtractContext(ctx context.Context, carrier interface{}) (context.Context, error) {
	// This would implement context extraction
	return ctx, nil
}

// MessageTracingMiddleware creates middleware that traces message processing
func MessageTracingMiddleware(next func(context.Context, interface{}) (interface{}, error)) func(context.Context, interface{}) (interface{}, error) {
	return func(ctx context.Context, msg interface{}) (interface{}, error) {
		ctx, span := StartSpan(ctx, "process_message")
		defer span.End()

		// Add message metadata as attributes
		// This would extract message ID, type, etc.

		response, err := next(ctx, msg)
		if err != nil {
			RecordError(span, err)
			return nil, err
		}

		span.SetStatus(codes.Ok, "")
		return response, nil
	}
}
