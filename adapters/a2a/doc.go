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

// Package a2a provides the A2A (Agent-to-Agent) protocol adapter for SAGE ADK.
//
// This package wraps the sage-a2a-go library to provide A2A protocol support
// within the SAGE ADK framework. It implements the ProtocolAdapter interface
// and handles type conversions between sage-adk and sage-a2a-go types.
//
// # Configuration
//
// The A2A adapter requires configuration:
//
//	a2a:
//	  server_url: "http://localhost:8080/"  # Required
//	  timeout: 30                           # Optional (seconds)
//	  user_agent: "sage-adk/0.1.0"         # Optional
//
// # Usage
//
//	cfg := &config.A2AConfig{
//	    ServerURL: "http://localhost:8080/",
//	    Timeout:   30,
//	}
//
//	adapter, err := a2a.NewAdapter(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	msg := types.NewMessage(
//	    types.MessageRoleUser,
//	    []types.Part{types.NewTextPart("Hello")},
//	)
//
//	err = adapter.SendMessage(context.Background(), &msg)
//
// # Type Conversion
//
// The adapter automatically converts between sage-adk and sage-a2a-go types:
//
//   - types.Message <-> protocol.Message
//   - types.Part <-> protocol.Part (TextPart, FilePart, DataPart)
//   - types.FileWithBytes <-> protocol.FileWithBytes (base64 encoding)
//   - types.FileWithURI <-> protocol.FileWithURI
//
// # Limitations
//
//   - ReceiveMessage() is not supported (A2A uses request-response model)
//   - Streaming support is planned but not yet implemented
//   - Task-based workflows are not yet supported (message-only)
package a2a
