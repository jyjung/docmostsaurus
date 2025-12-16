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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o docmostsaurus ./cmd/docmostsaurus

# Final stage
FROM alpine:3.19

# Install git, ca-certificates, and curl for health checks
RUN apk add --no-cache git ca-certificates tzdata curl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/docmostsaurus .

# Create directories for output and git repo with proper permissions
RUN mkdir -p /app/output /app/repo && \
    chmod 755 /app/output /app/repo

# Set default environment variables
ENV OUTPUT_DIR=/app/output
ENV SYNC_INTERVAL=1h
ENV HTTP_PORT=:8080
# Note: Lock file uses /tmp/docmostsaurus.lock (hardcoded)

# [Optional] Run as non-root user for better security
# To enable: uncomment the lines below and run on host:
#   sudo chown -R 1000:1000 ./output
# This ensures the mounted volume has correct permissions for UID 1000
#
# RUN adduser -D -u 1000 docmostsaurus && \
#     chown -R docmostsaurus:docmostsaurus /app /tmp
# USER docmostsaurus

# Expose health check port
EXPOSE 8080

# Health check configuration
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# SIGTERM for graceful shutdown
STOPSIGNAL SIGTERM

ENTRYPOINT ["/app/docmostsaurus"]
