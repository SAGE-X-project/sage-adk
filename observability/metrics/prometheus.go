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
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusCollector implements the Collector interface using Prometheus.
type PrometheusCollector struct {
	registry   *prometheus.Registry
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
	summaries  map[string]*prometheus.SummaryVec
	mu         sync.RWMutex
}

// NewPrometheusCollector creates a new Prometheus metrics collector.
func NewPrometheusCollector() *PrometheusCollector {
	return &PrometheusCollector{
		registry:   prometheus.NewRegistry(),
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		summaries:  make(map[string]*prometheus.SummaryVec),
	}
}

// IncrementCounter increments a counter metric by 1.
func (p *PrometheusCollector) IncrementCounter(name string, labels map[string]string) {
	p.AddCounter(name, 1, labels)
}

// AddCounter adds a value to a counter metric.
func (p *PrometheusCollector) AddCounter(name string, value float64, labels map[string]string) {
	counter := p.getOrCreateCounter(name, labels)
	counter.With(prometheus.Labels(labels)).Add(value)
}

// SetGauge sets a gauge metric to a specific value.
func (p *PrometheusCollector) SetGauge(name string, value float64, labels map[string]string) {
	gauge := p.getOrCreateGauge(name, labels)
	gauge.With(prometheus.Labels(labels)).Set(value)
}

// ObserveHistogram observes a value for a histogram metric.
func (p *PrometheusCollector) ObserveHistogram(name string, value float64, labels map[string]string) {
	histogram := p.getOrCreateHistogram(name, labels)
	histogram.With(prometheus.Labels(labels)).Observe(value)
}

// ObserveSummary observes a value for a summary metric.
func (p *PrometheusCollector) ObserveSummary(name string, value float64, labels map[string]string) {
	summary := p.getOrCreateSummary(name, labels)
	summary.With(prometheus.Labels(labels)).Observe(value)
}

// Handler returns an HTTP handler for exposing metrics.
func (p *PrometheusCollector) Handler() http.Handler {
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// getOrCreateCounter gets or creates a counter metric.
func (p *PrometheusCollector) getOrCreateCounter(name string, labels map[string]string) *prometheus.CounterVec {
	p.mu.RLock()
	counter, exists := p.counters[name]
	p.mu.RUnlock()

	if exists {
		return counter
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if counter, exists = p.counters[name]; exists {
		return counter
	}

	labelNames := p.getLabelNames(labels)
	counter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: "Auto-generated counter metric: " + name,
		},
		labelNames,
	)

	p.registry.MustRegister(counter)
	p.counters[name] = counter

	return counter
}

// getOrCreateGauge gets or creates a gauge metric.
func (p *PrometheusCollector) getOrCreateGauge(name string, labels map[string]string) *prometheus.GaugeVec {
	p.mu.RLock()
	gauge, exists := p.gauges[name]
	p.mu.RUnlock()

	if exists {
		return gauge
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if gauge, exists = p.gauges[name]; exists {
		return gauge
	}

	labelNames := p.getLabelNames(labels)
	gauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: "Auto-generated gauge metric: " + name,
		},
		labelNames,
	)

	p.registry.MustRegister(gauge)
	p.gauges[name] = gauge

	return gauge
}

// getOrCreateHistogram gets or creates a histogram metric.
func (p *PrometheusCollector) getOrCreateHistogram(name string, labels map[string]string) *prometheus.HistogramVec {
	p.mu.RLock()
	histogram, exists := p.histograms[name]
	p.mu.RUnlock()

	if exists {
		return histogram
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if histogram, exists = p.histograms[name]; exists {
		return histogram
	}

	labelNames := p.getLabelNames(labels)
	histogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    name,
			Help:    "Auto-generated histogram metric: " + name,
			Buckets: prometheus.DefBuckets,
		},
		labelNames,
	)

	p.registry.MustRegister(histogram)
	p.histograms[name] = histogram

	return histogram
}

// getOrCreateSummary gets or creates a summary metric.
func (p *PrometheusCollector) getOrCreateSummary(name string, labels map[string]string) *prometheus.SummaryVec {
	p.mu.RLock()
	summary, exists := p.summaries[name]
	p.mu.RUnlock()

	if exists {
		return summary
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if summary, exists = p.summaries[name]; exists {
		return summary
	}

	labelNames := p.getLabelNames(labels)
	summary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       name,
			Help:       "Auto-generated summary metric: " + name,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		labelNames,
	)

	p.registry.MustRegister(summary)
	p.summaries[name] = summary

	return summary
}

// getLabelNames extracts label names from a labels map.
func (p *PrometheusCollector) getLabelNames(labels map[string]string) []string {
	if len(labels) == 0 {
		return []string{}
	}

	names := make([]string, 0, len(labels))
	for name := range labels {
		names = append(names, name)
	}

	return names
}
