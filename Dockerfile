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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o doc2git ./cmd/doc2git

# Final stage
FROM alpine:3.19

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/doc2git .

# Create directories for output and git repo
RUN mkdir -p /app/output /app/repo

# Set default environment variables
ENV OUTPUT_DIR=/app/output
ENV GIT_REPO_PATH=/app/repo
ENV GIT_BRANCH=main
ENV SYNC_INTERVAL=1h

# Run as non-root user
RUN adduser -D -u 1000 doc2git
RUN chown -R doc2git:doc2git /app
USER doc2git

ENTRYPOINT ["/app/doc2git"]
