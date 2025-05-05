# proxy/Dockerfile

# Use a specific golang Alpine version
FROM golang:1.21.6-alpine3.19 AS builder

# Install build dependencies (minimal)
RUN apk add --no-cache gcc musl-dev git

# Set working directory
WORKDIR /app

# Copy source code files
COPY src/*.go /app/
COPY src/templates /app/templates/

# Create a file with our constants to eliminate dependency on celestial
RUN echo 'package main' > constants.go && \
    echo '' >> constants.go && \
    echo '// Constants copied from celestial package' >> constants.go && \
    echo 'const (' >> constants.go && \
    echo '    SPEED_OF_LIGHT   = 299792.458  // km/s' >> constants.go && \
    echo '    AU               = 149597870.7 // Astronomical unit in kilometers' >> constants.go && \
    echo '    EARTH_RADIUS     = 6378.137    // Earth radius in kilometers' >> constants.go && \
    echo '    SUN_RADIUS       = 695700.0    // Sun radius in kilometers' >> constants.go && \
    echo '    SECONDS_PER_DAY  = 86400.0     // Seconds in a day' >> constants.go && \
    echo '    DAYS_PER_CENTURY = 36525.0     // Days in a Julian century (365.25 * 100)' >> constants.go && \
    echo '    J2000_EPOCH      = 2451545.0   // J2000 epoch in Julian days (January 1, 2000, 12:00 TT)' >> constants.go && \
    echo ')' >> constants.go

# Add types for CelestialObject and Vector3
RUN echo 'package main' > types.go && \
    echo '' >> types.go && \
    echo '// CelestialObject defines a celestial object structure' >> types.go && \
    echo 'type CelestialObject struct {' >> types.go && \
    echo '    Name string' >> types.go && \
    echo '    Type string' >> types.go && \
    echo '}' >> types.go && \
    echo '' >> types.go && \
    echo '// Vector3 represents a 3D vector' >> types.go && \
    echo 'type Vector3 struct {' >> types.go && \
    echo '    X, Y, Z float64' >> types.go && \
    echo '}' >> types.go && \
    echo '' >> types.go

# Modify the source files to remove external dependencies
RUN rm -f models.go && \
    echo 'package main' > models.go && \
    echo '' >> models.go

# Create a standalone go.mod file using Go's own tools
RUN go mod init main && \
    go mod edit -go=1.21 && \
    go mod edit -require=github.com/gorilla/websocket@v1.5.0 && \
    go mod edit -require=github.com/prometheus/client_golang@v1.15.0 && \
    go mod edit -require=golang.org/x/crypto@v0.14.0 && \
    go mod edit -require=golang.org/x/time@v0.3.0

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
    && mkdir -p /etc/latency-space

# Copy the binary and templates
COPY --from=builder /app/latency-proxy /usr/local/bin/latency-proxy
COPY --from=builder /app/templates/ /app/templates/
RUN chmod +x /usr/local/bin/latency-proxy

# Expose ports
EXPOSE 80 443 1080 9090

CMD ["/usr/local/bin/latency-proxy"]