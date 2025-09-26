# Build stage
FROM golang:1.24-alpine AS builder

# Accept build argument for image tag
ARG IMAGE_TAG=""

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code and required files (excluding build artifacts and logs via .dockerignore)
COPY . /app/

# Build the application
RUN if [ -f ".version" ]; then \
        VERSION=$(cat .version | xargs); \
        BUILD="docker-$(date +%Y%m%d-%H%M%S)"; \
    else \
        VERSION="1.0.0"; \
        BUILD="docker-$(date +%Y%m%d-%H%M%S)"; \
    fi && \
    [ -z "$VERSION" ] && VERSION="1.0.0"; \
    [ -z "$BUILD" ] && BUILD="docker"; \
    COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
    CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-w -s -X aktis-collector-jira/internal/common.Version=$VERSION -X 'aktis-collector-jira/internal/common.Build=$BUILD' -X aktis-collector-jira/internal/common.GitCommit=$COMMIT" \
        -o bin/aktis-collector-jira ./cmd/aktis-collector-jira

# Final stage
FROM alpine:latest

# Accept build argument for image tag in final stage
ARG IMAGE_TAG=""

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 appuser && \
    adduser -D -u 1001 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/aktis-collector-jira .

# Copy configuration example
COPY --from=builder /app/internal/config.example.json ./config.example.json

# Create runtime directories with proper permissions
RUN mkdir -p data logs backups && \
    chown -R appuser:appuser /app && \
    chmod -R 755 /app

# Switch to non-root user
USER appuser

# Set environment variables for Docker deployment
ENV DOCKER_DEPLOYMENT=true
ENV IMAGE_TAG=${IMAGE_TAG}

# Create volume mount points for persistent data
VOLUME ["/app/data", "/app/logs", "/app/backups"]

# Health check - collector doesn't have HTTP endpoints, so just check if process is running
HEALTHCHECK --interval=60s --timeout=10s --start-period=10s --retries=2 \
    CMD pgrep aktis-collector-jira > /dev/null || exit 1

# Default command (can be overridden)
CMD ["./aktis-collector-jira", "-config", "/app/config.json"]