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

# Copy the source files, including the vendor directory if it exists
COPY src/ .

# Create a local vendor directory with the shared package
RUN mkdir -p vendor/github.com/latency-space/shared
COPY src/shared/ vendor/github.com/latency-space/shared/

# Configure Go to use vendoring
ENV GOFLAGS="-mod=vendor"
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