# Rate Limiting Example

This example demonstrates advanced rate limiting strategies in SAGE ADK.

## Features

### 1. Token Bucket Algorithm
- Smooth rate limiting with burst support
- Configurable rate and capacity
- Automatic token refill
- Per-key tracking

```go
limiter := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
    Rate:     100,  // 100 requests per second
    Capacity: 200,  // Allow bursts up to 200
})

if limiter.Allow("user-123") {
    // Process request
}
```

### 2. Sliding Window Algorithm
- Precise time-based rate limiting
- No boundary issues
- Accurate request counting
- Window expiration

```go
limiter := ratelimit.NewSlidingWindow(ratelimit.SlidingWindowConfig{
    Limit:  1000,           // 1000 requests
    Window: time.Minute,    // Per minute
})

if limiter.Allow("user-456") {
    // Process request
}
```

### 3. Middleware Integration
- Easy integration with agent handlers
- Per-user, per-context, or global limits
- Custom key functions
- Error handling

```go
middleware := ratelimit.NewTokenBucketMiddleware(
    ratelimit.TokenBucketConfig{
        Rate:     10.0,
        Capacity: 20,
    },
    ratelimit.PerUserKeyFunc,
)

rateLimitedHandler := middleware(handler)
```

### 4. Distributed Rate Limiting (requires Redis)
- Rate limiting across multiple instances
- Shared state using Redis
- Multiple algorithms (sliding window, fixed window)
- Automatic cleanup

```go
limiter, _ := ratelimit.NewDistributed(ratelimit.DistributedConfig{
    RedisClient: redisClient,
    Limit:       1000,
    Window:      time.Minute,
    Algorithm:   ratelimit.AlgorithmSlidingWindow,
})
```

## Use Cases

### API Rate Limiting
```go
// Limit API calls per user
middleware := ratelimit.NewTokenBucketMiddleware(
    ratelimit.TokenBucketConfig{
        Rate:     100,  // 100 req/sec
        Capacity: 200,  // Allow bursts
    },
    func(ctx context.Context, msg *types.Message) string {
        return fmt.Sprintf("user:%s", msg.Metadata["user_id"])
    },
)
```

### Abuse Prevention
```go
// Strict limits for suspicious activity
limiter := ratelimit.NewSlidingWindow(ratelimit.SlidingWindowConfig{
    Limit:  10,              // Only 10 requests
    Window: time.Minute,     // Per minute
})
```

### Burst Traffic Handling
```go
// Allow bursts but maintain average rate
limiter := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
    Rate:     50,   // Average 50 req/sec
    Capacity: 500,  // But allow bursts up to 500
})
```

### Fair Resource Allocation
```go
// Per-user limits ensure fair sharing
middleware := ratelimit.NewSlidingWindowMiddleware(
    ratelimit.SlidingWindowConfig{
        Limit:  1000,
        Window: time.Hour,
    },
    ratelimit.PerUserKeyFunc,
)
```

## Running the Example

```bash
# Run the basic example
go run main.go

# Run with Redis for distributed rate limiting
docker run -d -p 6379:6379 redis:alpine
go run main.go
```

## Performance

The rate limiters are optimized for high throughput:

- **Token Bucket**: ~2-3M ops/sec (single key)
- **Sliding Window**: ~1-2M ops/sec (single key)
- **Distributed**: ~10-50K ops/sec (network latency)

Benchmarks:
```bash
cd ../../ratelimit
go test -bench=. -benchmem
```

## Configuration Best Practices

### Token Bucket
- Use for smooth rate limiting
- Set capacity higher than rate for burst tolerance
- Good for APIs with variable load

### Sliding Window
- Use for precise time-based limits
- More accurate but slightly slower
- Good for strict quotas

### Distributed
- Use when running multiple instances
- Requires Redis or similar store
- Slightly higher latency

## Monitoring

All limiters provide statistics:

```go
stats := limiter.Stats()
fmt.Printf("Allowed: %d\n", stats.Allowed)
fmt.Printf("Denied: %d\n", stats.Denied)
fmt.Printf("Hit Rate: %.2f%%\n",
    float64(stats.Denied)/float64(stats.Allowed+stats.Denied)*100)
```

## Error Handling

```go
middleware := ratelimit.NewMiddleware(ratelimit.MiddlewareConfig{
    Limiter: limiter,
    KeyFunc: ratelimit.PerUserKeyFunc,
    OnRateLimitExceeded: func(ctx context.Context, msg *types.Message, key string) (*types.Message, error) {
        // Custom handling
        return types.NewMessage(
            types.MessageRoleAssistant,
            []types.Part{
                types.NewTextPart("Rate limit exceeded. Please try again later."),
            },
        ), nil
    },
})
```

## See Also

- [Token Bucket Algorithm](https://en.wikipedia.org/wiki/Token_bucket)
- [Sliding Window Counter](https://hechao.li/2018/06/25/Rate-Limiter-Part1/)
- [Distributed Rate Limiting](https://redis.io/docs/manual/patterns/rate-limiter/)
