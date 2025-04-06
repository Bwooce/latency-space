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
  
  # Set DNS servers
  echo "nameserver 8.8.8.8" > /etc/resolv.conf
  echo "nameserver 8.8.4.4" >> /etc/resolv.conf
  echo "nameserver 1.1.1.1" >> /etc/resolv.conf
  
  # Check again
  if ! ping -c 1 github.com &> /dev/null; then
    echo "❌ DNS still not working. Please run fix-dns manually."
    exit 1
  fi
fi

# Pull latest code
echo "📥 Pulling latest code from GitHub..."
git fetch origin
git reset --hard origin/main

# Stop the current containers
echo "🛑 Stopping current containers..."
docker-compose down || echo "⚠️ Warning: docker-compose down failed, continuing..."

# Rebuild the containers
echo "🔨 Rebuilding containers..."
docker-compose build --no-cache || echo "⚠️ Warning: build failed, continuing with existing images..."

# Start the containers
echo "🚀 Starting containers..."
docker-compose up -d

# Check if everything is running
echo "🔍 Checking if containers are running..."
sleep 10
if docker-compose ps | grep -q "Up"; then
  echo "✅ Containers are running properly!"
else
  echo "❌ Containers failed to start. Checking logs..."
  docker-compose logs
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
if curl -s --max-time 5 http://localhost:80 &> /dev/null; then
  echo "✅ HTTP proxy is running!"
else
  echo "❌ HTTP proxy is not running!"
  exit 1
fi

echo "✅ Deployment completed successfully!"