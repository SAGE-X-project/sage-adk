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

package llm

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/sage-x-project/sage-adk/pkg/errors"
)

// MockProvider is a mock LLM provider for testing.
type MockProvider struct {
	name      string
	responses []string
	index     int
	mu        sync.Mutex
}

// NewMockProvider creates a new mock provider with pre-defined responses.
func NewMockProvider(name string, responses []string) *MockProvider {
	return &MockProvider{
		name:      name,
		responses: responses,
		index:     0,
	}
}

// Name returns the provider name.
func (m *MockProvider) Name() string {
	return m.name
}

// Complete generates a mock completion response.
func (m *MockProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.index >= len(m.responses) {
		return nil, errors.ErrLLMInvalidResponse.WithMessage("no more mock responses available")
	}

	content := m.responses[m.index]
	m.index++

	response := &CompletionResponse{
		ID:           "mock-" + uuid.New().String(),
		Model:        req.Model,
		Content:      content,
		FinishReason: "stop",
		Usage: &Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}

	return response, nil
}

// Stream is not implemented in Phase 1.
func (m *MockProvider) Stream(ctx context.Context, req *CompletionRequest, fn StreamFunc) error {
	return errors.ErrNotImplemented.WithMessage("streaming not implemented in mock provider")
}

// SupportsStreaming returns false for mock provider.
func (m *MockProvider) SupportsStreaming() bool {
	return false
}
