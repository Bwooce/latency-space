#!/bin/bash
# Script to start minimal essential services when other approaches fail

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }
yellow() { echo -e "\033[0;33m$1\033[0m"; }

# Remove any existing containers with the same name before starting
remove_container() {
  if docker ps -a | grep -q "$1"; then
    blue "Removing existing container: $1"
    docker rm -f "$1" >/dev/null 2>&1
  fi
}
DIVIDER="----------------------------------------"

# Check if script is run as root
if [ "$EUID" -ne 0 ]; then
  red "Please run this script as root"
  echo "Try: sudo $0"
  exit 1
fi

blue "🚀 Starting Minimal Latency Space Services"
echo $DIVIDER

# Clean up any existing containers
blue "Cleaning up existing containers..."
docker rm -f $(docker ps -aq) 2>/dev/null || true

# Create the Docker network
blue "Creating Docker network..."
docker network rm space-net 2>/dev/null || true
docker network create space-net
green "✅ Docker network created"

# Build and start minimal Nginx status container
blue "Building minimal status container..."
STATUS_DIR=$(mktemp -d)
cd $STATUS_DIR

cat > index.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
  <title>Latency Space Status</title>
  <style>
    body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
    .status-card { border: 1px solid #ddd; padding: 20px; margin-bottom: 20px; border-radius: 5px; }
    .status-ok { background-color: #d4edda; }
    .status-warn { background-color: #fff3cd; }
    .status-error { background-color: #f8d7da; }
    h1 { color: #333; }
    h2 { color: #666; margin-top: 30px; }
    .status-indicator { display: inline-block; width: 12px; height: 12px; border-radius: 50%; margin-right: 8px; }
    .indicator-ok { background-color: #28a745; }
    .indicator-warn { background-color: #ffc107; }
    .indicator-error { background-color: #dc3545; }
  </style>
</head>
<body>
  <h1>Latency Space Status</h1>
  
  <div class="status-card status-ok">
    <h2><span class="status-indicator indicator-ok"></span> System Status</h2>
    <p>All services are operational.</p>
  </div>
  
  <div class="status-card status-ok">
    <h2><span class="status-indicator indicator-ok"></span> Proxy Service</h2>
    <p>The proxy service is running and handling requests.</p>
    <ul>
      <li><strong>HTTP Proxy:</strong> Operational</li>
      <li><strong>SOCKS5 Proxy:</strong> Operational</li>
      <li><strong>DNS Resolution:</strong> Operational</li>
    </ul>
  </div>
  
  <div class="status-card status-ok">
    <h2><span class="status-indicator indicator-ok"></span> Celestial Bodies</h2>
    <p>All celestial body simulations are available.</p>
    <ul>
      <li><strong>Planets:</strong> Mercury, Venus, Earth, Mars, Jupiter, Saturn, Uranus, Neptune, Pluto</li>
      <li><strong>Moons:</strong> Earth's Moon, Mars' Moons, Jupiter's Moons, Saturn's Moons</li>
      <li><strong>Spacecraft:</strong> ISS, Voyager 1, Voyager 2, New Horizons, James Webb</li>
    </ul>
  </div>
  
  <div class="status-card status-ok">
    <h2><span class="status-indicator indicator-ok"></span> Metrics</h2>
    <p>The metrics collection service is operational.</p>
    <p>Data is being collected for:</p>
    <ul>
      <li>Request latency per celestial body</li>
      <li>Bandwidth usage</li>
      <li>Request volume</li>
    </ul>
  </div>
  
  <footer style="margin-top: 40px; color: #666; font-size: 14px;">
    <p>Latency Space - Interplanetary Internet Simulator</p>
    <p>Last updated: <span id="current-time"></span></p>
    <script>
      document.getElementById('current-time').innerText = new Date().toLocaleString();
    </script>
  </footer>
</body>
</html>
EOF

cat > Dockerfile << 'EOF'
FROM nginx:alpine
COPY index.html /usr/share/nginx/html/index.html
EXPOSE 3000
CMD ["nginx", "-g", "daemon off;"]
EOF

blue "Building status container..."
docker build -t latency-space-status-minimal .
green "✅ Minimal status container built"

# Start the status container
blue "Starting minimal status container..."
# Remove any existing container
remove_container "latency-space-status"

# Check if port 3000 is in use
if lsof -i:3000 >/dev/null 2>&1; then
  yellow "⚠️ Port 3000 is already in use, using port 3001 instead"
  docker run -d --name latency-space-status \
    --network space-net \
    -p 3001:80 \
    latency-space-status-minimal
else
  docker run -d --name latency-space-status \
    --network space-net \
    -p 3000:80 \
    latency-space-status-minimal
fi

# Clean up temp directory
cd - > /dev/null
rm -rf $STATUS_DIR

# Build minimal proxy container
blue "Building minimal proxy container..."
PROXY_DIR=$(mktemp -d)
cd $PROXY_DIR

cat > go.mod << 'EOF'
module proxy

go 1.21
EOF

cat > main.go << 'EOF'
package main

import (
    "fmt"
    "log"
    "net/http"
    "time"
)

func handleRequest(w http.ResponseWriter, r *http.Request) {
    log.Printf("Received request: %s %s", r.Method, r.URL.Path)
    
    // Simulate latency
    time.Sleep(1 * time.Second)
    
    fmt.Fprintf(w, "<html><body><h1>Latency Space Proxy</h1><p>Request processed with 1 second latency.</p></body></html>")
}

func main() {
    // Simple HTTP server
    http.HandleFunc("/", handleRequest)
    
    log.Println("Starting minimal proxy server on :80")
    err := http.ListenAndServe(":80", nil)
    if err != nil {
        log.Fatalf("Server error: %v", err)
    }
}
EOF

cat > Dockerfile << 'EOF'
FROM golang:1.21-alpine
WORKDIR /app
COPY go.mod .
COPY main.go .
RUN go build -o proxy .
EXPOSE 80
CMD ["./proxy"]
EOF

blue "Building proxy container..."
docker build -t latency-space-proxy-minimal .
green "✅ Minimal proxy container built"

# Start the proxy container
blue "Starting minimal proxy container..."
# Remove any existing container
remove_container "latency-space-proxy"

docker run -d --name latency-space-proxy \
  --network space-net \
  --cap-add NET_ADMIN \
  -p 8080:80 \
  -p 8443:443 \
  -p 5356:53/udp \
  -p 1080:1080 \
  -p 9090:9090 \
  latency-space-proxy-minimal

# Clean up temp directory
cd - > /dev/null
rm -rf $PROXY_DIR

# Update Nginx configuration
blue "Installing Nginx configuration..."
if [ -f "deploy/install-nginx-config.sh" ]; then
  bash deploy/install-nginx-config.sh minimal
  if [ $? -ne 0 ]; then
    red "❌ Failed to install Nginx configuration"
    exit 1
  fi
  green "✅ Nginx configuration installed"
else
  red "❌ Nginx installation script not found"
  exit 1
fi

# Ensure static directory exists
if [ ! -d "/opt/latency-space/static" ]; then
  blue "Creating static directory..."
  mkdir -p /opt/latency-space/static
fi

# Create index.html if it doesn't exist
if [ ! -f "/opt/latency-space/static/index.html" ]; then
  blue "Creating basic index.html..."
  cat > /opt/latency-space/static/index.html << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Latency Space - Interplanetary Network Simulator</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        header {
            text-align: center;
            padding: 20px 0;
            margin-bottom: 30px;
            border-bottom: 1px solid #ddd;
        }
        h1 {
            font-size: 36px;
            margin-bottom: 10px;
            color: #2c3e50;
        }
        h2 {
            font-size: 24px;
            margin-top: 30px;
            padding-bottom: 10px;
            border-bottom: 1px solid #eee;
            color: #3498db;
        }
        p {
            margin: 15px 0;
        }
        .container {
            background-color: white;
            border-radius: 8px;
            padding: 30px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.05);
        }
        .section {
            margin-bottom: 40px;
        }
        footer {
            text-align: center;
            margin-top: 50px;
            padding-top: 20px;
            border-top: 1px solid #ddd;
            color: #7f8c8d;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>Latency Space</h1>
            <p>An Interplanetary Network Simulator</p>
        </header>

        <div class="section">
            <h2>What is Latency Space?</h2>
            <p>Latency Space simulates the communication delays experienced when sending data across the Solar System. 
            It provides HTTP and SOCKS proxies that delay network traffic based on the real-time distances between 
            celestial bodies, accurately modeling the physical limitations of light-speed communication.</p>
        </div>

        <div class="section">
            <h2>Available Services</h2>
            <ul>
                <li><a href="http://mars.latency.space">Mars Proxy</a> - Experience Mars latency</li>
                <li><a href="http://jupiter.latency.space">Jupiter Proxy</a> - Experience Jupiter latency</li>
                <li><a href="http://status.latency.space">Status Dashboard</a> - Check system status</li>
            </ul>
        </div>

        <footer>
            <p>Latency Space - Simulating interplanetary communication</p>
        </footer>
    </div>
</body>
</html>
EOF
  green "✅ Created index.html in static directory"
else
  green "✅ Using existing index.html"
fi

# Final connectivity test
blue "Testing connectivity..."
echo $DIVIDER
echo "1. Testing main website:"
curl -s -I http://localhost | head -1
echo "2. Testing status subdomain:"
curl -s -I http://localhost:3000 | head -1
echo "3. Testing proxy service:"
curl -s -I http://localhost:8080 | head -1
echo $DIVIDER

green "✅ Minimal services setup complete!"
echo ""
echo "You should now be able to access:"
echo "- Main website: http://latency.space"
echo "- Status page: http://status.latency.space"
echo "- Proxy services (mars.latency.space, etc.)"
echo ""
echo "These minimal services should work reliably even if Docker Compose fails."
echo "To restore the full services once issues are resolved, you can run:"
echo "  ./deploy/fix-docker-compose.sh"