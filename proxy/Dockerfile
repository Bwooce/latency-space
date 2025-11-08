# proxy/Dockerfile

# Use a specific golang Alpine version
FROM golang:1.21.6-alpine3.19 AS builder

# Install build dependencies (minimal)
RUN apk add --no-cache \
    gcc=~13.2 \
    musl-dev=~1.2 \
    git=~2.43

# Set working directory
WORKDIR /app

# Copy shared directory
COPY shared/ /shared/

# Copy source code files
COPY proxy/src /app
COPY proxy/src/templates /app/templates

# Build with no vendor directory and fixed dependencies
ENV CGO_ENABLED=0 \
    GOOS=linux
RUN go mod download && \
    go build -mod=mod -o latency-proxy

# Use specific Alpine version for final image
FROM alpine:3.19.1

# Install required tools in a single layer
RUN apk add --no-cache \
    iproute2=~6.6 \
    iptables=~1.8 \
    openssl=~3.1 \
    libc-utils=~0.7 \
    && mkdir -p /etc/latency-space

# Copy the binary and templates
COPY --from=builder /app/latency-proxy /usr/local/bin/latency-proxy
COPY --from=builder /app/templates/ /app/templates/
RUN chmod +x /usr/local/bin/latency-proxy

# Expose ports
EXPOSE 80 443 1080 9090

CMD ["/usr/local/bin/latency-proxy"]