#!/bin/bash
# Script to fix Docker DNS resolution issues

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }
yellow() { echo -e "\033[0;33m$1\033[0m"; }

# Check if script is run as root
if [ "$EUID" -ne 0 ]; then
  red "Please run this script as root"
  echo "Try: sudo $0"
  exit 1
fi

blue "🔧 Docker DNS Resolution Fix"
echo "----------------------------------------"

# Create a backup of Docker daemon.json if it exists
if [ -f "/etc/docker/daemon.json" ]; then
  blue "Creating backup of daemon.json..."
  cp /etc/docker/daemon.json /etc/docker/daemon.json.bak.$(date +%s)
  green "✅ Backup created"
fi

# Create or update Docker daemon.json
blue "Creating/updating Docker daemon.json with proper DNS settings..."
mkdir -p /etc/docker
cat > /etc/docker/daemon.json << EOF
{
  "dns": ["8.8.8.8", "8.8.4.4"],
  "dns-opts": ["ndots:1"],
  "dns-search": ["latency.space"],
  "bip": "172.18.0.1/24",
  "default-address-pools": [
    {"base": "172.18.0.0/16", "size": 24}
  ]
}
EOF

green "✅ Docker daemon.json created/updated"

# Restart Docker service to apply changes
blue "Restarting Docker service..."
systemctl restart docker
if [ $? -eq 0 ]; then
  green "✅ Docker service restarted successfully"
else
  red "❌ Failed to restart Docker service"
  exit 1
fi

# Wait for Docker to become available
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
  exit 1
fi
echo ""
green "✅ Docker is now available"

# Check if we're in the right directory
cd /opt/latency-space || { red "❌ Could not change to /opt/latency-space directory"; exit 1; }

# Stop all containers
blue "Stopping all containers..."
docker-compose down || docker compose down || docker stop $(docker ps -q)
green "✅ All containers stopped"

# Create hosts file entries for containers
blue "Creating custom hosts file for containers..."
cat > /tmp/container-hosts << EOF
127.0.0.1 localhost
::1 localhost ip6-localhost ip6-loopback
fe00::0 ip6-localnet
ff00::0 ip6-mcastprefix
ff02::1 ip6-allnodes
ff02::2 ip6-allrouters

# Container DNS entries
172.18.0.2 proxy
172.18.0.3 prometheus
172.18.0.4 grafana
172.18.0.5 status
EOF
green "✅ Custom hosts file created"

# Start the containers
blue "Starting containers..."
docker-compose up -d || docker compose up -d
if [ $? -eq 0 ]; then
  green "✅ Containers started successfully"
else
  red "❌ Failed to start containers"
  exit 1
fi

# Copy hosts file to containers
blue "Copying hosts file to containers..."
for container in $(docker ps -q); do
  name=$(docker inspect --format '{{.Name}}' $container | sed 's/^\///')
  blue "Configuring $name..."
  
  # Copy hosts file
  docker cp /tmp/container-hosts $container:/etc/hosts
  if [ $? -eq 0 ]; then
    green "✅ Copied hosts file to $name"
  else
    yellow "⚠️ Failed to copy hosts file to $name"
  fi
  
  # Add nameserver configuration
  docker exec $container sh -c "echo 'nameserver 8.8.8.8' > /etc/resolv.conf"
  docker exec $container sh -c "echo 'nameserver 8.8.4.4' >> /etc/resolv.conf"
  if [ $? -eq 0 ]; then
    green "✅ Updated resolv.conf in $name"
  else
    yellow "⚠️ Failed to update resolv.conf in $name"
  fi
done

# Verify DNS resolution in containers
blue "Verifying DNS resolution in containers..."
for container in $(docker ps -q); do
  name=$(docker inspect --format '{{.Name}}' $container | sed 's/^\///')
  echo "Testing resolution in $name:"
  
  # Test DNS resolution of status container
  if docker exec $container cat /etc/hosts | grep -q status; then
    green "✅ $name has status in hosts file"
  else
    red "❌ $name missing status in hosts file"
  fi
  
  # Test nslookup if available
  if docker exec $container which nslookup &>/dev/null; then
    docker exec $container nslookup status || echo "nslookup failed"
  elif docker exec $container which dig &>/dev/null; then
    docker exec $container dig status || echo "dig failed"
  elif docker exec $container which getent &>/dev/null; then
    docker exec $container getent hosts status || echo "getent failed"
  else
    yellow "⚠️ No DNS tools available in $name"
  fi
done

# Try pinging between containers
blue "Testing connectivity between containers..."
if docker exec $(docker ps -q -f name=proxy) ping -c 1 status &>/dev/null; then
  green "✅ Proxy can ping status"
else
  yellow "⚠️ Proxy cannot ping status - this might be expected if ping is not installed"
fi

# Test access to status dashboard from proxy container
blue "Testing HTTP access from proxy to status..."
if docker exec $(docker ps -q -f name=proxy) which curl &>/dev/null; then
  docker exec $(docker ps -q -f name=proxy) curl -I http://status 2>/dev/null || echo "HTTP request failed"
elif docker exec $(docker ps -q -f name=proxy) which wget &>/dev/null; then
  docker exec $(docker ps -q -f name=proxy) wget -q -O /dev/null http://status 2>/dev/null || echo "HTTP request failed"
else
  yellow "⚠️ No HTTP tools available in proxy container"
fi

# Verify Nginx configuration
blue "Verifying Nginx configuration..."
nginx -t
if [ $? -eq 0 ]; then
  blue "Reloading Nginx..."
  systemctl reload nginx
  green "✅ Nginx configuration is valid and service reloaded"
else
  red "❌ Nginx configuration test failed"
  exit 1
fi

# Test status dashboard
blue "Testing status dashboard directly via port 3000..."
curl -I http://localhost:3000 || echo "Failed to connect"

blue "Testing status dashboard via Nginx..."
curl -I -H "Host: status.latency.space" http://localhost || echo "Failed to connect"

echo "----------------------------------------"
green "✅ Docker DNS fix completed!"
echo ""
echo "1. Try accessing status.latency.space in a browser"
echo "2. If it's still not working, check Nginx logs: tail -f /var/log/nginx/error.log"
echo "3. If needed, run server health check: sudo ./deploy/server-health-check.sh"