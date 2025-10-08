// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later

package middleware

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

func TestLogger(t *testing.T) {
	var buf strings.Builder
	logger := log.New(&buf, "", 0)

	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	chain := NewChain(Logger(logger))
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	_, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test-123") {
		t.Error("Logger should log message ID")
	}
	if !strings.Contains(output, "Incoming message") {
		t.Error("Logger should log incoming message")
	}
	if !strings.Contains(output, "Request completed") {
		t.Error("Logger should log completion")
	}
}

func TestLogger_WithError(t *testing.T) {
	var buf strings.Builder
	logger := log.New(&buf, "", 0)

	expectedErr := errors.New("test error")
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return nil, expectedErr
	}

	chain := NewChain(Logger(logger))
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	_, err := chain.Execute(context.Background(), msg, handler)
	if err != expectedErr {
		t.Fatalf("Execute() error = %v, want %v", err, expectedErr)
	}

	output := buf.String()
	if !strings.Contains(output, "Request failed") {
		t.Error("Logger should log failure")
	}
	if !strings.Contains(output, "test error") {
		t.Error("Logger should log error message")
	}
}

func TestLogger_NilLogger(t *testing.T) {
	// Should use default logger without panic
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	chain := NewChain(Logger(nil))
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	_, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestRequestID(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		// Check that request ID was added to context
		requestID, ok := RequestIDFromContext(ctx)
		if !ok {
			t.Error("Request ID not found in context")
		}
		if requestID != "test-123" {
			t.Errorf("Request ID = %s, want test-123", requestID)
		}
		return msg, nil
	}

	chain := NewChain(RequestID())
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	_, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestRequestID_EmptyMessageID(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		// Should generate request ID if message ID is empty
		requestID, ok := RequestIDFromContext(ctx)
		if !ok {
			t.Error("Request ID not found in context")
		}
		if requestID == "" {
			t.Error("Request ID should be generated")
		}
		return msg, nil
	}

	chain := NewChain(RequestID())
	msg := &types.Message{
		MessageID: "", // Empty
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	_, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestTimer(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		// Check that start time was added to context
		startTime := ctx.Value(StartTimeKey)
		if startTime == nil {
			t.Error("Start time not found in context")
		}

		// Simulate work
		time.Sleep(10 * time.Millisecond)

		return msg, nil
	}

	chain := NewChain(Timer())
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	resp, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Check that timing metadata was added
	if resp.Metadata == nil {
		t.Fatal("Response metadata is nil")
	}
	processingTime := resp.Metadata["processing_time_ms"]
	if processingTime == "" {
		t.Error("Processing time not added to metadata")
	}
}

func TestTimer_WithError(t *testing.T) {
	expectedErr := errors.New("test error")
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return nil, expectedErr
	}

	chain := NewChain(Timer())
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	resp, err := chain.Execute(context.Background(), msg, handler)
	if err != expectedErr {
		t.Fatalf("Execute() error = %v, want %v", err, expectedErr)
	}
	if resp != nil {
		t.Error("Response should be nil on error")
	}
}

func TestRecovery(t *testing.T) {
	// Suppress log output for this test
	log.SetOutput(os.NewFile(0, os.DevNull))
	defer log.SetOutput(os.Stderr)

	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		panic("test panic")
	}

	chain := NewChain(Recovery())
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	resp, err := chain.Execute(context.Background(), msg, handler)
	if err == nil {
		t.Error("Execute() should return error after panic")
	}
	if !strings.Contains(err.Error(), "panic recovered") {
		t.Errorf("Error should indicate panic recovery: %v", err)
	}
	if resp != nil {
		t.Error("Response should be nil after panic")
	}
}

func TestRecovery_NoPanic(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	chain := NewChain(Recovery())
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	resp, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resp == nil {
		t.Error("Response should not be nil")
	}
}

func TestValidator_Success(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	chain := NewChain(Validator())
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	_, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestValidator_NilMessage(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	chain := NewChain(Validator())

	_, err := chain.Execute(context.Background(), nil, handler)
	if err == nil {
		t.Error("Execute() should return error for nil message")
	}
	if !strings.Contains(err.Error(), "message is nil") {
		t.Errorf("Error message should indicate nil message: %v", err)
	}
}

func TestValidator_EmptyMessageID(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	chain := NewChain(Validator())
	msg := &types.Message{
		MessageID: "", // Empty
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	_, err := chain.Execute(context.Background(), msg, handler)
	if err == nil {
		t.Error("Execute() should return error for empty message ID")
	}
}

func TestValidator_EmptyRole(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	chain := NewChain(Validator())
	msg := &types.Message{
		MessageID: "test-123",
		Role:      "", // Empty
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	_, err := chain.Execute(context.Background(), msg, handler)
	if err == nil {
		t.Error("Execute() should return error for empty role")
	}
}

func TestValidator_NoParts(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	chain := NewChain(Validator())
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{}, // Empty
	}

	_, err := chain.Execute(context.Background(), msg, handler)
	if err == nil {
		t.Error("Execute() should return error for empty parts")
	}
}

func TestRateLimiter_Success(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	config := RateLimiterConfig{
		MaxRequests: 3,
		Window:      1 * time.Second,
	}

	chain := NewChain(RateLimiter(config))
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	// Should allow first 3 requests
	for i := 0; i < 3; i++ {
		_, err := chain.Execute(context.Background(), msg, handler)
		if err != nil {
			t.Fatalf("Execute() error on request %d: %v", i+1, err)
		}
	}
}

func TestRateLimiter_Exceeded(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	config := RateLimiterConfig{
		MaxRequests: 2,
		Window:      1 * time.Second,
	}

	chain := NewChain(RateLimiter(config))
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		_, err := chain.Execute(context.Background(), msg, handler)
		if err != nil {
			t.Fatalf("Execute() error on request %d: %v", i+1, err)
		}
	}

	// Third request should fail
	_, err := chain.Execute(context.Background(), msg, handler)
	if err == nil {
		t.Error("Execute() should return error when rate limit exceeded")
	}
	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Errorf("Error should indicate rate limit exceeded: %v", err)
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	config := RateLimiterConfig{
		MaxRequests: 2,
		Window:      100 * time.Millisecond,
	}

	chain := NewChain(RateLimiter(config))
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	// Use up limit
	for i := 0; i < 2; i++ {
		_, err := chain.Execute(context.Background(), msg, handler)
		if err != nil {
			t.Fatalf("Execute() error on request %d: %v", i+1, err)
		}
	}

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// Should allow requests again
	_, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Errorf("Execute() error after window reset: %v", err)
	}
}

func TestMetadata(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		// Check that metadata was added to context
		metadata, ok := MetadataFromContext(ctx)
		if !ok {
			t.Error("Metadata not found in context")
		}
		if metadata["key1"] != "value1" {
			t.Errorf("Metadata[key1] = %v, want value1", metadata["key1"])
		}
		return msg, nil
	}

	metadata := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	chain := NewChain(Metadata(metadata))
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	resp, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Check that metadata was added to response
	if resp.Metadata == nil {
		t.Fatal("Response metadata is nil")
	}
	if resp.Metadata["key1"] != "value1" {
		t.Errorf("Response.Metadata[key1] = %v, want value1", resp.Metadata["key1"])
	}
	if resp.Metadata["key2"] != 42 {
		t.Errorf("Response.Metadata[key2] = %v, want 42", resp.Metadata["key2"])
	}
}

func TestMetadata_WithError(t *testing.T) {
	expectedErr := errors.New("test error")
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return nil, expectedErr
	}

	metadata := map[string]interface{}{
		"key1": "value1",
	}

	chain := NewChain(Metadata(metadata))
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	resp, err := chain.Execute(context.Background(), msg, handler)
	if err != expectedErr {
		t.Fatalf("Execute() error = %v, want %v", err, expectedErr)
	}
	if resp != nil {
		t.Error("Response should be nil on error")
	}
}

func TestTimeout_Success(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		time.Sleep(10 * time.Millisecond)
		return msg, nil
	}

	chain := NewChain(Timeout(100 * time.Millisecond))
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	_, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestTimeout_Exceeded(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		time.Sleep(200 * time.Millisecond)
		return msg, nil
	}

	chain := NewChain(Timeout(50 * time.Millisecond))
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	_, err := chain.Execute(context.Background(), msg, handler)
	if err == nil {
		t.Error("Execute() should return error when timeout exceeded")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Error should indicate timeout: %v", err)
	}
}

func TestContentFilter_Allow(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	filterFunc := func(content string) (bool, string) {
		return true, "" // Allow all
	}

	chain := NewChain(ContentFilter(filterFunc))
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	_, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestContentFilter_Block(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	filterFunc := func(content string) (bool, string) {
		if strings.Contains(content, "bad") {
			return false, "contains prohibited word"
		}
		return true, ""
	}

	chain := NewChain(ContentFilter(filterFunc))
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "this is bad content"}},
	}

	_, err := chain.Execute(context.Background(), msg, handler)
	if err == nil {
		t.Error("Execute() should return error when content is blocked")
	}
	if !strings.Contains(err.Error(), "content blocked") {
		t.Errorf("Error should indicate content blocked: %v", err)
	}
}

func TestContentFilter_NonTextParts(t *testing.T) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	filterFunc := func(content string) (bool, string) {
		return false, "should not be called for non-text parts"
	}

	chain := NewChain(ContentFilter(filterFunc))
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts: []types.Part{
			&types.FilePart{
				Kind: "file",
				File: &types.FileWithURI{URI: "https://example.com/image.jpg"},
			},
		},
	}

	// Should not call filter function for non-text parts
	_, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMiddleware_Integration(t *testing.T) {
	var buf strings.Builder
	logger := log.New(&buf, "", 0)

	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		// Verify all middleware effects
		requestID, ok := RequestIDFromContext(ctx)
		if !ok || requestID == "" {
			t.Error("RequestID not found in context")
		}

		metadata, ok := MetadataFromContext(ctx)
		if !ok || metadata["source"] != "test" {
			t.Error("Metadata not found in context")
		}

		return msg, nil
	}

	// Create chain with multiple middleware
	chain := NewChain(
		Recovery(),
		Logger(logger),
		RequestID(),
		Timer(),
		Validator(),
		Metadata(map[string]interface{}{"source": "test"}),
	)

	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	resp, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify response has metadata from all middleware
	if resp.Metadata == nil {
		t.Fatal("Response metadata is nil")
	}
	if resp.Metadata["processing_time_ms"] == "" {
		t.Error("Timer metadata not added")
	}
	if resp.Metadata["source"] != "test" {
		t.Error("Custom metadata not added")
	}

	// Verify logging occurred
	output := buf.String()
	if !strings.Contains(output, "test-123") {
		t.Error("Logger did not log message ID")
	}
}

func TestMiddleware_ErrorPropagation(t *testing.T) {
	expectedErr := fmt.Errorf("handler error")
	handlerCalled := false

	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		handlerCalled = true
		return nil, expectedErr
	}

	// Validator should catch error and propagate it
	chain := NewChain(
		Recovery(),
		Timer(),
		Validator(),
	)

	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	resp, err := chain.Execute(context.Background(), msg, handler)
	if err != expectedErr {
		t.Errorf("Execute() error = %v, want %v", err, expectedErr)
	}
	if resp != nil {
		t.Error("Response should be nil on error")
	}
	if !handlerCalled {
		t.Error("Handler should have been called")
	}
}
