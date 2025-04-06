#!/bin/bash
# Emergency restart script for proxy only

set -e

echo "🚨 EMERGENCY RESTART 🚨"
echo "This script will attempt to start just the proxy service with minimal configuration"

# Check if we're in the right directory
if [ ! -d "./proxy" ]; then
  echo "❌ This script must be run from the project root directory"
  exit 1
fi

# Stop all existing containers
echo "🛑 Stopping all containers..."
docker ps -a | grep "latency-space" | awk '{print $1}' | xargs -r docker stop
docker ps -a | grep "latency-space" | awk '{print $1}' | xargs -r docker rm

# Try to restart Docker if necessary
echo "🔄 Restarting Docker..."
systemctl restart docker
sleep 5

# Create an ultra-minimal docker-compose file
echo "📝 Creating ultra-minimal configuration..."
cat > docker-compose.emergency.yml << 'EOF'
services:
  proxy:
    image: proxy
    ports:
      - "8080:80"
      - "1080:1080"
    restart: unless-stopped
EOF

# Check if the image exists
if docker images | grep -q "proxy"; then
  echo "✅ Found existing proxy image"
else
  echo "❌ No proxy image found. Building from scratch..."
  cd proxy
  docker build -t proxy .
  cd ..
fi

# Start just the proxy
echo "🚀 Starting proxy..."
docker compose -f docker-compose.emergency.yml up -d

# Verify it's running
echo "🔍 Checking if proxy is running..."
if docker ps | grep -q "proxy"; then
  echo "✅ Proxy is running!"
else
  echo "❌ Failed to start proxy"
  docker logs $(docker ps -a | grep "proxy" | awk '{print $1}')
  exit 1
fi

echo "🌐 Verify it works by accessing:"
echo "  HTTP proxy: http://localhost:8080"
echo "  SOCKS proxy: Configure for localhost:1080"
echo ""
echo "This is a minimal emergency setup. Run the standard update.sh when possible."