#!/bin/bash
# Script to fix status container issues
# This script focuses on fixing issues specific to the status container

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }
yellow() { echo -e "\033[0;33m$1\033[0m"; }
DIVIDER="----------------------------------------"

# Check if script is run as root
if [ "$EUID" -ne 0 ]; then
  red "Please run this script as root"
  echo "Try: sudo $0"
  exit 1
fi

blue "🔧 Status Container Fix Script"
echo $DIVIDER

# Ensure we're in the right directory
cd /opt/latency-space || { red "❌ Could not change to /opt/latency-space directory"; exit 1; }
blue "Working in $(pwd)"

# Check Docker service status
blue "Checking Docker service..."
if ! systemctl is-active --quiet docker; then
  yellow "⚠️ Docker service is not running. Starting it..."
  systemctl start docker
  systemctl enable docker
  if ! systemctl is-active --quiet docker; then
    red "❌ Failed to start Docker service"
    exit 1
  fi
  green "✅ Docker service started and enabled"
fi

# Check status directory and Dockerfile
blue "Checking status directory and Dockerfile..."
if [ ! -d "status" ]; then
  red "❌ Status directory not found"
  exit 1
fi

if [ ! -f "status/Dockerfile" ]; then
  red "❌ Status Dockerfile not found"
  exit 1
fi

green "✅ Status directory and Dockerfile found"

# Fix the port issue in docker-compose.yml
blue "Checking port configuration in docker-compose.yml..."
if grep -q '"3000:80"' docker-compose.yml; then
  blue "Port mapping is already using port 80 inside container, which matches Nginx config"
else
  yellow "⚠️ Port mapping may be incorrect in docker-compose.yml"
  
  # Fix the port mapping - status container uses Nginx on port 80 internally
  blue "Updating port mapping in docker-compose.yml..."
  sed -i 's/"3000:3000"/"3000:80"/g' docker-compose.yml
  
  if grep -q '"3000:80"' docker-compose.yml; then
    green "✅ Port mapping updated to 3000:80"
  else
    red "❌ Failed to update port mapping"
  fi
fi

# Stop and remove any existing status container
blue "Stopping and removing existing status container..."
docker stop $(docker ps -a -q --filter name=status) 2>/dev/null || true
docker rm $(docker ps -a -q --filter name=status) 2>/dev/null || true
green "✅ Removed any existing status container"

# Rebuild the status container
blue "Rebuilding status container from scratch..."
docker compose build --no-cache status
if [ $? -ne 0 ]; then
  yellow "⚠️ docker compose build failed, trying alternative approach..."
  docker compose build --no-cache status
  if [ $? -ne 0 ]; then
    red "❌ Failed to build status container"
    
    # Try direct Docker build
    blue "Trying direct Docker build..."
    docker build -t latency-space-status ./status
    if [ $? -ne 0 ]; then
      red "❌ All build approaches failed"
      # Check the Dockerfile for issues
      blue "Checking Dockerfile for issues..."
      cat status/Dockerfile
      exit 1
    else
      green "✅ Direct Docker build successful"
    fi
  else
    green "✅ Successfully rebuilt with docker compose v2"
  fi
else
  green "✅ Successfully rebuilt with docker-compose v1"
fi

# Start the status container
blue "Starting status container..."
docker compose up -d status
if [ $? -ne 0 ]; then
  yellow "⚠️ docker compose up failed, trying alternative approach..."
  docker compose up -d status
  if [ $? -ne 0 ]; then
    red "❌ Failed to start status container"
    
    # Try running container directly if we have the image
    if docker images | grep -q latency-space-status; then
      blue "Trying to start container directly..."
      # Create network if it doesn't exist
      if ! docker network ls | grep -q space-net; then
        docker network create space-net
      fi
      
      docker run -d --name latency-space-status \
        -p 3000:80 \
        --network space-net \
        --restart unless-stopped \
        latency-space-status
      
      if [ $? -ne 0 ]; then
        red "❌ All attempts to start the container failed"
        exit 1
      else
        green "✅ Container started directly with Docker run"
      fi
    else
      red "❌ No image available for direct run"
      exit 1
    fi
  else
    green "✅ Container started with docker compose v2"
  fi
else
  green "✅ Container started with docker-compose v1"
fi

# Check if container is running
blue "Checking if status container is running..."
if docker ps | grep -q "status"; then
  green "✅ Status container is running"
  
  # Get the container's logs
  blue "Recent container logs:"
  docker logs $(docker ps -q -f name=status) --tail 10
else
  red "❌ Status container failed to start or stay running"
  blue "Checking container logs..."
  docker logs $(docker ps -a -q -f name=status) --tail 20
  exit 1
fi

# Check nginx configuration
blue "Checking Nginx configuration on host..."
if [ -f "/etc/nginx/sites-enabled/latency.space" ]; then
  blue "Checking if Nginx is correctly configured for status.latency.space..."
  if grep -q "proxy_pass.*status:80" /etc/nginx/sites-enabled/latency.space; then
    yellow "⚠️ Nginx is configured to use status:80 but container may be exposing port 3000"
    
    # Update nginx config
    if [ -f "config/nginx/latency.space.conf" ]; then
      blue "Updating Nginx configuration to use status:80..."
      sudo sed -i 's/set $upstream_status http:\/\/status:3000/set $upstream_status http:\/\/status:80/g' config/nginx/latency.space.conf
      
      # Install updated configuration
      if [ -f "deploy/install-nginx-config.sh" ]; then
        blue "Installing updated Nginx configuration..."
        sudo bash deploy/install-nginx-config.sh
        green "✅ Nginx configuration updated and installed"
      fi
    fi
  else
    green "✅ Nginx configuration appears to be correct"
  fi
else
  yellow "⚠️ Nginx configuration not found at /etc/nginx/sites-enabled/latency.space"
  
  # Install the Nginx configuration
  if [ -f "deploy/install-nginx-config.sh" ]; then
    blue "Installing Nginx configuration..."
    sudo bash deploy/install-nginx-config.sh
    green "✅ Nginx configuration installed"
  else
    red "❌ Nginx installation script not found"
  fi
fi

# Test connectivity
blue "Testing status dashboard connectivity..."
echo "1. Docker container status: $(docker ps | grep status || echo 'Not running')"
echo "2. Direct access on port 3000: $(curl -s -I -m 2 http://localhost:3000 | head -1 || echo 'Failed')"
echo "3. Through Nginx via status.latency.space: $(curl -s -I -m 2 -H "Host: status.latency.space" http://localhost | head -1 || echo 'Failed')"

# Final status
echo $DIVIDER
green "✅ Status container fix script completed!"
echo ""
echo "If you're still experiencing issues:"
echo "1. Check Docker network connectivity: docker network inspect space-net"
echo "2. Check container logs: docker logs \$(docker ps -q -f name=status)"
echo "3. Check Nginx logs: tail -f /var/log/nginx/error.log"
echo "4. Try accessing the status dashboard at: http://status.latency.space"
echo "5. Try direct access at: http://YOUR_SERVER_IP:3000"