# Multi-stage build for SAGE ADK Agent
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=v1.1.0" \
    -o /app/sage-adk-agent \
    ./cmd/adk

# Final stage - minimal image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 sage && \
    adduser -D -u 1000 -G sage sage

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/sage-adk-agent /app/sage-adk-agent

# Create necessary directories
RUN mkdir -p /app/data /tmp && \
    chown -R sage:sage /app /tmp

# Switch to non-root user
USER sage

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set entrypoint
ENTRYPOINT ["/app/sage-adk-agent"]
CMD ["serve", "--config", "/etc/sage-adk/config.yaml"]
