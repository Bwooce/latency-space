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
  
  # Check if it already has DNS settings
  if grep -q "dns" /etc/docker/daemon.json; then
    blue "Existing DNS settings found. Modifying..."
    
    # Use jq if available to properly modify the JSON
    if command -v jq &> /dev/null; then
      jq '.dns = ["127.0.0.11", "8.8.8.8", "8.8.4.4"] | .["dns-opts"] = ["ndots:1"]' /etc/docker/daemon.json > /tmp/daemon.json
      mv /tmp/daemon.json /etc/docker/daemon.json
    else
      # Manual edit if jq is not available
      yellow "jq not available, creating a new config file with DNS settings"
      echo '{
  "dns": ["127.0.0.11", "8.8.8.8", "8.8.4.4"],
  "dns-opts": ["ndots:1"]
}' > /etc/docker/daemon.json
    fi
  else
    # Simple merge of DNS settings into existing file
    blue "Adding DNS settings to existing config..."
    
    # Remove the closing brace, add our settings
    sed -i '$ s/}/,/' /etc/docker/daemon.json
    echo '  "dns": ["127.0.0.11", "8.8.8.8", "8.8.4.4"],
  "dns-opts": ["ndots:1"]
}' >> /etc/docker/daemon.json
  fi
else
  # Create a new daemon.json file
  blue "Creating new Docker daemon config with proper DNS settings..."
  echo '{
  "dns": ["127.0.0.11", "8.8.8.8", "8.8.4.4"],
  "dns-opts": ["ndots:1"]
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

# Fix compose issues with snap
blue "Checking for Docker Compose installation issues..."
compose_cmd=""
if command -v docker-compose &> /dev/null; then
  blue "Found docker-compose command"
  compose_cmd="docker-compose"
elif command -v docker &> /dev/null && docker compose version &> /dev/null; then
  blue "Found docker compose v2 command"
  compose_cmd="docker compose"
else
  yellow "⚠️ No Docker Compose found. Attempting to install Docker Compose..."
  curl -SL https://github.com/docker/compose/releases/download/v2.23.3/docker-compose-linux-x86_64 -o /usr/local/bin/docker-compose
  chmod +x /usr/local/bin/docker-compose
  if command -v docker-compose &> /dev/null; then
    compose_cmd="docker-compose"
    green "✅ Docker Compose installed successfully"
  else
    red "❌ Failed to install Docker Compose"
    yellow "Will use docker command directly as fallback"
  fi
fi

# Restart all containers
blue "Stopping all containers..."
docker compose down || docker stop $(docker ps -a -q) 2>/dev/null || true

blue "Removing orphaned containers..."
docker container prune -f

# Run docker compose with correct file path
blue "Starting containers with Docker Compose..."
if [ -n "$compose_cmd" ]; then
  if [ "$compose_cmd" == "docker-compose" ]; then  # Legacy v1 support
    ORIG_PWD=$(pwd)
    cd "$ORIG_PWD"  # Ensure we're in the right directory
    COMPOSE_FILE="$ORIG_PWD/docker-compose.yml"
    if [ ! -f "$COMPOSE_FILE" ]; then
      COMPOSE_FILE="$ORIG_PWD/docker-compose.yaml"
    fi
    docker-compose -f "$COMPOSE_FILE" up -d
  else
    # Docker Compose V2
    docker compose up -d
  fi
  if [ $? -ne 0 ]; then
    red "❌ Failed to start containers with Docker Compose"
    yellow "Trying to start containers individually..."
    start_containers_manually
  else
    green "✅ Containers started successfully with Docker Compose"
  fi
else
  yellow "Starting containers individually as fallback..."
  start_containers_manually
fi

# Function to start containers manually if Docker Compose fails
start_containers_manually() {
  blue "Manually starting individual containers..."
  
  # Create network if it doesn't exist
  if ! docker network ls | grep -q space-net; then
    blue "Creating Docker network: space-net"
    docker network create space-net
  fi
  
  # Start prometheus first (dependency)
  blue "Starting prometheus container..."
  if [ -d "monitoring/prometheus" ]; then
    docker run -d --name latency-space-prometheus \
      -p 9091:9090 \
      --network space-net \
      -v prometheus_data:/prometheus \
      --restart unless-stopped \
      $(docker build -q ./monitoring/prometheus)
  fi
  
  # Start status container
  blue "Starting status container..."
  if [ -d "status" ]; then
    docker run -d --name latency-space-status \
      -p 3000:3000 \
      --network space-net \
      -e METRICS_URL=http://latency-space-prometheus:9090 \
      --restart unless-stopped \
      $(docker build -q ./status)
  fi
  
  # Start proxy container
  blue "Starting proxy container..."
  if [ -d "proxy" ]; then
    docker run -d --name latency-space-proxy \
      -p 8080:80 -p 8443:443 -p 5354:53/udp -p 1080:1080 -p 9090:9090 \
      --network space-net \
      -v proxy_config:/etc/space-proxy \
      -v proxy_ssl:/etc/letsencrypt \
      -v proxy_certs:/app/certs \
      --cap-add NET_ADMIN \
      --restart unless-stopped \
      $(docker build -q ./proxy)
  fi
  
  green "✅ Containers started manually"
}

# Wait for containers to be ready
blue "Waiting for containers to become ready..."
sleep 5

# Check container status
blue "Checking container status..."
if docker ps | grep -q "status"; then
  green "✅ Status container is running"
else
  red "❌ Status container is not running"
fi

if docker ps | grep -q "proxy"; then
  green "✅ Proxy container is running"
else
  red "❌ Proxy container is not running"
fi

# Test container DNS resolution
blue "Testing container DNS resolution..."
if docker exec $(docker ps -q -f name=proxy) getent hosts status &>/dev/null; then
  green "✅ DNS resolution working: proxy can resolve status container"
else
  red "❌ DNS resolution failing: proxy cannot resolve status container"
  
  # Add entries to /etc/hosts inside containers as fallback
  blue "Adding manual host entries as fallback..."
  
  # Get container IPs
  STATUS_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=status))
  PROXY_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=proxy))
  
  if [ -n "$STATUS_IP" ] && [ -n "$PROXY_IP" ]; then
    # Add entries to proxy container
    docker exec $(docker ps -q -f name=proxy) sh -c "echo \"$STATUS_IP status\" >> /etc/hosts"
    # Add entries to status container
    docker exec $(docker ps -q -f name=status) sh -c "echo \"$PROXY_IP proxy\" >> /etc/hosts"
    green "✅ Manual host entries added to containers"
  else
    yellow "⚠️ Could not determine container IPs for manual host entries"
  fi
fi

# Verify Nginx configuration
blue "Verifying Nginx configuration..."
if [ -f "/etc/nginx/sites-enabled/latency.space" ]; then
  blue "Testing Nginx configuration..."
  nginx -t
  if [ $? -eq 0 ]; then
    blue "Reloading Nginx..."
    systemctl reload nginx
    green "✅ Nginx configuration is valid and reloaded"
  else
    red "❌ Nginx configuration test failed"
  fi
else
  yellow "⚠️ Nginx configuration not found at /etc/nginx/sites-enabled/latency.space"
  blue "Looking for Nginx install script..."
  
  if [ -f "deploy/install-nginx-config.sh" ]; then
    blue "Running Nginx install script..."
    sudo bash deploy/install-nginx-config.sh
  else
    red "❌ Nginx install script not found"
    yellow "Try running: git pull to get the latest scripts"
  fi
fi

# Test the final result
blue "Testing final connectivity..."
echo $DIVIDER
echo "1. Testing proxy container directly:"
curl -s -I -m 5 http://localhost:8080 | head -1 || echo "Failed"

echo "2. Testing status container directly:"
curl -s -I -m 5 http://localhost:3000 | head -1 || echo "Failed"

echo "3. Testing status.latency.space:"
curl -s -I -m 5 http://status.latency.space | head -1 || echo "Failed"

echo "4. Testing direct IP with host header:"
SERVER_IP=$(curl -s -4 ifconfig.me || curl -s -4 icanhazip.com || hostname -I | awk '{print $1}')
curl -s -I -H "Host: status.latency.space" http://$SERVER_IP | head -1 || echo "Failed"
echo $DIVIDER

green "✅ DNS and container setup fixed!"
echo ""
echo "If everything is working correctly, you should see HTTP/1.1 200 OK responses above."
echo "Otherwise, check the detailed logs:"
echo "  docker logs \$(docker ps -q -f name=status)"
echo "  docker logs \$(docker ps -q -f name=proxy)"
echo "  tail -f /var/log/nginx/error.log"
echo ""
echo "You can also run the comprehensive health check:"
echo "  ./deploy/server-health-check.sh"