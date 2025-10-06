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

// Package sage provides the SAGE (Secure Agent Guarantee Engine) protocol adapter.
//
// This package wraps the sage library to provide SAGE protocol support with
// full security features including DID-based authentication, cryptographic
// signing, and RFC-9421 compliant message verification.
//
// # Phase 1 Implementation
//
// The current implementation (Phase 1) focuses on basic adapter structure and
// verification capabilities. Full message sending and receiving will be added
// in future phases when the transport layer is implemented.
//
// Implemented:
//   - Basic adapter initialization
//   - Configuration loading
//   - Message verification (basic validation)
//   - Protocol interface compliance
//
// Not Implemented (Future):
//   - Message sending (requires transport layer)
//   - Message receiving (requires transport layer)
//   - Streaming support
//   - Full signature verification
//   - DID registration and management
//
// # Configuration
//
// The SAGE adapter requires configuration:
//
//	sage:
//	  enabled: true
//	  network: "ethereum"           # or "kaia", "sepolia"
//	  did: "did:sage:eth:0x..."     # Agent's DID (required)
//	  key_path: "/path/to/keys"     # Path to key storage
//	  registry_contract: "0x..."    # DID registry contract address
//
// # Usage
//
//	cfg := &config.SAGEConfig{
//	    Enabled: true,
//	    Network: "ethereum",
//	    DID:     "did:sage:eth:0x1234567890abcdef",
//	}
//
//	adapter, err := sage.NewAdapter(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Verify a message
//	msg := &types.Message{
//	    Security: &types.SecurityMetadata{
//	        Mode:     types.ProtocolModeSAGE,
//	        AgentDID: "did:sage:eth:0x...",
//	    },
//	}
//
//	err = adapter.Verify(context.Background(), msg)
//
// # Security Features
//
// SAGE provides enhanced security compared to A2A:
//
//   - DID-based Authentication: Agents identified by blockchain DIDs
//   - Cryptographic Signatures: All messages signed with private keys
//   - On-chain Verification: DID resolution from blockchain
//   - RFC-9421 Compliance: HTTP message signature standard
//   - Key Rotation: Support for key lifecycle management
//
// # Design Principles
//
// Based on AI agent development research:
//
//   - Security First: All SAGE security guarantees maintained
//   - Progressive Disclosure: Complex security made simple
//   - Zero Trust: Verify all messages by default
//   - Adapter Pattern: Wrap sage library without reimplementation
//
// # Limitations (Phase 1)
//
//   - SendMessage() returns ErrNotImplemented (no transport layer)
//   - ReceiveMessage() returns ErrNotImplemented (no transport layer)
//   - Verify() performs basic validation only (full verification in Phase 2)
//   - Streaming not supported
//   - DID management not included (assumes pre-registered DID)
//
// # Future Enhancements
//
// Phase 2:
//   - Full signature verification using SAGE core
//   - Message serialization and signing
//   - DID resolution from blockchain
//
// Phase 3:
//   - gRPC or HTTP transport layer
//   - Message sending and receiving
//   - Streaming support
//
// Phase 4:
//   - DID registration and lifecycle management
//   - Key rotation
//   - Multi-chain support
//   - SAGE handshake protocol
package sage
