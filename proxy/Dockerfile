# proxy/Dockerfile

# Stage 1: Copier - Capture the full project context including 'shared'
FROM alpine:3.19.1 AS copier
WORKDIR /source
# This COPY command assumes the docker build command is run from the project root
# It copies the entire project content into the /source directory of this stage
COPY . .

# Stage 2: Builder - Build the Go application
# Use Go 1.24 on Alpine
FROM golang:1.24-alpine AS builder

# Install build dependencies (minimal)
RUN apk add --no-cache gcc musl-dev git

# Set working directory
WORKDIR /app

# Copy the 'shared' directory from the 'copier' stage first
COPY --from=copier /source/shared /shared

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
# Change working directory to the source directory
WORKDIR /app/src
# Build targeting the current directory (.)
RUN go build -mod=mod -o /latency-proxy .
# Change back to the parent directory
WORKDIR /app

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