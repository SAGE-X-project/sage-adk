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

package builder

import (
	"fmt"

	"github.com/sage-x-project/sage-adk/core/protocol"
)

// validator validates builder configuration.
type validator struct {
	builder *Builder
	errors  []error
}

// addError adds a validation error.
func (v *validator) addError(err error) {
	v.errors = append(v.errors, err)
}

// validateName validates agent name.
func (v *validator) validateName() {
	if v.builder.name == "" {
		v.addError(fmt.Errorf("agent name cannot be empty"))
		return
	}

	// Agent name should be alphanumeric with hyphens/underscores
	for _, c := range v.builder.name {
		if !isValidNameChar(c) {
			v.addError(fmt.Errorf("agent name contains invalid character: %c (use only a-z, 0-9, -, _)", c))
			return
		}
	}

	// Length check
	if len(v.builder.name) > 64 {
		v.addError(fmt.Errorf("agent name too long (max 64 characters): %s", v.builder.name))
	}
}

// validateProtocol validates protocol configuration.
func (v *validator) validateProtocol() {
	switch v.builder.protocolMode {
	case protocol.ProtocolA2A:
		// A2A always valid (has defaults)
		return

	case protocol.ProtocolSAGE:
		// SAGE requires configuration
		if v.builder.sageConfig == nil {
			v.addError(fmt.Errorf("SAGE mode requires SAGEConfig (use WithSAGEConfig)"))
			return
		}

		// Validate SAGE config
		if v.builder.sageConfig.DID == "" {
			v.addError(fmt.Errorf("SAGE config requires DID"))
		}
		if v.builder.sageConfig.Network == "" {
			v.addError(fmt.Errorf("SAGE config requires Network"))
		}
		if v.builder.sageConfig.RPCEndpoint == "" {
			v.addError(fmt.Errorf("SAGE config requires RPCEndpoint"))
		}

	case protocol.ProtocolAuto:
		// Auto mode is valid (falls back to A2A if SAGE not configured)
		return

	default:
		v.addError(fmt.Errorf("invalid protocol mode: %v", v.builder.protocolMode))
	}
}

// validateLLM validates LLM configuration.
func (v *validator) validateLLM() {
	// LLM is optional (agent can work without LLM for pure routing)
	if v.builder.llmProvider == nil {
		// No error, just a warning (could log this)
		return
	}

	// Validate LLM provider name
	if v.builder.llmProvider.Name() == "" {
		v.addError(fmt.Errorf("LLM provider has no name"))
	}
}

// validateStorage validates storage configuration.
func (v *validator) validateStorage() {
	// Storage is required (but has default)
	if v.builder.storageBackend == nil {
		v.addError(fmt.Errorf("storage backend is required"))
	}
}

// validateHandler validates message handler.
func (v *validator) validateHandler() {
	// Handler is required (but has default echo handler)
	if v.builder.messageHandler == nil {
		v.addError(fmt.Errorf("message handler is required"))
	}
}

// isValidNameChar checks if a character is valid in agent name.
func isValidNameChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}
