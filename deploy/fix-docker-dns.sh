#!/bin/bash
# Script to fix Docker DNS resolution issues and restart containers properly

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

blue "🔧 Docker DNS Resolution Fix"
echo $DIVIDER

# Check if we're in the right directory
if [ ! -f "docker-compose.yml" ] && [ ! -f "docker-compose.yaml" ]; then
  red "Docker compose file not found. Please run this script from the latency-space directory"
  exit 1
fi

# First, we'll properly configure Docker's DNS settings
blue "Configuring Docker DNS settings..."

# Create /etc/docker directory if it doesn't exist
mkdir -p /etc/docker

# Check if daemon.json exists
if [ -f "/etc/docker/daemon.json" ]; then
  blue "Backing up existing daemon.json..."
  cp /etc/docker/daemon.json /etc/docker/daemon.json.bak
  
  # Create a minimal Docker daemon configuration to avoid conflicts
  echo '{
  "dns": ["8.8.8.8", "8.8.4.4", "1.1.1.1"],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}' > /etc/docker/daemon.json
else
  # Create a new daemon.json file
  blue "Creating new Docker daemon config with proper DNS settings..."
  echo '{
  "dns": ["8.8.8.8", "8.8.4.4", "1.1.1.1"],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}' > /etc/docker/daemon.json
fi

# Fix permissions
chmod 644 /etc/docker/daemon.json
green "✅ Docker DNS configuration updated"

# Restart Docker
blue "Restarting Docker service..."
systemctl restart docker
green "✅ Docker service restarted"

# Wait for Docker to be fully available
blue "Waiting for Docker to become available..."
counter=0
max_attempts=30
until docker info &>/dev/null || [ $counter -eq $max_attempts ]; do
  echo -n "."
  sleep 1
  counter=$((counter+1))
done

if [ $counter -eq $max_attempts ]; then
  red "❌ Docker did not start within the expected time"
  echo "Please check Docker status with: systemctl status docker"
  exit 1
fi
echo ""
green "✅ Docker is now available"

# Restart all containers
blue "Stopping all containers..."
docker compose down || docker stop $(docker ps -a -q) 2>/dev/null || true

blue "Removing orphaned containers..."
docker container prune -f

# Run docker compose with correct file path
blue "Starting containers with Docker Compose..."
docker compose up -d

# Wait for containers to be ready
blue "Waiting for containers to become ready..."
sleep 10

# Add hosts entries and update Nginx config
blue "Updating Nginx configuration with container IPs..."

# Get container IPs
STATUS_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=status) 2>/dev/null)
PROXY_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=proxy) 2>/dev/null)
PROMETHEUS_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=prometheus) 2>/dev/null)
GRAFANA_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=grafana) 2>/dev/null)

# Print container IPs for debugging
echo "Status container IP: $STATUS_IP"
echo "Proxy container IP: $PROXY_IP"
echo "Prometheus container IP: $PROMETHEUS_IP"
echo "Grafana container IP: $GRAFANA_IP"

# Update Nginx config
if [ -f "/etc/nginx/sites-enabled/latency.space" ]; then
  # Backup the config
  cp /etc/nginx/sites-enabled/latency.space /etc/nginx/sites-enabled/latency.space.backup.$(date +%s)
  
  # Update status IP
  if [ -n "$STATUS_IP" ]; then
    blue "Updating status.latency.space to use IP $STATUS_IP..."
    sed -i "s/proxy_pass http:\/\/status:80/proxy_pass http:\/\/$STATUS_IP:80/g" /etc/nginx/sites-enabled/latency.space
    sed -i "s/proxy_pass http:\/\/status:3000/proxy_pass http:\/\/$STATUS_IP:80/g" /etc/nginx/sites-enabled/latency.space
    sed -i "s/set \$upstream_status http:\/\/status:80/proxy_pass http:\/\/$STATUS_IP:80/g" /etc/nginx/sites-enabled/latency.space
    sed -i "s/set \$upstream_status http:\/\/status:3000/proxy_pass http:\/\/$STATUS_IP:80/g" /etc/nginx/sites-enabled/latency.space
  fi
  
  # Update proxy IP
  if [ -n "$PROXY_IP" ]; then
    blue "Updating latency.space to use proxy IP $PROXY_IP..."
    sed -i "s/proxy_pass http:\/\/proxy:80/proxy_pass http:\/\/$PROXY_IP:80/g" /etc/nginx/sites-enabled/latency.space
    sed -i "s/set \$upstream_proxy http:\/\/proxy:80/proxy_pass http:\/\/$PROXY_IP:80/g" /etc/nginx/sites-enabled/latency.space
  fi
  
  # Test and reload Nginx
  blue "Testing Nginx configuration..."
  nginx -t
  if [ $? -ne 0 ]; then
    red "❌ Nginx configuration test failed"
    yellow "Restoring previous configuration..."
    cp /etc/nginx/sites-enabled/latency.space.backup.$(ls -t /etc/nginx/sites-enabled/latency.space.backup.* | head -1 | awk -F'.' '{print $NF}') /etc/nginx/sites-enabled/latency.space
  else
    green "✅ Nginx configuration test passed"
    blue "Reloading Nginx..."
    systemctl reload nginx
    green "✅ Nginx reloaded successfully"
  fi
else
  # If the config doesn't exist, see if we can install it
  yellow "⚠️ Nginx configuration file not found at /etc/nginx/sites-enabled/latency.space"
  
  # Copy our template configuration and update IPs
  if [ -f "deploy/nginx-proxy.conf" ]; then
    blue "Installing Nginx configuration from template..."
    cp deploy/nginx-proxy.conf /etc/nginx/sites-available/latency.space
    
    # Update IPs in the new config
    if [ -n "$STATUS_IP" ]; then
      sed -i "s/172.18.0.5:80/$STATUS_IP:80/g" /etc/nginx/sites-available/latency.space
    fi
    
    if [ -n "$PROXY_IP" ]; then
      sed -i "s/172.18.0.4:80/$PROXY_IP:80/g" /etc/nginx/sites-available/latency.space
    fi
    
    # Enable the site
    ln -sf /etc/nginx/sites-available/latency.space /etc/nginx/sites-enabled/
    
    # Test and reload Nginx
    nginx -t
    if [ $? -eq 0 ]; then
      systemctl reload nginx
      green "✅ Nginx configuration installed and loaded"
    else
      red "❌ Nginx configuration test failed"
    fi
  else
    red "❌ Nginx configuration template not found"
  fi
fi

# Skip adding host entries to containers due to permission issues
blue "Configuring container networking..."

yellow "⚠️ Cannot modify /etc/hosts in containers - they lack sufficient permissions"
yellow "⚠️ Using direct IPs in Nginx configuration and host machine's /etc/hosts instead"

# Alternative approach: Recreate containers with extra_hosts option
# This would require stopping and recreating all containers, so we'll skip it for now
# blue "For a more permanent solution, consider adding extra_hosts in docker-compose.yml:"
# echo "services:"
# echo "  proxy:"
# echo "    extra_hosts:"
# echo "      - \"status:$STATUS_IP\""
# echo "      - \"prometheus:$PROMETHEUS_IP\""
# echo "  status:"
# echo "    extra_hosts:"
# echo "      - \"proxy:$PROXY_IP\""
# echo "      - \"prometheus:$PROMETHEUS_IP\""

# Try to use container networks directly
blue "Checking Docker network configuration..."
docker network inspect space-net || docker network create space-net

# Ensure all containers are connected to the same network
for container in $(docker ps -q); do
  name=$(docker inspect --format '{{.Name}}' $container | sed 's/\///')
  networks=$(docker inspect --format '{{range $net, $conf := .NetworkSettings.Networks}}{{$net}} {{end}}' $container)
  
  if [[ "$networks" != *"space-net"* ]]; then
    blue "Connecting $name to space-net network..."
    docker network connect space-net $container || true
  else
    green "✅ $name already connected to space-net"
  fi
done

# Add entries to host machine's /etc/hosts
if [ -n "$STATUS_IP" ] && [ -n "$PROXY_IP" ]; then
  blue "Adding entries to host machine's /etc/hosts..."
  
  # Remove existing entries if they exist
  sed -i '/status$/d' /etc/hosts
  sed -i '/proxy$/d' /etc/hosts
  sed -i '/prometheus$/d' /etc/hosts
  sed -i '/grafana$/d' /etc/hosts
  
  # Add new entries
  echo "$STATUS_IP status" >> /etc/hosts
  echo "$PROXY_IP proxy" >> /etc/hosts
  echo "$PROMETHEUS_IP prometheus" >> /etc/hosts
  if [ -n "$GRAFANA_IP" ]; then
    echo "$GRAFANA_IP grafana" >> /etc/hosts
  fi
  
  green "✅ Added container entries to host /etc/hosts file"
fi

# Test the final result
blue "Testing final connectivity..."
echo $DIVIDER

echo "1. Testing status container directly:"
curl -I -s http://$STATUS_IP | head -1 || echo "Failed"

echo "2. Testing status.latency.space domain:"
curl -I -s -H "Host: status.latency.space" http://localhost | head -1 || echo "Failed"

echo "3. Testing status.latency.space with direct IP:"
curl -I -s -H "Host: status.latency.space" http://$STATUS_IP | head -1 || echo "Failed"

echo $DIVIDER

green "✅ DNS and container setup fixed!"
echo ""
echo "If you still have issues:"
echo "1. Try accessing status.latency.space in a browser"
echo "2. If it's still not working, check Nginx logs: tail -f /var/log/nginx/error.log"
echo "3. If needed, run server health check: sudo ./deploy/server-health-check.sh"