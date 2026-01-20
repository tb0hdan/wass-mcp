# Stage 1: Build
FROM golang:1.25-bookworm AS builder

WORKDIR /build

# Copy go mod files first for better layer caching
COPY ../go.mod go.sum ./
RUN go mod download

# Copy source code
COPY .. .

# Build the binary with optimizations
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w" \
    -o wass-mcp \
    ./cmd/wass-mcp/*.go

# Install Nuclei in the builder stage
RUN go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest

# Stage 2: Runtime
FROM debian:bookworm-slim

# Enable contrib and non-free repositories, install security scanning tools
RUN sed -i 's/^Components: main$/Components: main contrib non-free/' /etc/apt/sources.list.d/debian.sources \
    && apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    nikto \
    wapiti \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Create non-root user with home directory (wapiti needs writable home)
RUN useradd -r -u 1000 -m -s /sbin/nologin wass

# Create data directory for SQLite database
RUN mkdir -p /data && chown wass:wass /data

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/wass-mcp .
COPY --from=builder /go/bin/nuclei /usr/local/bin/nuclei

# Set ownership
RUN nuclei -update-templates
RUN chown wass:wass /app/wass-mcp

USER wass

# Expose default port
EXPOSE 8989

# Default database location
ENV WASS_DB_PATH=/data/wass-mcp.db

CMD ["/app/wass-mcp", "--bind", "0.0.0.0:8989", "--db", "/data/wass-mcp.db"]
