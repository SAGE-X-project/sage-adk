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

// Package message provides message routing and processing functionality.
//
// The message package implements a router that:
//   - Routes messages to appropriate protocol adapters (A2A or SAGE)
//   - Manages middleware chains for message processing
//   - Supports automatic protocol detection
//   - Provides thread-safe message handling
//
// # Router
//
// The Router is the central component for message handling:
//
//	router := message.NewRouter(protocol.ProtocolAuto)
//
//	// Register protocol adapters
//	router.RegisterAdapter(a2aAdapter)
//	router.RegisterAdapter(sageAdapter)
//
//	// Add middleware
//	router.UseMiddleware(loggingMiddleware)
//	router.UseMiddleware(authMiddleware)
//
//	// Set handler
//	router.SetHandler(func(ctx context.Context, msg *types.Message) (*types.Message, error) {
//	    // Process message
//	    return response, nil
//	})
//
//	// Route message
//	response, err := router.Route(ctx, incomingMessage)
//
// # Protocol Selection
//
// The router supports three protocol modes:
//
//   - ProtocolAuto: Automatically detects protocol from message metadata
//   - ProtocolA2A: Always uses A2A protocol adapter
//   - ProtocolSAGE: Always uses SAGE protocol adapter
//
// # Middleware Integration
//
// The router integrates with the middleware package:
//
//	router.UseMiddleware(middleware.Logging())
//	router.UseMiddleware(middleware.RequestID())
//	router.UseMiddleware(customMiddleware)
//
// Middleware is executed in the order it's added, with the first middleware
// being the outermost layer.
//
// # Thread Safety
//
// The Router is thread-safe and can be used concurrently from multiple goroutines.
// All methods that access or modify router state are protected by internal mutexes.
//
// # Example
//
//	package main
//
//	import (
//	    "context"
//	    "github.com/sage-x-project/sage-adk/core/message"
//	    "github.com/sage-x-project/sage-adk/core/protocol"
//	    "github.com/sage-x-project/sage-adk/pkg/types"
//	)
//
//	func main() {
//	    // Create router with auto-detection
//	    router := message.NewRouter(protocol.ProtocolAuto)
//
//	    // Register adapters
//	    router.RegisterAdapter(a2aAdapter)
//	    router.RegisterAdapter(sageAdapter)
//
//	    // Set handler
//	    router.SetHandler(func(ctx context.Context, msg *types.Message) (*types.Message, error) {
//	        // Get protocol adapter from context
//	        adapter, _ := message.AdapterFromContext(ctx)
//	        log.Printf("Processing message with %s protocol", adapter.Name())
//
//	        // Process message
//	        response := types.NewMessage(types.MessageRoleAssistant, []types.Part{
//	            types.NewTextPart("Hello!"),
//	        })
//	        return response, nil
//	    })
//
//	    // Route incoming message
//	    msg := types.NewMessage(types.MessageRoleUser, []types.Part{
//	        types.NewTextPart("Hi"),
//	    })
//	    response, err := router.Route(context.Background(), msg)
//	}
package message
