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
	"net/http"
)

// Collector is the interface for metrics collection.
type Collector interface {
	// IncrementCounter increments a counter metric by 1
	IncrementCounter(name string, labels map[string]string)

	// AddCounter adds a value to a counter metric
	AddCounter(name string, value float64, labels map[string]string)

	// SetGauge sets a gauge metric to a specific value
	SetGauge(name string, value float64, labels map[string]string)

	// ObserveHistogram observes a value for a histogram metric
	ObserveHistogram(name string, value float64, labels map[string]string)

	// ObserveSummary observes a value for a summary metric
	ObserveSummary(name string, value float64, labels map[string]string)

	// Handler returns an HTTP handler for exposing metrics
	Handler() http.Handler
}

// Labels is a convenience type for metric labels.
type Labels map[string]string

// NoLabels returns an empty label map.
func NoLabels() Labels {
	return Labels{}
}

// NewLabels creates a new label map.
func NewLabels(keyvals ...string) Labels {
	if len(keyvals)%2 != 0 {
		panic("NewLabels requires an even number of arguments")
	}

	labels := make(Labels)
	for i := 0; i < len(keyvals); i += 2 {
		labels[keyvals[i]] = keyvals[i+1]
	}
	return labels
}

// With adds a label to the label map.
func (l Labels) With(key, value string) Labels {
	newLabels := make(Labels, len(l)+1)
	for k, v := range l {
		newLabels[k] = v
	}
	newLabels[key] = value
	return newLabels
}
