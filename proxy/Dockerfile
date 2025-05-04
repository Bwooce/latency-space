# proxy/Dockerfile

# Use specific golang Alpine version for builder
FROM golang:1.21.6-alpine3.19 AS builder

WORKDIR /app

# Install only essential build dependencies
RUN apk add --no-cache \
    gcc=~13.2 \
    musl-dev=~1.2

# Copy shared package first
COPY src/shared /app/shared

# Copy only go.mod and go.sum for better caching
COPY src/go.mod src/go.sum /app/src/

# Prepare vendor directory structure with our shared package
WORKDIR /app/src
RUN mkdir -p vendor/github.com/latency-space/shared && \
    cp -r ../shared/* vendor/github.com/latency-space/shared/

# Set environment variables for hermetic builds
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOFLAGS="-mod=vendor"

# Copy the source code (Go will use the vendored dependencies)
COPY src/ .

# Build with vendored dependencies
RUN go build -o /latency-proxy

# Use specific Alpine version for final image
FROM alpine:3.19.1

# Install required tools in a single layer
RUN apk add --no-cache \
    iproute2=~6.6 \
    iptables=~1.8 \
    openssl=~3.1 \
    && mkdir -p /etc/latency-space

# Copy and prepare binary
COPY --from=builder /latency-proxy /usr/local/bin/latency-proxy
RUN chmod +x /usr/local/bin/latency-proxy

# Copy templates
COPY src/templates/ /app/templates/

# Expose ports
EXPOSE 80 443 1080 9090

CMD ["/usr/local/bin/latency-proxy"]