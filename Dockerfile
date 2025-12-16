FROM golang:1.21-alpine AS builder

# Install git for go-git
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o docmost-sync ./cmd/docmost-file-sync

# Final stage
FROM alpine:3.19

# Install git, ca-certificates, and curl for health checks
RUN apk add --no-cache git ca-certificates tzdata curl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/docmost-sync .

# Create directories for output and git repo
RUN mkdir -p /app/output /app/repo

# Set default environment variables
ENV OUTPUT_DIR=/app/output
ENV SYNC_INTERVAL=1h
ENV HTTP_PORT=:8080
# Note: Lock file uses /tmp/docmostsaurus.lock (hardcoded)

# Run as non-root user
RUN adduser -D -u 1000 docmost-sync
RUN chown -R docmost-sync:docmost-sync /app
USER docmost-sync

# Expose health check port
EXPOSE 8080

# Health check configuration
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# SIGTERM for graceful shutdown
STOPSIGNAL SIGTERM

ENTRYPOINT ["/app/docmost-sync"]
