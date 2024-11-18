FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev linux-headers

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /latency-proxy

FROM alpine:latest

# Install required tools
RUN apk add --no-cache \
    iproute2 \
    iptables \
    openssl \
    ca-certificates

COPY --from=builder /latency-proxy /usr/local/bin/latency-proxy
RUN chmod +x /usr/local/bin/latency-proxy

# Create necessary directories
RUN mkdir -p /etc/latency-space /app/certs
VOLUME /app/certs

CMD ["/usr/local/bin/latency-proxy"]
