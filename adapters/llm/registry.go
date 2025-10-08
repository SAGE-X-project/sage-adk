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
	"sync"

	"github.com/sage-x-project/sage-adk/pkg/errors"
)

// Registry manages LLM providers.
type Registry struct {
	mu              sync.RWMutex
	providers       map[string]Provider
	defaultProvider Provider
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register registers a provider with the given name.
func (r *Registry) Register(name string, provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[name] = provider
}

// Get retrieves a provider by name.
func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[name]
	if !ok {
		return nil, errors.ErrNotFound.WithDetail("provider", name)
	}

	return provider, nil
}

// SetDefault sets the default provider.
func (r *Registry) SetDefault(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.defaultProvider = provider
}

// Default returns the default provider.
func (r *Registry) Default() Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.defaultProvider
}
