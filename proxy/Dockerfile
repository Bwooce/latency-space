# proxy/Dockerfile
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev linux-headers

# Set working directory
WORKDIR /app/src

# Copy go mod files
COPY src/go.mod .
COPY src/go.sum .

# Download dependencies
RUN go mod download

# Copy source code
COPY src/ .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /latency-proxy

# Create final image
FROM alpine:latest

# Install tc and other network tools
RUN apk add --no-cache iproute2 iptables

# Copy binary from builder
COPY --from=builder /latency-proxy /usr/local/bin/latency-proxy

# Make it executable
RUN chmod +x /usr/local/bin/latency-proxy

# Create config directory
RUN mkdir -p /etc/latency-space

# Run the proxy
CMD ["/usr/local/bin/latency-proxy"]

