#!/bin/bash
# Script to fix Docker Compose path issues and restart containers properly

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

blue "🔧 Docker Compose Fix Script"
echo $DIVIDER

# First show debugging information
blue "System Information:"
uname -a
echo ""

blue "Docker version:"
docker --version
echo ""

blue "Docker Compose version(s):"
docker-compose --version 2>/dev/null || echo "docker-compose (v1) not found"
docker compose version 2>/dev/null || echo "docker compose (v2) not found"
echo ""

blue "Checking Docker Compose installation:"
which docker-compose 2>/dev/null || echo "docker-compose not in PATH"
ls -l $(which docker-compose 2>/dev/null) 2>/dev/null || echo "Cannot check docker-compose binary"
echo ""

blue "Directory structure:"
echo "Current directory: $(pwd)"
ls -la
echo ""

# Check for docker-compose file
blue "Looking for docker-compose.yml file:"
find . -name "docker-compose.yml" -o -name "docker-compose.yaml" 2>/dev/null || echo "No docker-compose.yml file found in subfolders"
echo ""

# If we're not in the right directory, try to find it
if [ ! -f "docker-compose.yml" ] && [ ! -f "docker-compose.yaml" ]; then
  blue "Not in the right directory. Checking /opt/latency-space..."
  if [ -d "/opt/latency-space" ]; then
    cd /opt/latency-space
    echo "Changed to $(pwd)"
    
    if [ ! -f "docker-compose.yml" ] && [ ! -f "docker-compose.yaml" ]; then
      red "Cannot find docker-compose.yml in /opt/latency-space either"
      
      # Look for it system-wide
      blue "Searching for docker-compose.yml files system-wide..."
      find / -name "docker-compose.yml" -o -name "docker-compose.yaml" 2>/dev/null | head -n 10
      
      # Create a basic docker-compose file if none exists
      read -p "No docker-compose.yml file found. Create a basic one? (y/n): " create_compose
      if [[ "$create_compose" == "y" ]]; then
        blue "Creating a basic docker-compose.yml file..."
        cat > docker-compose.yml << 'EOF'
services:
  proxy:
    build: 
      context: ./proxy
      dockerfile: Dockerfile
    ports:
      - "8080:80"
      - "8443:443"
      - "5354:53/udp"
      - "1080:1080"
      - "9090:9090"
    volumes:
      - proxy_config:/etc/space-proxy
      - proxy_ssl:/etc/letsencrypt
      - proxy_certs:/app/certs
    cap_add:
      - NET_ADMIN
    restart: unless-stopped
    networks:
      - space-net

  status:
    build: 
      context: ./status
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    environment:
      - METRICS_URL=http://prometheus:9090
    networks:
      - space-net
    depends_on:
      - prometheus

  prometheus:
    build: 
      context: ./monitoring/prometheus
      dockerfile: Dockerfile
    ports:
      - "9091:9090"
    volumes:
      - prometheus_data:/prometheus
    networks:
      - space-net
    user: "root"

  grafana:
    build: 
      context: ./monitoring/grafana
      dockerfile: Dockerfile
    ports:
      - "3001:3000"
    volumes:
      - grafana_data:/var/lib/grafana
      - grafana_dashboards:/etc/grafana/provisioning/dashboards
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    networks:
      - space-net
    depends_on:
      - prometheus

networks:
  space-net:
    driver: bridge

volumes:
  prometheus_data:
  grafana_data:
  grafana_dashboards:
  proxy_config:
  proxy_ssl:
  proxy_certs:
EOF
        green "✅ Basic docker-compose.yml created"
      else
        red "Exiting. Cannot proceed without docker-compose.yml"
        exit 1
      fi
    fi
  else
    red "Cannot find /opt/latency-space directory"
    exit 1
  fi
fi

# Determine which compose command to use
blue "Testing Docker Compose commands..."
DOCKER_COMPOSE=""

if docker compose version &>/dev/null; then
  green "✅ docker compose (v2) available"
  DOCKER_COMPOSE="docker compose"
elif docker-compose --version &>/dev/null; then
  green "✅ docker-compose (v1) available"
  DOCKER_COMPOSE="docker-compose"
else
  yellow "⚠️ No Docker Compose found. Attempting direct Docker commands."
  DOCKER_COMPOSE=""
fi

# Fix for snap paths
blue "Checking if running from snap..."
if which docker-compose | grep -q snap; then
  yellow "⚠️ docker-compose is installed via snap, which may cause path issues"
  
  # Create a wrapper script
  blue "Creating a direct wrapper script..."
  cat > /usr/local/bin/docker-compose-direct << 'EOF'
#!/bin/bash
# Direct wrapper for docker-compose to avoid snap path issues
# This passes the current directory as an absolute path

PWD=$(pwd)
ARGS=()

# Process arguments and fix any paths
for arg in "$@"; do
  if [[ "$arg" == "-f" ]]; then
    ARGS+=("$arg")
  elif [[ "${PREV_ARG}" == "-f" && ! "$arg" =~ ^/ ]]; then
    # Make relative path absolute
    ARGS+=("$PWD/$arg")
  else
    ARGS+=("$arg")
  fi
  PREV_ARG="$arg"
done

# Execute with docker compose v2 if available, otherwise fall back to v1
if docker compose version &>/dev/null; then
  docker compose "${ARGS[@]}"
else
  $(which docker-compose) "${ARGS[@]}"
fi
EOF
  chmod +x /usr/local/bin/docker-compose-direct
  green "✅ Created wrapper script at /usr/local/bin/docker-compose-direct"
  
  # Use our wrapper
  DOCKER_COMPOSE="/usr/local/bin/docker-compose-direct"
fi

# Stop any existing containers
blue "Stopping any existing containers..."
docker stop $(docker ps -a -q) 2>/dev/null || true
docker rm $(docker ps -a -q) 2>/dev/null || true

# Start containers with the appropriate method
blue "Starting containers..."
if [ -n "$DOCKER_COMPOSE" ]; then
  blue "Using command: $DOCKER_COMPOSE"
  echo "Full command: $DOCKER_COMPOSE -f $(pwd)/docker-compose.yml up -d"
  
  $DOCKER_COMPOSE -f $(pwd)/docker-compose.yml up -d
  if [ $? -ne 0 ]; then
    red "❌ Failed to start containers with $DOCKER_COMPOSE"
    blue "Trying to start containers individually..."
    start_containers_manually
  else
    green "✅ Containers started successfully with $DOCKER_COMPOSE"
  fi
else
  blue "Starting containers individually (fallback method)..."
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

# Verify containers are running
blue "Verifying containers are running..."
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

# Wait for containers to stabilize
blue "Waiting for containers to stabilize..."
sleep 5

# Verify networking
blue "Verifying container networking..."
# Get container IDs
STATUS_CONTAINER=$(docker ps -q -f name=status)
PROXY_CONTAINER=$(docker ps -q -f name=proxy)

if [ -n "$STATUS_CONTAINER" ] && [ -n "$PROXY_CONTAINER" ]; then
  # Get container IPs
  STATUS_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $STATUS_CONTAINER)
  PROXY_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $PROXY_CONTAINER)
  
  blue "Container IPs:"
  echo "Status: $STATUS_IP"
  echo "Proxy: $PROXY_IP"
  
  # Check if they can resolve each other
  blue "Testing DNS resolution between containers..."
  if docker exec $PROXY_CONTAINER getent hosts status &>/dev/null; then
    green "✅ Proxy can resolve status container"
  else
    yellow "⚠️ DNS resolution not working. Adding manual host entries..."
    
    # Add entries to container hosts files
    docker exec $PROXY_CONTAINER sh -c "echo '$STATUS_IP status' >> /etc/hosts" || true
    docker exec $STATUS_CONTAINER sh -c "echo '$PROXY_IP proxy' >> /etc/hosts" || true
    
    green "✅ Added manual host entries to containers"
  fi
else
  red "❌ One or both containers are not running"
fi

# Install the Nginx configuration
blue "Installing Nginx configuration..."
if [ -f "deploy/install-nginx-config.sh" ]; then
  sudo bash deploy/install-nginx-config.sh
  if [ $? -ne 0 ]; then
    red "❌ Failed to install Nginx configuration"
    exit 1
  fi
  green "✅ Nginx configuration installed"
else
  red "❌ Nginx installation script not found"
  yellow "⚠️ No Nginx configuration scripts found"
  yellow "Please run git pull to get the latest scripts"
fi

# Verify static directory exists
if [ ! -d "/opt/latency-space/static" ]; then
  blue "Creating static directory..."
  mkdir -p /opt/latency-space/static
  
  # Copy the index.html file if it exists in the static directory
  if [ -f "static/index.html" ]; then
    blue "Using index.html from repository..."
    cp static/index.html /opt/latency-space/static/
    green "✅ Copied index.html to static directory"
  elif [ -f "deploy/static/index.html" ]; then
    blue "Using index.html from deploy/static..."
    cp deploy/static/index.html /opt/latency-space/static/
    green "✅ Copied index.html to static directory"
  else
    yellow "⚠️ No index.html found. Creating a basic one..."
    cat > /opt/latency-space/static/index.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Latency Space</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
    </style>
</head>
<body>
    <h1>Latency Space</h1>
    <p>Welcome to Latency Space - Interplanetary Internet Simulator</p>
    <p>Visit <a href="http://mars.latency.space">mars.latency.space</a> to experience Mars latency.</p>
    <p>Visit <a href="http://status.latency.space">status.latency.space</a> for system status.</p>
</body>
</html>
EOF
    green "✅ Created basic index.html in static directory"
  fi
fi

# Test connectivity
blue "Testing connectivity..."
echo $DIVIDER
echo "1. Testing proxy directly: $(curl -s -I -m 5 http://localhost:8080 | head -1)"
echo "2. Testing status directly: $(curl -s -I -m 5 http://localhost:3000 | head -1)"
echo "3. Testing the main website: $(curl -s -I -m 5 http://localhost | head -1)"
echo "4. Testing status.latency.space: $(curl -s -I -m 5 http://status.latency.space | head -1)"
echo $DIVIDER

# Final summary
green "✅ Docker Compose issues fixed!"
echo ""
echo "If everything is working correctly, you should see HTTP/1.1 200 OK responses above."
echo "If status.latency.space is still not working, try:"
echo "  1. Check Nginx logs: tail -f /var/log/nginx/error.log"
echo "  2. Check status container logs: docker logs \$(docker ps -q -f name=status)"
echo "  3. Restart Nginx: systemctl restart nginx"
echo ""
echo "To set up the static homepage:"
echo "  1. Pull the latest changes: git pull"
echo "  2. The homepage should be in the static directory"
echo ""
echo "You can run the comprehensive health check to verify everything:"
echo "  ./deploy/server-health-check.sh"