# proxy/Dockerfile

# Stage 1: Copier - Capture the full project context including 'shared'
FROM alpine:3.19.1 AS copier
WORKDIR /source
# This COPY command assumes the docker build command is run from the project root
# It copies the entire project content into the /source directory of this stage
COPY . .

# Stage 2: Builder - Build the Go application
# Use a specific golang Alpine version
FROM golang:1.21.6-alpine3.19 AS builder

# Install build dependencies (minimal)
RUN apk add --no-cache gcc musl-dev git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first from the proxy directory (current context)
COPY go.mod go.sum ./
# Download dependencies
RUN go mod download

# Copy source code from the proxy directory (current context) AFTER go mod download
COPY src ./src
# Removed template copy from builder stage

# Copy the 'shared' directory from the 'copier' stage BEFORE build
COPY --from=copier /source/shared /shared

# Set build environment variables
ENV CGO_ENABLED=0 GOOS=linux
# Build the application
# Build targeting the main package within the copied src directory
RUN go build -mod=mod -o /latency-proxy ./src

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