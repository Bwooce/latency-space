# proxy/Dockerfile

# Use specific golang Alpine version for builder
FROM golang:1.21.6-alpine3.19 AS builder

WORKDIR /app/src

# Combine installation and build steps to reduce layers
RUN apk add --no-cache \
    gcc==13.2.1_git20231014-r0 \
    musl-dev==1.2.4_git20230717-r4 \
    linux-headers==6.5-r0

# Copy dependency files first for better caching
COPY src/go.mod src/go.sum ./
RUN go mod download

# Copy source and build
COPY src/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /latency-proxy

# Use specific Alpine version for final image
FROM alpine:3.19.1

# Install required tools in a single layer
RUN apk add --no-cache \
    iproute2==6.6.0-r0 \
    iptables==1.8.10-r3 \
    openssl==3.1.7-r1 \
    && mkdir -p /etc/latency-space

# Copy and prepare binary
COPY --from=builder /latency-proxy /usr/local/bin/latency-proxy
RUN chmod +x /usr/local/bin/latency-proxy

CMD ["/usr/local/bin/latency-proxy"]

