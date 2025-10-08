# Implementation Plan: PHASE 3 - Production Features

**Version:** 1.0
**Date:** 2025-10-08
**Status:** Planning
**Target Release:** v0.2.0

## Overview

This document provides a detailed implementation plan for PHASE 3 production features. It follows TDD principles with 90%+ test coverage goal.

---

## 1. Monitoring & Observability

### 1.1 Implementation Order

**Week 1: Core Metrics + Logging**
- Day 1-2: Package structure + Metrics interface
- Day 3-4: Prometheus integration + Agent metrics
- Day 5-6: Structured logging + Context propagation
- Day 7: Integration + Testing

**Week 2: Health Checks + Tracing**
- Day 1-2: Health check endpoints
- Day 3-4: OpenTelemetry integration
- Day 5-6: Span creation + Context propagation
- Day 7: Documentation + Examples

### 1.2 File Structure & Implementation

```
observability/
├── doc.go                          # Package documentation
├── config.go                       # Configuration types
├── config_test.go
│
├── metrics/
│   ├── doc.go
│   ├── collector.go                # Metrics collector interface
│   ├── collector_test.go
│   ├── prometheus.go               # Prometheus implementation
│   ├── prometheus_test.go
│   ├── agent.go                    # Agent-specific metrics
│   ├── agent_test.go
│   ├── llm.go                      # LLM metrics
│   ├── llm_test.go
│   ├── protocol.go                 # Protocol metrics
│   ├── protocol_test.go
│   ├── storage.go                  # Storage metrics
│   ├── storage_test.go
│   ├── middleware.go               # Middleware metrics
│   ├── middleware_test.go
│   └── resilience.go               # Resilience metrics
│       └── resilience_test.go
│
├── logging/
│   ├── doc.go
│   ├── logger.go                   # Logger interface
│   ├── logger_test.go
│   ├── structured.go               # JSON structured logger
│   ├── structured_test.go
│   ├── context.go                  # Context propagation
│   ├── context_test.go
│   ├── sampling.go                 # Log sampling
│   ├── sampling_test.go
│   ├── fields.go                   # Common log fields
│   └── adapters/
│       ├── zap.go                  # zap adapter
│       ├── zap_test.go
│       ├── zerolog.go              # zerolog adapter
│       └── zerolog_test.go
│
├── tracing/
│   ├── doc.go
│   ├── tracer.go                   # Tracer interface
│   ├── tracer_test.go
│   ├── otel.go                     # OpenTelemetry implementation
│   ├── otel_test.go
│   ├── context.go                  # Trace context helpers
│   ├── context_test.go
│   ├── spans.go                    # Span creation helpers
│   └── spans_test.go
│
└── health/
    ├── doc.go
    ├── checker.go                  # Health checker interface
    ├── checker_test.go
    ├── liveness.go                 # Liveness probe
    ├── liveness_test.go
    ├── readiness.go                # Readiness probe
    ├── readiness_test.go
    ├── startup.go                  # Startup probe
    ├── startup_test.go
    └── handler.go                  # HTTP handlers
        └── handler_test.go
```

### 1.3 Implementation Steps

#### Step 1: Metrics Collection (Day 1-2)

**1.1 Create metrics collector interface**
```go
// observability/metrics/collector.go
package metrics

import "context"

type Collector interface {
    // Counters
    IncrementCounter(name string, labels map[string]string)
    AddCounter(name string, value float64, labels map[string]string)

    // Gauges
    SetGauge(name string, value float64, labels map[string]string)

    // Histograms
    ObserveHistogram(name string, value float64, labels map[string]string)

    // Summary
    ObserveSummary(name string, value float64, labels map[string]string)
}

type Config struct {
    Enabled  bool
    Port     int
    Path     string
    Interval int
}
```

**1.2 Implement Prometheus collector**
```go
// observability/metrics/prometheus.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
)

type PrometheusCollector struct {
    registry  *prometheus.Registry
    counters  map[string]*prometheus.CounterVec
    gauges    map[string]*prometheus.GaugeVec
    histograms map[string]*prometheus.HistogramVec
    summaries map[string]*prometheus.SummaryVec
    mu        sync.RWMutex
}

func NewPrometheusCollector() *PrometheusCollector {
    return &PrometheusCollector{
        registry:   prometheus.NewRegistry(),
        counters:   make(map[string]*prometheus.CounterVec),
        gauges:     make(map[string]*prometheus.GaugeVec),
        histograms: make(map[string]*prometheus.HistogramVec),
        summaries:  make(map[string]*prometheus.SummaryVec),
    }
}

func (p *PrometheusCollector) IncrementCounter(name string, labels map[string]string) {
    counter := p.getOrCreateCounter(name, labels)
    counter.With(prometheus.Labels(labels)).Inc()
}

func (p *PrometheusCollector) Handler() http.Handler {
    return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{})
}
```

**1.3 Create agent metrics**
```go
// observability/metrics/agent.go
package metrics

const (
    MetricAgentStatus         = "sage_agent_status"
    MetricRequestsTotal       = "sage_agent_requests_total"
    MetricRequestDuration     = "sage_agent_request_duration_seconds"
    MetricErrorsTotal         = "sage_agent_errors_total"
    MetricActiveGoroutines    = "sage_agent_active_goroutines"
    MetricMemoryUsage         = "sage_agent_memory_bytes"
)

type AgentMetrics struct {
    collector Collector
}

func NewAgentMetrics(collector Collector) *AgentMetrics {
    return &AgentMetrics{collector: collector}
}

func (m *AgentMetrics) RecordRequest(agentID, protocol string, duration float64) {
    labels := map[string]string{
        "agent_id": agentID,
        "protocol": protocol,
    }
    m.collector.IncrementCounter(MetricRequestsTotal, labels)
    m.collector.ObserveHistogram(MetricRequestDuration, duration, labels)
}

func (m *AgentMetrics) RecordError(agentID, errorType string) {
    labels := map[string]string{
        "agent_id": agentID,
        "type":     errorType,
    }
    m.collector.IncrementCounter(MetricErrorsTotal, labels)
}
```

**1.4 Tests**
```go
// observability/metrics/collector_test.go
func TestPrometheusCollector(t *testing.T) {
    collector := NewPrometheusCollector()

    t.Run("increment counter", func(t *testing.T) {
        collector.IncrementCounter("test_counter", map[string]string{"label": "value"})
        // Verify counter incremented
    })

    t.Run("set gauge", func(t *testing.T) {
        collector.SetGauge("test_gauge", 42.0, map[string]string{"label": "value"})
        // Verify gauge value
    })

    t.Run("observe histogram", func(t *testing.T) {
        collector.ObserveHistogram("test_histogram", 0.5, map[string]string{"label": "value"})
        // Verify histogram recorded
    })
}
```

#### Step 2: Structured Logging (Day 3-4)

**2.1 Create logger interface**
```go
// observability/logging/logger.go
package logging

import "context"

type Level string

const (
    LevelDebug Level = "debug"
    LevelInfo  Level = "info"
    LevelWarn  Level = "warn"
    LevelError Level = "error"
    LevelFatal Level = "fatal"
)

type Logger interface {
    Debug(ctx context.Context, msg string, fields ...Field)
    Info(ctx context.Context, msg string, fields ...Field)
    Warn(ctx context.Context, msg string, fields ...Field)
    Error(ctx context.Context, msg string, fields ...Field)
    Fatal(ctx context.Context, msg string, fields ...Field)

    With(fields ...Field) Logger
}

type Field struct {
    Key   string
    Value interface{}
}

func String(key, value string) Field {
    return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
    return Field{Key: key, Value: value}
}

func Error(err error) Field {
    return Field{Key: "error", Value: err.Error()}
}
```

**2.2 Implement structured logger**
```go
// observability/logging/structured.go
package logging

import (
    "context"
    "encoding/json"
    "os"
    "time"
)

type StructuredLogger struct {
    level  Level
    output io.Writer
    fields []Field
}

func NewStructuredLogger(level Level) *StructuredLogger {
    return &StructuredLogger{
        level:  level,
        output: os.Stdout,
        fields: []Field{},
    }
}

func (l *StructuredLogger) Info(ctx context.Context, msg string, fields ...Field) {
    if !l.shouldLog(LevelInfo) {
        return
    }

    entry := l.buildEntry(ctx, LevelInfo, msg, fields...)
    l.write(entry)
}

func (l *StructuredLogger) buildEntry(ctx context.Context, level Level, msg string, fields ...Field) map[string]interface{} {
    entry := map[string]interface{}{
        "timestamp": time.Now().UTC().Format(time.RFC3339Nano),
        "level":     string(level),
        "message":   msg,
    }

    // Add context fields
    if reqID := ctx.Value("request_id"); reqID != nil {
        entry["request_id"] = reqID
    }
    if traceID := ctx.Value("trace_id"); traceID != nil {
        entry["trace_id"] = traceID
    }

    // Add logger fields
    for _, f := range l.fields {
        entry[f.Key] = f.Value
    }

    // Add message fields
    for _, f := range fields {
        entry[f.Key] = f.Value
    }

    return entry
}

func (l *StructuredLogger) write(entry map[string]interface{}) {
    data, _ := json.Marshal(entry)
    l.output.Write(append(data, '\n'))
}
```

**2.3 Context propagation**
```go
// observability/logging/context.go
package logging

import "context"

type contextKey string

const (
    requestIDKey contextKey = "request_id"
    traceIDKey   contextKey = "trace_id"
    agentIDKey   contextKey = "agent_id"
)

func WithRequestID(ctx context.Context, requestID string) context.Context {
    return context.WithValue(ctx, requestIDKey, requestID)
}

func GetRequestID(ctx context.Context) string {
    if v := ctx.Value(requestIDKey); v != nil {
        return v.(string)
    }
    return ""
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
    return context.WithValue(ctx, traceIDKey, traceID)
}
```

#### Step 3: Health Checks (Day 1-2, Week 2)

**3.1 Health checker interface**
```go
// observability/health/checker.go
package health

import "context"

type Status string

const (
    StatusHealthy   Status = "healthy"
    StatusUnhealthy Status = "unhealthy"
    StatusDegraded  Status = "degraded"
)

type CheckResult struct {
    Name    string                 `json:"name"`
    Status  Status                 `json:"status"`
    Message string                 `json:"message,omitempty"`
    Details map[string]interface{} `json:"details,omitempty"`
}

type Checker interface {
    Name() string
    Check(ctx context.Context) CheckResult
}
```

**3.2 Liveness probe**
```go
// observability/health/liveness.go
package health

import "context"

type LivenessChecker struct {
    agentRunning bool
}

func NewLivenessChecker() *LivenessChecker {
    return &LivenessChecker{agentRunning: true}
}

func (c *LivenessChecker) Name() string {
    return "liveness"
}

func (c *LivenessChecker) Check(ctx context.Context) CheckResult {
    if c.agentRunning {
        return CheckResult{
            Name:   c.Name(),
            Status: StatusHealthy,
        }
    }
    return CheckResult{
        Name:    c.Name(),
        Status:  StatusUnhealthy,
        Message: "agent not running",
    }
}
```

**3.3 Readiness probe**
```go
// observability/health/readiness.go
package health

import (
    "context"
    "time"
)

type ReadinessChecker struct {
    checks []Checker
}

func NewReadinessChecker(checks ...Checker) *ReadinessChecker {
    return &ReadinessChecker{checks: checks}
}

func (c *ReadinessChecker) Name() string {
    return "readiness"
}

func (c *ReadinessChecker) Check(ctx context.Context) CheckResult {
    results := make([]CheckResult, 0, len(c.checks))

    for _, check := range c.checks {
        result := check.Check(ctx)
        results = append(results, result)

        if result.Status == StatusUnhealthy {
            return CheckResult{
                Name:    c.Name(),
                Status:  StatusUnhealthy,
                Message: "dependency unhealthy",
                Details: map[string]interface{}{
                    "checks": results,
                },
            }
        }
    }

    return CheckResult{
        Name:   c.Name(),
        Status: StatusHealthy,
        Details: map[string]interface{}{
            "checks": results,
        },
    }
}
```

#### Step 4: Integration with Agent (Day 7, Week 1 + Week 2)

**4.1 Add to agent builder**
```go
// builder/builder.go

func (b *AgentBuilder) WithObservability(config *observability.Config) *AgentBuilder {
    b.observabilityConfig = config
    return b
}

func (b *AgentBuilder) Build() (*agent.Agent, error) {
    // ... existing code ...

    // Setup observability
    if b.observabilityConfig != nil && b.observabilityConfig.Metrics.Enabled {
        collector := metrics.NewPrometheusCollector()
        agentMetrics := metrics.NewAgentMetrics(collector)
        // Inject into agent
    }

    if b.observabilityConfig != nil && b.observabilityConfig.Logging.Enabled {
        logger := logging.NewStructuredLogger(logging.LevelInfo)
        // Inject into agent
    }
}
```

**4.2 Middleware integration**
```go
// core/middleware/observability.go
package middleware

import (
    "github.com/sage-x-project/sage-adk/observability/metrics"
    "github.com/sage-x-project/sage-adk/observability/logging"
    "time"
)

func Observability(metrics *metrics.AgentMetrics, logger logging.Logger) Middleware {
    return func(next MessageHandler) MessageHandler {
        return func(ctx context.Context, msg MessageContext) error {
            start := time.Now()

            logger.Info(ctx, "message received",
                logging.String("message_id", msg.ID()),
                logging.String("protocol", msg.Protocol()),
            )

            err := next(ctx, msg)

            duration := time.Since(start).Seconds()
            metrics.RecordRequest(msg.AgentID(), msg.Protocol(), duration)

            if err != nil {
                logger.Error(ctx, "message handling failed",
                    logging.String("message_id", msg.ID()),
                    logging.Error(err),
                )
                metrics.RecordError(msg.AgentID(), "handler_error")
            }

            return err
        }
    }
}
```

### 1.4 Test Plan

**Unit Tests:**
- ✅ Metrics collector: counter, gauge, histogram, summary
- ✅ Logger: all log levels, field handling, context propagation
- ✅ Health checks: liveness, readiness, startup
- ✅ Tracer: span creation, context propagation

**Integration Tests:**
- ✅ Prometheus scraping
- ✅ Log output verification
- ✅ Health check endpoints
- ✅ End-to-end tracing

**Performance Tests:**
- ✅ Metrics overhead < 1ms
- ✅ Logging overhead < 500μs
- ✅ No memory leaks

---

## 2. Rate Limiting

### 2.1 Implementation Order

**Week 1: Core Limiter**
- Day 1-2: Package structure + Interface + Types
- Day 3-4: Token bucket strategy + Tests
- Day 5-6: Storage backends (memory, Redis)
- Day 7: Integration + Testing

**Week 2: Middleware + Advanced**
- Day 1-2: Rate limit middleware
- Day 3-4: Fixed window + Sliding window strategies
- Day 5-6: PostgreSQL storage + LLM-specific limits
- Day 7: Documentation + Examples

### 2.2 File Structure

```
ratelimit/
├── doc.go
├── limiter.go                      # Rate limiter interface
├── limiter_test.go
├── types.go                        # Config & result types
├── types_test.go
├── errors.go
│
├── strategies/
│   ├── token_bucket.go
│   ├── token_bucket_test.go
│   ├── fixed_window.go
│   ├── fixed_window_test.go
│   ├── sliding_window.go
│   ├── sliding_window_test.go
│   ├── leaky_bucket.go
│   └── leaky_bucket_test.go
│
├── storage/
│   ├── interface.go                # Storage interface
│   ├── memory.go                   # In-memory storage
│   ├── memory_test.go
│   ├── redis.go                    # Redis storage
│   ├── redis_test.go
│   ├── postgres.go                 # PostgreSQL storage
│   └── postgres_test.go
│
├── middleware.go                   # Rate limit middleware
└── middleware_test.go
```

### 2.3 Implementation Steps

#### Step 1: Core Interface (Day 1-2)

**1.1 Rate limiter interface**
```go
// ratelimit/limiter.go
package ratelimit

import (
    "context"
    "time"
)

type Limiter interface {
    Allow(ctx context.Context, key string, cost int) (*Result, error)
    Reset(ctx context.Context, key string) error
}

type Result struct {
    Allowed   bool
    Limit     int
    Remaining int
    ResetAt   time.Time
    RetryAfter time.Duration
}

type Config struct {
    Strategy string                       // token_bucket, fixed_window, sliding_window
    Storage  string                       // memory, redis, postgres
    Global   Rule                         // Global rate limit
    PerUser  Rule                         // Per-user rate limit
    LLM      map[string]Rule              // LLM provider limits
}

type Rule struct {
    RequestsPerSecond int
    RequestsPerMinute int
    RequestsPerHour   int
    RequestsPerDay    int
    TokensPerMinute   int
    TokensPerHour     int
    MaxConcurrent     int
}
```

#### Step 2: Token Bucket Strategy (Day 3-4)

**2.1 Implementation**
```go
// ratelimit/strategies/token_bucket.go
package strategies

import (
    "context"
    "time"
    "sync"
)

type TokenBucket struct {
    storage     Storage
    capacity    int           // Maximum tokens
    refillRate  float64       // Tokens per second
    refillInterval time.Duration
}

func NewTokenBucket(storage Storage, capacity int, refillRate float64) *TokenBucket {
    return &TokenBucket{
        storage:    storage,
        capacity:   capacity,
        refillRate: refillRate,
        refillInterval: time.Second,
    }
}

func (tb *TokenBucket) Allow(ctx context.Context, key string, cost int) (*Result, error) {
    bucket, err := tb.storage.Get(ctx, key)
    if err != nil {
        // Initialize new bucket
        bucket = &Bucket{
            Tokens:   tb.capacity,
            LastRefill: time.Now(),
        }
    }

    // Refill tokens
    now := time.Now()
    elapsed := now.Sub(bucket.LastRefill).Seconds()
    tokensToAdd := int(elapsed * tb.refillRate)
    bucket.Tokens = min(bucket.Tokens + tokensToAdd, tb.capacity)
    bucket.LastRefill = now

    // Check if request allowed
    allowed := bucket.Tokens >= cost
    if allowed {
        bucket.Tokens -= cost
    }

    // Save bucket state
    tb.storage.Set(ctx, key, bucket)

    result := &Result{
        Allowed:   allowed,
        Limit:     tb.capacity,
        Remaining: bucket.Tokens,
    }

    if !allowed {
        result.RetryAfter = time.Duration(float64(cost-bucket.Tokens)/tb.refillRate) * time.Second
        result.ResetAt = now.Add(result.RetryAfter)
    }

    return result, nil
}

type Bucket struct {
    Tokens     int
    LastRefill time.Time
}
```

**2.2 Tests**
```go
// ratelimit/strategies/token_bucket_test.go
func TestTokenBucket(t *testing.T) {
    storage := NewMemoryStorage()
    limiter := NewTokenBucket(storage, 10, 1.0) // 10 tokens, 1/sec refill

    t.Run("allow under limit", func(t *testing.T) {
        result, err := limiter.Allow(context.Background(), "user1", 5)
        assert.NoError(t, err)
        assert.True(t, result.Allowed)
        assert.Equal(t, 5, result.Remaining)
    })

    t.Run("deny over limit", func(t *testing.T) {
        result, err := limiter.Allow(context.Background(), "user1", 10)
        assert.NoError(t, err)
        assert.False(t, result.Allowed)
        assert.Greater(t, result.RetryAfter, time.Duration(0))
    })

    t.Run("refill over time", func(t *testing.T) {
        time.Sleep(2 * time.Second)
        result, err := limiter.Allow(context.Background(), "user1", 2)
        assert.NoError(t, err)
        assert.True(t, result.Allowed)
    })
}
```

#### Step 3: Storage Backends (Day 5-6)

**3.1 Memory storage**
```go
// ratelimit/storage/memory.go
package storage

import (
    "context"
    "sync"
)

type MemoryStorage struct {
    data map[string]*Bucket
    mu   sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
    return &MemoryStorage{
        data: make(map[string]*Bucket),
    }
}

func (s *MemoryStorage) Get(ctx context.Context, key string) (*Bucket, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    bucket, ok := s.data[key]
    if !ok {
        return nil, ErrNotFound
    }
    return bucket, nil
}

func (s *MemoryStorage) Set(ctx context.Context, key string, bucket *Bucket) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.data[key] = bucket
    return nil
}
```

**3.2 Redis storage**
```go
// ratelimit/storage/redis.go
package storage

import (
    "context"
    "encoding/json"
    "github.com/redis/go-redis/v9"
    "time"
)

type RedisStorage struct {
    client *redis.Client
    ttl    time.Duration
}

func NewRedisStorage(client *redis.Client, ttl time.Duration) *RedisStorage {
    return &RedisStorage{
        client: client,
        ttl:    ttl,
    }
}

func (s *RedisStorage) Get(ctx context.Context, key string) (*Bucket, error) {
    data, err := s.client.Get(ctx, "ratelimit:"+key).Bytes()
    if err == redis.Nil {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, err
    }

    var bucket Bucket
    if err := json.Unmarshal(data, &bucket); err != nil {
        return nil, err
    }

    return &bucket, nil
}

func (s *RedisStorage) Set(ctx context.Context, key string, bucket *Bucket) error {
    data, err := json.Marshal(bucket)
    if err != nil {
        return err
    }

    return s.client.Set(ctx, "ratelimit:"+key, data, s.ttl).Err()
}
```

#### Step 4: Middleware (Day 1-2, Week 2)

**4.1 Rate limit middleware**
```go
// ratelimit/middleware.go
package ratelimit

import (
    "context"
    "fmt"
    "net/http"
    "github.com/sage-x-project/sage-adk/core/agent"
)

func Middleware(limiter Limiter, keyFunc func(agent.MessageContext) string) agent.Middleware {
    return func(next agent.MessageHandler) agent.MessageHandler {
        return func(ctx context.Context, msg agent.MessageContext) error {
            key := keyFunc(msg)

            result, err := limiter.Allow(ctx, key, 1)
            if err != nil {
                return fmt.Errorf("rate limit check failed: %w", err)
            }

            // Add rate limit headers
            msg.SetResponseHeader("X-RateLimit-Limit", fmt.Sprintf("%d", result.Limit))
            msg.SetResponseHeader("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
            msg.SetResponseHeader("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetAt.Unix()))

            if !result.Allowed {
                msg.SetResponseHeader("Retry-After", fmt.Sprintf("%d", int(result.RetryAfter.Seconds())))
                return &RateLimitError{
                    Message:    "rate limit exceeded",
                    RetryAfter: result.RetryAfter,
                    Limit:      result.Limit,
                    Remaining:  result.Remaining,
                    Reset:      result.ResetAt,
                }
            }

            return next(ctx, msg)
        }
    }
}

// Key extraction functions
func KeyByUserID(msg agent.MessageContext) string {
    return "user:" + msg.UserID()
}

func KeyByAgentID(msg agent.MessageContext) string {
    return "agent:" + msg.AgentID()
}

func KeyGlobal(msg agent.MessageContext) string {
    return "global"
}
```

### 2.4 Test Plan

**Unit Tests:**
- ✅ Token bucket: allow, deny, refill
- ✅ Fixed window: boundary conditions
- ✅ Sliding window: accuracy
- ✅ Storage: get, set, expiration

**Integration Tests:**
- ✅ Middleware: rate limit enforcement
- ✅ Redis: distributed rate limiting
- ✅ Multi-user: isolation

**Load Tests:**
- ✅ 10,000 RPS sustained
- ✅ <100μs latency

---

## 3. Advanced Error Recovery

### 3.1 Implementation Order

**Week 1: Enhanced Retry + Fallback**
- Day 1-2: Enhance retry with jitter, budget
- Day 3-4: LLM fallback chains
- Day 5-6: Error classifier + Strategy selector
- Day 7: Integration + Testing

**Week 2: DLQ + Circuit Breaker**
- Day 1-2: Dead letter queue
- Day 3-4: Per-dependency circuit breakers
- Day 5-6: Storage fallbacks
- Day 7: Documentation + Examples

### 3.2 File Structure

```
core/resilience/               # Enhance existing
├── retry.go                   # Add jitter, budget
├── circuit_breaker.go         # Per-dependency support
├── fallback.go                # NEW: Fallback strategies
├── fallback_test.go
├── dlq.go                     # NEW: Dead letter queue
└── dlq_test.go

core/recovery/                 # NEW: Advanced recovery
├── doc.go
├── classifier.go              # Error classification
├── classifier_test.go
├── strategy.go                # Recovery strategy
├── strategy_test.go
├── llm_fallback.go            # LLM-specific
├── llm_fallback_test.go
├── storage_fallback.go        # Storage-specific
└── storage_fallback_test.go
```

### 3.3 Implementation Steps

#### Step 1: Enhance Retry (Day 1-2)

**1.1 Add jitter to existing retry**
```go
// core/resilience/retry.go (enhance existing)

type RetryConfig struct {
    // Existing fields...
    MaxAttempts      int
    InitialDelay     time.Duration
    MaxDelay         time.Duration

    // NEW fields
    Jitter           bool    // Add randomization
    BackoffMultiplier float64 // Growth rate (default 2.0)
    RetryBudget      int     // Max retries per time window
}

func (r *Retry) calculateBackoff(attempt int) time.Duration {
    delay := r.config.InitialDelay * time.Duration(math.Pow(r.config.BackoffMultiplier, float64(attempt)))

    if delay > r.config.MaxDelay {
        delay = r.config.MaxDelay
    }

    if r.config.Jitter {
        // Add ±25% jitter
        jitter := time.Duration(rand.Float64() * 0.5 * float64(delay))
        if rand.Intn(2) == 0 {
            delay += jitter
        } else {
            delay -= jitter
        }
    }

    return delay
}
```

#### Step 2: LLM Fallback (Day 3-4)

**2.1 Fallback chain**
```go
// core/recovery/llm_fallback.go
package recovery

import (
    "context"
    "github.com/sage-x-project/sage-adk/adapters/llm"
)

type LLMFallback struct {
    providers []llm.Provider
    strategy  FallbackStrategy
}

type FallbackStrategy string

const (
    StrategySequential FallbackStrategy = "sequential" // Try in order
    StrategyCostAware  FallbackStrategy = "cost_aware" // Cheapest first
    StrategyQualityAware FallbackStrategy = "quality_aware" // Best quality first
)

func NewLLMFallback(providers []llm.Provider, strategy FallbackStrategy) *LLMFallback {
    return &LLMFallback{
        providers: providers,
        strategy:  strategy,
    }
}

func (f *LLMFallback) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
    providers := f.orderProviders()

    var lastErr error
    for i, provider := range providers {
        resp, err := provider.Complete(ctx, req)
        if err == nil {
            return resp, nil
        }

        lastErr = err

        // Check if error is retryable
        if !isRetryable(err) {
            return nil, err
        }

        // Log fallback
        log.Warn("LLM fallback", "provider", i, "error", err)
    }

    return nil, fmt.Errorf("all LLM providers failed: %w", lastErr)
}

func (f *LLMFallback) orderProviders() []llm.Provider {
    switch f.strategy {
    case StrategyCostAware:
        return f.sortByCost()
    case StrategyQualityAware:
        return f.sortByQuality()
    default:
        return f.providers
    }
}
```

#### Step 3: Dead Letter Queue (Day 1-2, Week 2)

**3.1 DLQ implementation**
```go
// core/resilience/dlq.go
package resilience

import (
    "context"
    "time"
)

type DeadLetterQueue interface {
    Add(ctx context.Context, msg *FailedMessage) error
    List(ctx context.Context, limit int) ([]*FailedMessage, error)
    Retry(ctx context.Context, msgID string) error
    Delete(ctx context.Context, msgID string) error
    Clear(ctx context.Context) error
}

type FailedMessage struct {
    ID          string
    Message     interface{}
    Error       string
    Attempts    int
    FirstFailed time.Time
    LastFailed  time.Time
    Metadata    map[string]string
}

type MemoryDLQ struct {
    messages map[string]*FailedMessage
    maxSize  int
    mu       sync.RWMutex
}

func NewMemoryDLQ(maxSize int) *MemoryDLQ {
    return &MemoryDLQ{
        messages: make(map[string]*FailedMessage),
        maxSize:  maxSize,
    }
}

func (dlq *MemoryDLQ) Add(ctx context.Context, msg *FailedMessage) error {
    dlq.mu.Lock()
    defer dlq.mu.Unlock()

    if len(dlq.messages) >= dlq.maxSize {
        return ErrDLQFull
    }

    dlq.messages[msg.ID] = msg
    return nil
}
```

### 3.4 Test Plan

**Unit Tests:**
- ✅ Retry with jitter: randomization
- ✅ LLM fallback: sequential, cost-aware, quality-aware
- ✅ DLQ: add, list, retry, delete
- ✅ Error classifier: transient, permanent, critical

**Integration Tests:**
- ✅ End-to-end fallback chain
- ✅ DLQ auto-retry
- ✅ Circuit breaker + fallback

**Chaos Tests:**
- ✅ Random LLM failures
- ✅ Network partitions
- ✅ Storage corruption

---

## 4. Performance Tuning

### 4.1 Implementation Order

**Week 1: Caching + Connection Pooling**
- Day 1-2: Response caching
- Day 3-4: DID resolution caching
- Day 5-6: Connection pool tuning
- Day 7: Testing + Benchmarking

**Week 2: Profiling + Optimization**
- Day 1-2: Built-in profiling endpoints
- Day 3-4: Worker pool for concurrency
- Day 5-6: Memory optimization
- Day 7: Benchmark suite

### 4.2 Implementation Steps

#### Step 1: Response Caching (Day 1-2)

```go
// performance/cache/response.go
package cache

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "time"
)

type ResponseCache struct {
    cache Cache
    ttl   time.Duration
}

func NewResponseCache(cache Cache, ttl time.Duration) *ResponseCache {
    return &ResponseCache{cache: cache, ttl: ttl}
}

func (c *ResponseCache) Get(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, bool) {
    key := c.cacheKey(req)
    value, err := c.cache.Get(ctx, key)
    if err != nil {
        return nil, false
    }

    resp, ok := value.(*llm.CompletionResponse)
    return resp, ok
}

func (c *ResponseCache) Set(ctx context.Context, req *llm.CompletionRequest, resp *llm.CompletionResponse) error {
    key := c.cacheKey(req)
    return c.cache.Set(ctx, key, resp, c.ttl)
}

func (c *ResponseCache) cacheKey(req *llm.CompletionRequest) string {
    h := sha256.New()
    h.Write([]byte(req.Model))
    h.Write([]byte(fmt.Sprintf("%.2f", req.Temperature)))
    for _, msg := range req.Messages {
        h.Write([]byte(msg.Role))
        h.Write([]byte(msg.Content))
    }
    return hex.EncodeToString(h.Sum(nil))
}
```

#### Step 2: Profiling (Day 1-2, Week 2)

```go
// performance/profiling/handler.go
package profiling

import (
    "net/http"
    "net/http/pprof"
    "runtime"
)

func RegisterHandlers(mux *http.ServeMux) {
    mux.HandleFunc("/debug/pprof/", pprof.Index)
    mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
    mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
    mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
    mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

    // Custom handlers
    mux.HandleFunc("/debug/stats", statsHandler)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)

    stats := map[string]interface{}{
        "goroutines":     runtime.NumGoroutine(),
        "memory_alloc":   m.Alloc,
        "memory_total":   m.TotalAlloc,
        "memory_sys":     m.Sys,
        "gc_runs":        m.NumGC,
    }

    json.NewEncoder(w).Encode(stats)
}
```

### 4.3 Benchmarks

```go
// performance/benchmarks/agent_test.go
func BenchmarkMessageHandling(b *testing.B) {
    agent := setupTestAgent()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        msg := createTestMessage()
        agent.HandleMessage(context.Background(), msg)
    }
}

func BenchmarkLLMComplete(b *testing.B) {
    provider := llm.NewMockProvider()

    req := &llm.CompletionRequest{
        Messages: []llm.Message{
            {Role: llm.RoleUser, Content: "Hello"},
        },
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        provider.Complete(context.Background(), req)
    }
}
```

---

## 5. Production Deployment Guide

### 5.1 Implementation Order

**Week 1: Core Documentation**
- Day 1-2: Kubernetes deployment guide
- Day 3-4: Docker guide + Dockerfile
- Day 5-6: Configuration guide
- Day 7: AWS/GCP guides

**Week 2: Operations + Best Practices**
- Day 1-2: Monitoring setup guides
- Day 3-4: Security guide
- Day 5-6: Troubleshooting guide
- Day 7: Review + Publish

### 5.2 Document Outline

```markdown
# Production Deployment Guide

## 1. Architecture Patterns
### 1.1 Single Instance
### 1.2 Multi-Instance (Load Balanced)
### 1.3 Distributed (Microservices)
### 1.4 Serverless

## 2. Kubernetes Deployment
### 2.1 Prerequisites
### 2.2 Deployment Manifest
### 2.3 Service Definition
### 2.4 ConfigMap & Secrets
### 2.5 Horizontal Pod Autoscaler
### 2.6 Ingress

## 3. Docker
### 3.1 Dockerfile
### 3.2 Multi-stage Build
### 3.3 Image Optimization
### 3.4 Docker Compose

## 4. Cloud Providers
### 4.1 AWS
- ECS
- EKS
- Lambda
### 4.2 GCP
- GKE
- Cloud Run
- Functions
### 4.3 Azure
- AKS

## 5. Configuration
### 5.1 Environment Variables
### 5.2 Config Files
### 5.3 Secrets Management

## 6. Monitoring
### 6.1 Prometheus Setup
### 6.2 Grafana Dashboards
### 6.3 Logging (ELK/Loki)
### 6.4 Tracing (Jaeger)

## 7. Security
### 7.1 TLS/SSL
### 7.2 Authentication
### 7.3 Network Security

## 8. Backup & DR
### 8.1 Backup Strategy
### 8.2 Disaster Recovery

## 9. Troubleshooting
### 9.1 Common Issues
### 9.2 Debugging Guide
```

---

## Testing Strategy

### Unit Tests
- **Target**: 90%+ coverage
- **Tools**: Go test, testify
- **CI**: Run on every commit

### Integration Tests
- **Target**: Critical paths covered
- **Tools**: Docker Compose for dependencies
- **CI**: Run on PR

### E2E Tests
- **Target**: Happy paths + critical errors
- **Tools**: Custom test harness
- **CI**: Run nightly

### Performance Tests
- **Target**: Benchmarks for all critical paths
- **Tools**: Go benchmarks, pprof
- **CI**: Track regressions

### Load Tests
- **Target**: 10,000 RPS sustained
- **Tools**: k6, vegeta
- **CI**: Run weekly

---

## CI/CD Pipeline

```yaml
# .github/workflows/phase3.yml
name: Phase 3 CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run tests
        run: make test

      - name: Run integration tests
        run: make test-integration

      - name: Check coverage
        run: make test-coverage

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: golangci/golangci-lint-action@v3

  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run benchmarks
        run: make benchmark
```

---

## Risk Mitigation

### Technical Risks

**Risk**: Prometheus memory usage
- **Mitigation**: Metric cardinality limits, retention policies
- **Owner**: DevOps

**Risk**: OpenTelemetry overhead
- **Mitigation**: Sampling, async export
- **Owner**: Performance team

**Risk**: Rate limit storage failure
- **Mitigation**: Fallback to in-memory, circuit breaker
- **Owner**: Reliability team

### Schedule Risks

**Risk**: Testing takes longer than expected
- **Mitigation**: Parallelize tests, automated test generation
- **Buffer**: +2 weeks

**Risk**: Integration complexity
- **Mitigation**: Incremental integration, feature flags
- **Buffer**: +1 week

---

## Success Metrics

### Development Metrics
- ✅ Code coverage ≥ 90%
- ✅ All tests passing
- ✅ No critical security issues
- ✅ Documentation complete

### Performance Metrics
- ✅ Throughput: 10,000 RPS
- ✅ Latency: p95 < 100ms
- ✅ Memory: < 500MB
- ✅ CPU: < 50% at 5K RPS

### Operational Metrics
- ✅ MTTR < 1 hour
- ✅ Deployment time < 5 min
- ✅ Zero-downtime deployments
- ✅ 99.9% uptime

---

## Next Steps

1. **Week 1**: Start Phase 3A - Monitoring & Observability
2. **Week 3**: Start Rate Limiting
3. **Week 5**: Start Advanced Error Recovery
4. **Week 7**: Start Performance Tuning
5. **Week 9**: Start Deployment Guide
6. **Week 10**: Release v0.2.0
