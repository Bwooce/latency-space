# proxy/Dockerfile

# Use specific golang Alpine version for builder
FROM golang:1.21.6-alpine3.19 AS builder

WORKDIR /app/src

# Combine installation and build steps to reduce layers
RUN apk add --no-cache \
    gcc=~13.2 \
    musl-dev=~1.2 \
    linux-headers=~6.5 \
    git

# First copy the shared package to the correct location
COPY src/shared /go/src/github.com/latency-space/shared

# Copy source files
COPY src/ .

# Copy updated go.mod that doesn't use replace directive
RUN cp go.mod go.mod.orig && \
    go mod edit -dropreplace=github.com/latency-space/shared && \
    go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /latency-proxy

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