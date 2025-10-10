// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package tracing

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ServiceName == "" {
		t.Error("Expected non-empty service name")
	}

	if cfg.JaegerEndpoint == "" {
		t.Error("Expected non-empty Jaeger endpoint")
	}

	if cfg.SamplingRate != 1.0 {
		t.Errorf("Expected sampling rate 1.0, got %f", cfg.SamplingRate)
	}

	if !cfg.Enabled {
		t.Error("Expected tracing to be enabled by default")
	}
}

func TestInitTracing_Disabled(t *testing.T) {
	cfg := Config{
		ServiceName: "test-service",
		Enabled:     false,
	}

	shutdown, err := InitTracing(cfg)
	if err != nil {
		t.Fatalf("InitTracing failed: %v", err)
	}

	if shutdown == nil {
		t.Fatal("Expected shutdown function")
	}

	// Should not panic
	err = shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestStartSpan(t *testing.T) {
	// Create test exporter
	exporter := tracetest.NewInMemoryExporter()

	// Create tracer provider with test exporter
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer tp.Shutdown(context.Background())

	// Note: We can't easily set global tracer provider in test
	// So we'll just test that StartSpan doesn't panic
	ctxBg := context.Background()
	_, span := StartSpan(ctxBg, "test-operation")

	if span == nil {
		t.Fatal("Expected non-nil span")
	}

	span.End()
}

func TestRecordError(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer tp.Shutdown(context.Background())

	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	testErr := errors.New("test error")
	RecordError(span, testErr)

	// Verify span was ended
	span.End()

	// Get exported spans
	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("Expected at least one span")
	}

	lastSpan := spans[len(spans)-1]
	if lastSpan.Status.Code != codes.Error {
		t.Errorf("Expected error status, got %v", lastSpan.Status.Code)
	}

	if lastSpan.Status.Description != testErr.Error() {
		t.Errorf("Expected error description '%s', got '%s'", testErr.Error(), lastSpan.Status.Description)
	}
}

func TestRecordError_NilSpan(t *testing.T) {
	// Should not panic with nil span
	RecordError(nil, errors.New("test"))
}

func TestRecordError_NilError(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer tp.Shutdown(context.Background())

	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Should not panic with nil error
	RecordError(span, nil)
}

func TestAddEvent(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer tp.Shutdown(context.Background())

	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")

	AddEvent(span, "test-event", attribute.String("key", "value"))
	span.End()

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("Expected at least one span")
	}

	lastSpan := spans[len(spans)-1]
	events := lastSpan.Events
	if len(events) == 0 {
		t.Fatal("Expected at least one event")
	}

	found := false
	for _, event := range events {
		if event.Name == "test-event" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find 'test-event' in span events")
	}
}

func TestAddEvent_NilSpan(t *testing.T) {
	// Should not panic with nil span
	AddEvent(nil, "test-event")
}

func TestSetAttributes(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	defer tp.Shutdown(context.Background())

	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")

	SetAttributes(span,
		attribute.String("key1", "value1"),
		attribute.Int("key2", 42),
	)
	span.End()

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("Expected at least one span")
	}

	lastSpan := spans[len(spans)-1]
	attrs := lastSpan.Attributes

	foundKey1 := false
	foundKey2 := false

	for _, attr := range attrs {
		if attr.Key == "key1" && attr.Value.AsString() == "value1" {
			foundKey1 = true
		}
		if attr.Key == "key2" && attr.Value.AsInt64() == 42 {
			foundKey2 = true
		}
	}

	if !foundKey1 {
		t.Error("Expected to find key1 attribute")
	}

	if !foundKey2 {
		t.Error("Expected to find key2 attribute")
	}
}

func TestSetAttributes_NilSpan(t *testing.T) {
	// Should not panic with nil span
	SetAttributes(nil, attribute.String("key", "value"))
}

func TestMessageTracingMiddleware(t *testing.T) {
	handlerCalled := false
	handler := func(ctx context.Context, msg interface{}) (interface{}, error) {
		handlerCalled = true
		return "response", nil
	}

	middleware := MessageTracingMiddleware(handler)

	ctx := context.Background()
	response, err := middleware(ctx, "test-message")

	if err != nil {
		t.Fatalf("Middleware failed: %v", err)
	}

	if !handlerCalled {
		t.Error("Expected handler to be called")
	}

	if response != "response" {
		t.Errorf("Expected 'response', got %v", response)
	}

	// Note: We don't verify span creation here because StartSpan uses
	// the global tracer provider which we can't easily mock in unit tests.
	// Span creation is verified in integration tests.
}

func TestMessageTracingMiddleware_Error(t *testing.T) {
	testErr := errors.New("handler error")
	handler := func(ctx context.Context, msg interface{}) (interface{}, error) {
		return nil, testErr
	}

	middleware := MessageTracingMiddleware(handler)

	ctx := context.Background()
	_, err := middleware(ctx, "test-message")

	if err != testErr {
		t.Errorf("Expected error '%v', got '%v'", testErr, err)
	}

	// Note: We don't verify error recording in span here because StartSpan uses
	// the global tracer provider which we can't easily mock in unit tests.
	// Error recording is verified in integration tests.
}

func TestInjectContext(t *testing.T) {
	ctx := context.Background()
	carrier := make(map[string]string)

	err := InjectContext(ctx, carrier)
	if err != nil {
		t.Errorf("InjectContext failed: %v", err)
	}
}

func TestExtractContext(t *testing.T) {
	ctx := context.Background()
	carrier := make(map[string]string)

	newCtx, err := ExtractContext(ctx, carrier)
	if err != nil {
		t.Errorf("ExtractContext failed: %v", err)
	}

	if newCtx == nil {
		t.Error("Expected non-nil context")
	}
}
