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

# Restart the containers
echo "🔄 Restarting all containers..."
cd /opt/latency-space
docker compose down
echo "🏗️ Building proxy image..."
docker compose build --no-cache proxy
docker compose up -d
if [ $? -eq 0 ]; then
  echo "✅ All containers restarted successfully"
else
  echo "❌ Failed to restart containers"
  exit 1
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