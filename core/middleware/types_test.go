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
	"testing"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

func TestNewChain(t *testing.T) {
	chain := NewChain()
	if chain == nil {
		t.Fatal("NewChain() returned nil")
	}
	if chain.Len() != 0 {
		t.Errorf("Len() = %d, want 0", chain.Len())
	}
}

func TestNewChainWithMiddleware(t *testing.T) {
	mw1 := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			return next(ctx, msg)
		}
	}
	mw2 := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			return next(ctx, msg)
		}
	}

	chain := NewChain(mw1, mw2)
	if chain.Len() != 2 {
		t.Errorf("Len() = %d, want 2", chain.Len())
	}
}

func TestChain_Use(t *testing.T) {
	chain := NewChain()
	mw := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			return next(ctx, msg)
		}
	}

	result := chain.Use(mw)
	if result != chain {
		t.Error("Use() should return the chain for fluent API")
	}
	if chain.Len() != 1 {
		t.Errorf("Len() = %d, want 1", chain.Len())
	}
}

func TestChain_Then(t *testing.T) {
	var executionOrder []string

	mw1 := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			executionOrder = append(executionOrder, "mw1-before")
			resp, err := next(ctx, msg)
			executionOrder = append(executionOrder, "mw1-after")
			return resp, err
		}
	}

	mw2 := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			executionOrder = append(executionOrder, "mw2-before")
			resp, err := next(ctx, msg)
			executionOrder = append(executionOrder, "mw2-after")
			return resp, err
		}
	}

	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		executionOrder = append(executionOrder, "handler")
		return msg, nil
	}

	chain := NewChain(mw1, mw2)
	wrappedHandler := chain.Then(handler)

	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	_, err := wrappedHandler(context.Background(), msg)
	if err != nil {
		t.Fatalf("Handler execution failed: %v", err)
	}

	expectedOrder := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(executionOrder) != len(expectedOrder) {
		t.Fatalf("Execution order length = %d, want %d", len(executionOrder), len(expectedOrder))
	}

	for i, expected := range expectedOrder {
		if executionOrder[i] != expected {
			t.Errorf("executionOrder[%d] = %s, want %s", i, executionOrder[i], expected)
		}
	}
}

func TestChain_Execute(t *testing.T) {
	called := false
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		called = true
		return msg, nil
	}

	chain := NewChain()
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
		t.Error("Execute() returned nil response")
	}
	if !called {
		t.Error("Handler was not called")
	}
}

func TestChain_ExecuteWithError(t *testing.T) {
	expectedErr := errors.New("handler error")
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return nil, expectedErr
	}

	chain := NewChain()
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
		t.Error("Execute() should return nil response on error")
	}
}

func TestChain_MiddlewareCanModifyMessage(t *testing.T) {
	modifyingMw := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			// Add metadata before handler
			if msg.Metadata == nil {
				msg.Metadata = make(map[string]interface{})
			}
			msg.Metadata["modified"] = "true"
			return next(ctx, msg)
		}
	}

	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	chain := NewChain(modifyingMw)
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	resp, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resp.Metadata["modified"] != "true" {
		t.Error("Middleware did not modify message metadata")
	}
}

func TestChain_MiddlewareCanShortCircuit(t *testing.T) {
	handlerCalled := false

	shortCircuitMw := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			// Don't call next, return immediately
			return &types.Message{
				MessageID: "short-circuit",
				Role:      types.MessageRoleAgent,
				Parts:     []types.Part{&types.TextPart{Text: "short-circuited"}},
			}, nil
		}
	}

	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		handlerCalled = true
		return msg, nil
	}

	chain := NewChain(shortCircuitMw)
	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	resp, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if handlerCalled {
		t.Error("Handler should not be called when middleware short-circuits")
	}
	if resp.MessageID != "short-circuit" {
		t.Error("Response should be from short-circuit middleware")
	}
}

func TestContextWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "req-12345"

	ctx = ContextWithRequestID(ctx, requestID)
	retrievedID, ok := RequestIDFromContext(ctx)

	if !ok {
		t.Error("RequestIDFromContext() returned false")
	}
	if retrievedID != requestID {
		t.Errorf("RequestIDFromContext() = %s, want %s", retrievedID, requestID)
	}
}

func TestRequestIDFromContext_NotSet(t *testing.T) {
	ctx := context.Background()
	_, ok := RequestIDFromContext(ctx)

	if ok {
		t.Error("RequestIDFromContext() should return false when request ID not set")
	}
}

func TestContextWithMetadata(t *testing.T) {
	ctx := context.Background()
	metadata := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	ctx = ContextWithMetadata(ctx, metadata)
	retrievedMetadata, ok := MetadataFromContext(ctx)

	if !ok {
		t.Error("MetadataFromContext() returned false")
	}
	if len(retrievedMetadata) != 2 {
		t.Errorf("MetadataFromContext() length = %d, want 2", len(retrievedMetadata))
	}
	if retrievedMetadata["key1"] != "value1" {
		t.Errorf("Metadata[key1] = %v, want value1", retrievedMetadata["key1"])
	}
	if retrievedMetadata["key2"] != 42 {
		t.Errorf("Metadata[key2] = %v, want 42", retrievedMetadata["key2"])
	}
}

func TestMetadataFromContext_NotSet(t *testing.T) {
	ctx := context.Background()
	_, ok := MetadataFromContext(ctx)

	if ok {
		t.Error("MetadataFromContext() should return false when metadata not set")
	}
}

func TestChain_EmptyChain(t *testing.T) {
	chain := NewChain()
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	msg := &types.Message{
		MessageID: "test-123",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Text: "test"}},
	}

	resp, err := chain.Execute(context.Background(), msg, handler)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if resp.MessageID != msg.MessageID {
		t.Error("Empty chain should pass through message unchanged")
	}
}
