#!/bin/bash
# deploy/update.sh - Manual deployment script for the VPS

set -e

echo "🚀 Starting manual deployment..."

# Check for root privileges
if [ "$(id -u)" -ne 0 ]; then
  echo "❌ This script must be run as root"
  exit 1
fi

# Change to the project directory
cd /opt/latency-space || { echo "❌ Failed to change directory"; exit 1; }

# Fix DNS if needed
echo "🔍 Checking DNS..."
if ! ping -c 1 github.com &> /dev/null; then
  echo "⚠️ DNS issues detected, fixing..."
  
  # Run the DNS fix script if it exists
  if [ -f /usr/local/bin/fix-dns ]; then
    /usr/local/bin/fix-dns
  else
    # Set DNS servers directly
    if [ -L /etc/resolv.conf ]; then
      # For systemd-resolved systems
      cat > /etc/systemd/resolved.conf << 'EOF'
[Resolve]
DNS=8.8.8.8 8.8.4.4 1.1.1.1
FallbackDNS=9.9.9.9 149.112.112.112
DNSStubListener=yes
Cache=yes
EOF
      systemctl restart systemd-resolved
    else
      # Direct modification
      echo "nameserver 8.8.8.8" > /etc/resolv.conf
      echo "nameserver 8.8.4.4" >> /etc/resolv.conf
      echo "nameserver 1.1.1.1" >> /etc/resolv.conf
    fi
  fi
  
  # Check again
  if ! ping -c 1 github.com &> /dev/null; then
    echo "❌ DNS still not working. Please check your DNS configuration."
    exit 1
  fi
fi

# Pull latest code
echo "📥 Pulling latest code from GitHub..."
git fetch origin
git reset --hard origin/main

# Clean up any stuck containers with very forceful approach
echo "🧹 Cleaning up any problematic containers..."
# Stop all containers 
docker ps -aq | xargs -r docker stop || true

# Special handling for problematic node-exporter
echo "Forcefully removing node-exporter if present..."
container_id=$(docker ps -a | grep "node-exporter" | awk '{print $1}')
if [ -n "$container_id" ]; then
  # Try normal remove
  docker rm -f $container_id || true
  
  # If still exists, use extreme measures
  if docker ps -a | grep -q $container_id; then
    echo "Forceful removal required. Restarting Docker service..."
    systemctl restart docker || true
    sleep 5
  fi
fi

# Remove any remaining containers
docker ps -a | grep "latency-space" | awk '{print $1}' | xargs -r docker rm -f || true

# Stop the current containers
echo "🛑 Stopping current containers..."
docker compose down || echo "⚠️ Warning: docker compose down failed, continuing..."

# Fix permissions before building
echo "🔧 Fixing permissions..."
if [ -f deploy/fix-permissions.sh ]; then
  bash deploy/fix-permissions.sh
else
  # Quick permissions fix if the script doesn't exist
  mkdir -p monitoring/prometheus/rules config certs
  chmod -R 755 monitoring config certs
fi

# Rebuild the containers
echo "🔨 Rebuilding containers..."
docker compose build --no-cache || echo "⚠️ Warning: build failed, continuing with existing images..."

# Start the containers - use minimal config first, then try other versions if available
echo "🚀 Starting containers..."

# Check for read-only filesystem
if touch /opt/latency-space/test_file 2>/dev/null; then
  rm /opt/latency-space/test_file
  echo "✅ Filesystem is writable"
  WRITABLE=true
else
  echo "⚠️ WARNING: Filesystem appears to be read-only!"
  WRITABLE=false
fi

if [ "$WRITABLE" = "false" ] || [ -f docker-compose.minimal.yml ]; then
  echo "🔄 Using minimal configuration..."
  # First try the pre-built minimal config
  if [ -f docker-compose.minimal.yml ]; then
    docker compose -f docker-compose.minimal.yml up -d --force-recreate
  else
    # Create a truly minimal config that doesn't use any volumes
    echo "Creating ultra-minimal configuration..."
    cat > docker-compose.ultra-minimal.yml << 'EOF'
services:
  proxy:
    build: 
      context: ./proxy
      dockerfile: Dockerfile
    ports:
      - "8080:80"
      - "1080:1080"
    restart: unless-stopped
EOF
    docker compose -f docker-compose.ultra-minimal.yml up -d
  fi
else
  # Try regular config first
  if ! docker compose up -d; then
    echo "⚠️ Error starting with regular docker-compose.yml, trying simplified version..."
    if [ -f docker-compose.simple.yml ]; then
      docker compose -f docker-compose.simple.yml up -d
    else
      echo "Using minimal configuration..."
      docker compose -f docker-compose.minimal.yml up -d
    fi
  fi
fi

# Reload nginx to apply configuration changes
echo "🔄 Reloading Nginx..."
systemctl reload nginx || echo "⚠️ Warning: Failed to reload nginx"

# Check if everything is running
echo "🔍 Checking if containers are running..."
sleep 10
if docker compose ps | grep -q "Up"; then
  echo "✅ Containers are running properly!"
else
  echo "❌ Containers failed to start. Checking logs..."
  docker compose logs
  exit 1
fi

# Check if SOCKS proxy is working
echo "🧦 Testing SOCKS proxy..."
if nc -z localhost 1080; then
  echo "✅ SOCKS proxy is running!"
else
  echo "❌ SOCKS proxy is not running!"
  exit 1
fi

# Check if HTTP proxy is working
echo "🌐 Testing HTTP proxy..."
if curl -s --max-time 5 http://localhost:8080 &> /dev/null; then
  echo "✅ HTTP proxy is running on port 8080!"
else
  echo "❌ HTTP proxy is not running on port 8080!"
  exit 1
fi

# Check if proxy is accessible through Nginx
echo "🌐 Testing Nginx proxy..."
if curl -s --max-time 5 -H "Host: mars.latency.space" http://localhost:80 &> /dev/null; then
  echo "✅ HTTP proxy is accessible through Nginx!"
else
  echo "⚠️ Warning: HTTP proxy may not be accessible through Nginx"
fi

echo "✅ Deployment completed successfully!"
echo "🔍 If you encounter any issues, check the following:"
echo "  - Nginx configuration: /etc/nginx/sites-available/latency.space"
echo "  - Container logs: docker compose logs"
echo "  - Nginx logs: tail -f /var/log/nginx/error.log"