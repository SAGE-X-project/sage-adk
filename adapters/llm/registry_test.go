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
	"testing"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("NewRegistry() should not return nil")
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("test", []string{"response"})

	registry.Register("test", provider)

	got, err := registry.Get("test")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Name() != "test" {
		t.Errorf("Provider name = %v, want test", got.Name())
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Get("nonexistent")
	if err == nil {
		t.Error("Get() should return error for nonexistent provider")
	}
}

func TestRegistry_SetDefault(t *testing.T) {
	registry := NewRegistry()
	provider := NewMockProvider("default", []string{"response"})

	registry.SetDefault(provider)

	got := registry.Default()
	if got == nil {
		t.Fatal("Default() should not return nil")
	}

	if got.Name() != "default" {
		t.Errorf("Default provider name = %v, want default", got.Name())
	}
}

func TestRegistry_Default_Nil(t *testing.T) {
	registry := NewRegistry()

	got := registry.Default()
	if got != nil {
		t.Error("Default() should return nil when not set")
	}
}
