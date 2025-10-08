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

package metrics

import (
	"testing"
)

func TestNoLabels(t *testing.T) {
	labels := NoLabels()
	if labels == nil {
		t.Error("NoLabels() should not return nil")
	}
	if len(labels) != 0 {
		t.Errorf("NoLabels() should return empty map, got %d labels", len(labels))
	}
}

func TestNewLabels(t *testing.T) {
	tests := []struct {
		name     string
		keyvals  []string
		expected map[string]string
		panics   bool
	}{
		{
			name:     "empty labels",
			keyvals:  []string{},
			expected: map[string]string{},
		},
		{
			name:    "single label",
			keyvals: []string{"key", "value"},
			expected: map[string]string{
				"key": "value",
			},
		},
		{
			name:    "multiple labels",
			keyvals: []string{"key1", "value1", "key2", "value2"},
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:    "odd number of args",
			keyvals: []string{"key"},
			panics:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panics {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic but did not panic")
					}
				}()
			}

			labels := NewLabels(tt.keyvals...)

			if !tt.panics {
				if len(labels) != len(tt.expected) {
					t.Errorf("expected %d labels, got %d", len(tt.expected), len(labels))
				}

				for k, v := range tt.expected {
					if labels[k] != v {
						t.Errorf("expected label %s=%s, got %s", k, v, labels[k])
					}
				}
			}
		})
	}
}

func TestLabelsWith(t *testing.T) {
	labels := NewLabels("key1", "value1")
	newLabels := labels.With("key2", "value2")

	// Original should be unchanged
	if len(labels) != 1 {
		t.Errorf("original labels should have 1 entry, got %d", len(labels))
	}

	// New labels should have both
	if len(newLabels) != 2 {
		t.Errorf("new labels should have 2 entries, got %d", len(newLabels))
	}

	if newLabels["key1"] != "value1" {
		t.Error("key1 should be copied to new labels")
	}

	if newLabels["key2"] != "value2" {
		t.Error("key2 should be added to new labels")
	}
}
