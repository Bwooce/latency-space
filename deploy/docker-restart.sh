#!/bin/bash
# Restart script for latency.space Docker containers
# This script properly restarts Docker service and containers

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

blue "🔄 Restarting latency.space services"
echo $DIVIDER

# Check Docker service status
blue "Checking Docker service..."
if systemctl is-active --quiet docker; then
  green "✅ Docker service is running"
else
  yellow "⚠️ Docker service is not running. Starting it now..."
  systemctl start docker
  if systemctl is-active --quiet docker; then
    green "✅ Docker service started successfully"
  else
    red "❌ Failed to start Docker service. Please check systemctl status docker"
    exit 1
  fi
fi

# Set Docker to start on boot
blue "Enabling Docker to start on boot..."
systemctl enable docker
green "✅ Docker service enabled"

# Change to the latency-space directory
cd /opt/latency-space || { red "❌ Could not change to /opt/latency-space directory"; exit 1; }

# Stop existing containers
blue "Stopping all latency.space containers..."
docker compose down
green "✅ All containers stopped"

# Optional: Remove any dangling containers
blue "Cleaning up any dangling resources..."
docker system prune -f
green "✅ Cleanup complete"

# Start containers
blue "Starting all containers..."
docker compose up -d
if [ $? -eq 0 ]; then
  green "✅ All containers started successfully"
else
  red "❌ Error starting containers. Please check logs"
  exit 1
fi

# Verify containers are running
blue "Verifying container status..."
echo $DIVIDER
docker compose ps
echo $DIVIDER

# Check if status container is running
if ! docker ps | grep -q "status"; then
  red "❌ Status container is not running"
  
  # Try to start it specifically
  blue "Attempting to start status container specifically..."
  docker compose up -d status
  
  if docker ps | grep -q "status"; then
    green "✅ Status container started successfully"
  else
    red "❌ Failed to start status container"
    
    # Check logs for status container
    blue "Checking logs for status container..."
    docker compose logs status
    
    yellow "⚠️ The status container failed to start. This might be due to:"
    echo "   1. Misconfiguration in compose file"
    echo "   2. Network issues between containers"
    echo "   3. Resource constraints"
    
    # Offer to rebuild the status container
    read -p "Would you like to try rebuilding the status container? (y/n): " rebuild
    if [[ "$rebuild" == "y" ]]; then
      blue "Rebuilding status container..."
      docker compose build --no-cache status
      docker compose up -d status
      
      if docker ps | grep -q "status"; then
        green "✅ Status container rebuilt and started successfully"
      else
        red "❌ Failed to start status container after rebuilding"
      fi
    fi
  fi
fi

# Fix Nginx configuration
blue "Would you like to fix the Nginx configuration? (y/n): "
read fix_nginx
if [[ "$fix_nginx" == "y" ]]; then
  if [ -f "/opt/latency-space/deploy/install-nginx-config.sh" ]; then
    blue "Running Nginx configuration install script..."
    sudo bash /opt/latency-space/deploy/install-nginx-config.sh
  else
    red "❌ Nginx install script not found at /opt/latency-space/deploy/install-nginx-config.sh"
    yellow "Try running: git pull to get the latest scripts"
  fi
fi

# Fix DNS records
blue "Would you like to fix DNS records using Cloudflare API? (y/n): "
read fix_dns
if [[ "$fix_dns" == "y" ]]; then
  if [ -f "/opt/latency-space/deploy/fix-all-dns.sh" ]; then
    blue "Please enter your Cloudflare API token: "
    read -s CF_API_TOKEN
    export CF_API_TOKEN
    
    blue "Running DNS fix script..."
    bash /opt/latency-space/deploy/fix-all-dns.sh
  else
    red "❌ DNS fix script not found at /opt/latency-space/deploy/fix-all-dns.sh"
  fi
fi

# Final status check
echo $DIVIDER
blue "📊 Final status check:"
docker ps
green "✅ Restart process complete!"
echo ""
echo "If you're still experiencing issues, run the server health check:"
echo "  /opt/latency-space/deploy/server-health-check.sh"
echo ""
echo "You should now be able to access:"
echo "- http://status.latency.space"
echo "- http://mars.latency.space"
echo "- Other planetary subdomains"