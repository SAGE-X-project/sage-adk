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

// Package middleware provides a middleware chain system for processing messages.
//
// Middleware allows you to intercept and process messages before they reach
// the main handler and after the handler returns. This is useful for:
//   - Logging and monitoring
//   - Authentication and authorization
//   - Rate limiting
//   - Request validation
//   - Error handling and recovery
//   - Adding metadata
//   - Timing and performance tracking
//
// Example:
//
//	// Create middleware chain
//	chain := middleware.NewChain(
//	    middleware.Recovery(),      // Recover from panics
//	    middleware.Logger(logger),  // Log requests
//	    middleware.Timer(),         // Track execution time
//	    middleware.Validator(),     // Validate messages
//	)
//
//	// Create handler
//	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
//	    // Process message
//	    return response, nil
//	}
//
//	// Execute with middleware
//	response, err := chain.Execute(ctx, message, handler)
//
// Custom Middleware:
//
//	// Create custom middleware
//	customMiddleware := func(next middleware.Handler) middleware.Handler {
//	    return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
//	        // Before handler
//	        log.Println("Before processing")
//
//	        // Call next middleware/handler
//	        resp, err := next(ctx, msg)
//
//	        // After handler
//	        log.Println("After processing")
//
//	        return resp, err
//	    }
//	}
//
//	chain.Use(customMiddleware)
package middleware
