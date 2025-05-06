# proxy/Dockerfile

# Builder Stage - Build the Go application
# Use Go 1.24 on Alpine
FROM golang:1.24-alpine AS builder

# Install build dependencies (minimal)
RUN apk add --no-cache gcc musl-dev git

# Set working directory
WORKDIR /app

# Copy shared directory (assuming it's copied into context beforehand)
COPY shared /shared

# Copy go.mod and go.sum from the proxy directory (current context)
COPY go.mod go.sum ./
# Download dependencies
RUN go mod download

# Copy source code from the proxy directory (current context) AFTER go mod download
COPY src ./src
# Removed template copy from builder stage

# Set build environment variables
ENV CGO_ENABLED=0 GOOS=linux
# Build the application
# Build targeting the main package within the copied src directory
RUN go build -mod=mod -o /latency-proxy ./src/main.go

# Use specific Alpine version for final image
FROM alpine:3.19.1

# Install required tools in a single layer
RUN apk add --no-cache \
    iproute2=~6.6 \
    iptables=~1.8 \
    openssl=~3.1 \
    && mkdir -p /etc/latency-space

 # Copy the built binary from the builder stage
 COPY --from=builder /latency-proxy /usr/local/bin/latency-proxy
 # Copy templates from the builder stage (from the copied src directory)
 COPY --from=builder /app/src/templates /app/templates
 # Copy shared data from the builder stage
 COPY --from=builder /shared /shared
RUN chmod +x /usr/local/bin/latency-proxy

 # Expose necessary ports
EXPOSE 80 443 1080 9090

CMD ["/usr/local/bin/latency-proxy"]