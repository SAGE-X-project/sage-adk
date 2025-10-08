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

	"github.com/sage-x-project/sage-adk/pkg/types"
)

// Handler is the final handler that processes a message.
type Handler func(ctx context.Context, msg *types.Message) (*types.Message, error)

// Middleware is a function that wraps a Handler.
type Middleware func(Handler) Handler

// Chain manages a chain of middleware.
type Chain struct {
	middlewares []Middleware
}

// NewChain creates a new middleware chain.
func NewChain(middlewares ...Middleware) *Chain {
	return &Chain{
		middlewares: middlewares,
	}
}

// Use adds middleware to the chain.
func (c *Chain) Use(mw Middleware) *Chain {
	c.middlewares = append(c.middlewares, mw)
	return c
}

// Then wraps a handler with all middleware in the chain.
func (c *Chain) Then(h Handler) Handler {
	// Apply middleware in reverse order so the first middleware
	// added is the outermost (executes first)
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		h = c.middlewares[i](h)
	}
	return h
}

// Execute runs the handler with the middleware chain.
func (c *Chain) Execute(ctx context.Context, msg *types.Message, h Handler) (*types.Message, error) {
	handler := c.Then(h)
	return handler(ctx, msg)
}

// Len returns the number of middleware in the chain.
func (c *Chain) Len() int {
	return len(c.middlewares)
}

// Context keys for middleware data
type contextKey string

const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey contextKey = "request_id"

	// StartTimeKey is the context key for request start time.
	StartTimeKey contextKey = "start_time"

	// MetadataKey is the context key for metadata.
	MetadataKey contextKey = "metadata"
)

// ContextWithRequestID adds a request ID to the context.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// RequestIDFromContext retrieves the request ID from the context.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(RequestIDKey).(string)
	return requestID, ok
}

// ContextWithMetadata adds metadata to the context.
func ContextWithMetadata(ctx context.Context, metadata map[string]interface{}) context.Context {
	return context.WithValue(ctx, MetadataKey, metadata)
}

// MetadataFromContext retrieves metadata from the context.
func MetadataFromContext(ctx context.Context) (map[string]interface{}, bool) {
	metadata, ok := ctx.Value(MetadataKey).(map[string]interface{})
	return metadata, ok
}
