# Multi-stage Dockerfile for SAGE ADK

# Stage 1: Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -o /build/bin/sage-adk \
    ./cmd/adk

# Stage 2: Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 adk && \
    adduser -D -u 1000 -G adk adk

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/bin/sage-adk /app/sage-adk

# Copy configuration examples
COPY --from=builder /build/.env.example /app/.env.example
COPY --from=builder /build/config.yaml.example /app/config.yaml.example

# Change ownership
RUN chown -R adk:adk /app

# Switch to non-root user
USER adk

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set entrypoint
ENTRYPOINT ["/app/sage-adk"]

# Default command
CMD ["serve"]
