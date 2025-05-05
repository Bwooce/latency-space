# proxy/Dockerfile

# Use a specific golang Alpine version
FROM golang:1.21.6-alpine3.19 AS builder

# Install build dependencies (minimal)
RUN apk add --no-cache gcc musl-dev git openssh-client

# Set working directory
WORKDIR /app

# Configure Git for SSH access
RUN mkdir -p /root/.ssh && ssh-keyscan github.com >> /root/.ssh/known_hosts
RUN git config --global url."ssh://git@github.com/".insteadOf "https://github.com/"
# Mount SSH key here using --mount=type=ssh when building

# Copy source code files
COPY src/ /app/
COPY src/templates /app/templates/

# Build with no vendor directory and fixed dependencies
ENV GOPRIVATE=github.com/latency-space/*
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
    && mkdir -p /etc/latency-space

# Copy the binary and templates
COPY --from=builder /app/latency-proxy /usr/local/bin/latency-proxy
COPY --from=builder /app/templates/ /app/templates/
RUN chmod +x /usr/local/bin/latency-proxy

# Expose ports
EXPOSE 80 443 1080 9090

CMD ["/usr/local/bin/latency-proxy"]