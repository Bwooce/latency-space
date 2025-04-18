#!/bin/bash
# Script to fix status container issues
# This script diagnoses and resolves issues with the status container

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

# Check if we're in the right directory
if [ ! -f "docker-compose.yml" ]; then
  red "Please run this script from the latency-space directory"
  echo "Try: cd /opt/latency-space && ./deploy/fix-status-container.sh"
  exit 1
fi

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

# Check if the status container is in the docker-compose.yml
blue "Checking docker-compose.yml for status service..."
if ! grep -q "status:" docker-compose.yml; then
  red "❌ Status service not found in docker-compose.yml"
  exit 1
fi
green "✅ Status service found in docker-compose.yml"

# Stop any existing status container
blue "Stopping any existing status containers..."
docker-compose stop status 2>/dev/null
docker rm -f $(docker ps -a -q --filter name=status) 2>/dev/null

# Check the ports used by the status container
blue "Checking port availability..."
if netstat -tln | grep -q ":3000 "; then
  yellow "⚠️ Port 3000 is already in use"
  echo "Process using port 3000:"
  lsof -i :3000
  yellow "Consider modifying docker-compose.yml to use a different port"
else
  green "✅ Port 3000 is available"
fi

# Rebuild the status container
blue "Rebuilding status container..."
# Check for docker-compose file versions
if [ -f "docker-compose.yml" ]; then
  COMPOSE_FILE="docker-compose.yml"
elif [ -f "docker-compose.yaml" ]; then
  COMPOSE_FILE="docker-compose.yaml"
else
  red "❌ docker-compose.yml file not found"
  echo "Available files in current directory:"
  ls -la
  exit 1
fi

blue "Using compose file: $COMPOSE_FILE"
# Try with explicit file parameter
docker-compose -f $COMPOSE_FILE build --no-cache status
if [ $? -ne 0 ]; then
  yellow "⚠️ Build failed with docker-compose, trying with docker compose (v2)..."
  # Try with Docker Compose v2 syntax
  docker compose -f $COMPOSE_FILE build --no-cache status
  if [ $? -ne 0 ]; then
    red "❌ Failed to build status container"
    echo "Docker compose error. Trying alternative approaches..."
    
    # Try building the status image directly
    blue "Trying direct Docker build..."
    if [ -d "status" ] && [ -f "status/Dockerfile" ]; then
      docker build -t latency-space-status ./status
      if [ $? -ne 0 ]; then
        red "❌ All build approaches failed"
        exit 1
      else
        green "✅ Direct image build successful"
      fi
    else
      red "❌ Status directory or Dockerfile not found"
      exit 1
    fi
  else
    green "✅ Status container rebuilt successfully using docker compose v2"
  fi
else
  green "✅ Status container rebuilt successfully"
fi

# Start the status container
blue "Starting status container..."
if [ -n "$COMPOSE_FILE" ]; then
  # Try docker-compose first
  docker-compose -f $COMPOSE_FILE up -d status
  if [ $? -ne 0 ]; then
    yellow "⚠️ Failed with docker-compose, trying docker compose v2..."
    docker compose -f $COMPOSE_FILE up -d status
    if [ $? -ne 0 ]; then
      red "❌ Failed to start status container with both docker-compose commands"
      
      # If we built a direct image, try running it directly
      if docker images | grep -q latency-space-status; then
        blue "Trying to run the container directly..."
        docker run -d --name latency-space-status \
          -p 3000:3000 \
          --network space-net \
          -e METRICS_URL=http://prometheus:9090 \
          latency-space-status
        if [ $? -ne 0 ]; then
          red "❌ All attempts to start the container failed"
          
          blue "Checking Docker network..."
          if ! docker network ls | grep -q space-net; then
            yellow "⚠️ space-net network not found, creating it..."
            docker network create space-net
          fi
          
          blue "Trying again with network..."
          docker run -d --name latency-space-status \
            -p 3000:3000 \
            --network space-net \
            -e METRICS_URL=http://prometheus:9090 \
            latency-space-status
          if [ $? -ne 0 ]; then
            red "❌ All attempts to start the container failed"
            exit 1
          fi
        fi
      else
        blue "Checking status container logs..."
        docker-compose -f $COMPOSE_FILE logs status || docker compose -f $COMPOSE_FILE logs status
        
        yellow "⚠️ Try checking if there's a port conflict or if another service is preventing it from starting"
        exit 1
      fi
    fi
  fi
else
  red "❌ COMPOSE_FILE variable not set. This should not happen."
  exit 1
fi

# Check if the container is running
if docker ps | grep -q "status\|latency-space-status"; then
  green "✅ Status container is now running"
else
  red "❌ Status container failed to start"
  blue "Container logs:"
  docker-compose -f $COMPOSE_FILE logs status 2>/dev/null || docker compose -f $COMPOSE_FILE logs status 2>/dev/null || echo "Could not retrieve logs"
  exit 1
fi

# Check if Nginx is configured correctly
blue "Checking Nginx configuration..."
NGINX_CONFIG="/etc/nginx/sites-enabled/latency.space"
if [ ! -f "$NGINX_CONFIG" ]; then
  yellow "⚠️ Nginx configuration not found at $NGINX_CONFIG"
  
  # Run the Nginx configuration install
  if [ -f "deploy/install-nginx-config.sh" ]; then
    blue "Running Nginx configuration install..."
    sudo bash deploy/install-nginx-config.sh
  else
    red "❌ Nginx install script not found"
    yellow "Try running: git pull to get the latest scripts"
    exit 1
  fi
else
  # Check if the Nginx configuration has the correct status container port
  if grep -q "status:3000" "$NGINX_CONFIG"; then
    green "✅ Nginx configuration has correct status container reference"
  else
    yellow "⚠️ Nginx configuration might not be pointing to status:3000"
    echo "Current configuration:"
    grep -A 20 "status.latency.space" "$NGINX_CONFIG"
    
    read -p "Do you want to fix the Nginx configuration now? (y/n): " fix_nginx
    if [[ "$fix_nginx" == "y" ]]; then
      if [ -f "deploy/install-nginx-config.sh" ]; then
        sudo bash deploy/install-nginx-config.sh
      else
        red "❌ Nginx install script not found"
        yellow "Try running: git pull to get the latest scripts"
      fi
    fi
  fi
fi

# Check Docker network connectivity
blue "Checking Docker network connectivity..."
NETWORK_NAME=$(docker inspect -f '{{range $key, $value := .NetworkSettings.Networks}}{{$key}}{{end}}' $(docker ps -q -f name=status))
if [ -z "$NETWORK_NAME" ]; then
  red "❌ Could not determine Docker network for status container"
else
  green "✅ Status container is on network: $NETWORK_NAME"
  
  # Check if proxy is on the same network
  if docker ps -q -f name=proxy &>/dev/null; then
    PROXY_NETWORK=$(docker inspect -f '{{range $key, $value := .NetworkSettings.Networks}}{{$key}}{{end}}' $(docker ps -q -f name=proxy))
    if [ "$PROXY_NETWORK" == "$NETWORK_NAME" ]; then
      green "✅ Proxy and status containers are on the same network"
    else
      red "❌ Proxy and status containers are on different networks"
      echo "Proxy: $PROXY_NETWORK vs Status: $NETWORK_NAME"
      
      yellow "⚠️ This will cause connectivity issues. Recreating all containers on the same network..."
      docker-compose down
      docker-compose up -d
    fi
  else
    yellow "⚠️ Proxy container is not running, starting it..."
    docker-compose up -d proxy
  fi
  
  # Test network connectivity
  echo ""
  blue "Testing container DNS resolution..."
  if docker exec $(docker ps -q -f name=proxy) getent hosts status &>/dev/null; then
    green "✅ Status hostname is resolvable from proxy container"
  else
    red "❌ Status hostname is NOT resolvable from proxy container"
    yellow "⚠️ This might indicate a Docker DNS issue"
    
    # Check Docker DNS
    echo ""
    blue "Checking Docker DNS configuration..."
    if grep -q '"dns": \["127.0.0.11"\]' /etc/docker/daemon.json 2>/dev/null; then
      green "✅ Docker DNS configuration exists"
    else
      yellow "⚠️ Setting up Docker DNS configuration..."
      mkdir -p /etc/docker
      echo '{
  "dns": ["127.0.0.11", "8.8.8.8", "8.8.4.4"],
  "dns-opts": ["ndots:1"]
}' > /etc/docker/daemon.json
      
      blue "Restarting Docker service to apply DNS changes..."
      systemctl restart docker
      
      # Restart containers after Docker restart
      blue "Restarting containers..."
      docker-compose up -d
    fi
  fi
fi

# Test the status endpoint
echo ""
blue "Testing status endpoint..."
if curl -s -I -m 5 http://localhost:3000 | grep -q "200 OK"; then
  green "✅ Status container is responding on localhost:3000"
else
  yellow "⚠️ Status container is not responding on localhost:3000"
fi

if curl -s -I -m 5 http://status.latency.space | grep -q "200 OK"; then
  green "✅ status.latency.space is working!"
else
  yellow "⚠️ status.latency.space is not responding correctly"
  echo "This might be due to DNS propagation delay or Nginx configuration issues"
  
  # Suggest direct testing
  echo ""
  blue "Try testing with direct IP and Host header:"
  SERVER_IP=$(curl -s -4 ifconfig.me || curl -s -4 icanhazip.com)
  echo "curl -H 'Host: status.latency.space' http://$SERVER_IP"
fi

# Final summary
echo $DIVIDER
blue "Summary of actions taken:"
echo "1. Rebuilt and restarted the status container"
echo "2. Verified Docker networking between containers"
echo "3. Ensured proper Nginx configuration"

green "✅ Fix script completed!"
echo ""
echo "If you're still experiencing issues, try:"
echo "1. Check server access logs: tail -f /var/log/nginx/access.log"
echo "2. Check error logs: tail -f /var/log/nginx/error.log"
echo "3. Verify DNS resolution: host status.latency.space"
echo "4. Run the comprehensive health check: ./deploy/server-health-check.sh"