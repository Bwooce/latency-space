#!/bin/bash
# deploy/update.sh - Comprehensive deployment script for the VPS
# - Handles DNS configuration
# - Fixes Docker network bridge conflicts
# - Ensures HTTPS/SSL certificates are properly configured
# - Restarts services with proper configuration

set -e

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }
yellow() { echo -e "\033[0;33m$1\033[0m"; }
DIVIDER="----------------------------------------"

blue "🚀 Starting comprehensive deployment..."
echo $DIVIDER

# Check for root privileges
if [ "$(id -u)" -ne 0 ]; then
  red "❌ This script must be run as root"
  exit 1
fi

# Change to the project directory
cd /opt/latency-space || { red "❌ Failed to change directory"; exit 1; }

# Fix DNS if needed
blue "🔍 Checking DNS..."
if ! ping -c 1 github.com &> /dev/null; then
  yellow "⚠️ DNS issues detected, fixing..."
  
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
    red "❌ DNS still not working. Please check your DNS configuration."
    exit 1
  fi
fi

# Pull latest code
blue "📥 Pulling latest code from GitHub..."
git fetch origin
git reset --hard origin/main

# Clean up any stuck containers with very forceful approach
blue "🧹 Cleaning up any problematic containers..."
# Stop all containers 
docker ps -aq | xargs -r docker stop || true

# Fix Docker bridge network conflicts
blue "🌉 Checking for Docker bridge network conflicts..."
echo $DIVIDER

# Determine the main project bridge
PROJECT_NETWORK_NAME="latency-space_space-net"
blue "Looking for the main project bridge network: $PROJECT_NETWORK_NAME"

# Get all Docker bridge networks
BRIDGES=$(docker network ls --filter driver=bridge --format "{{.Name}},{{.ID}}")
echo "Docker networks:"
echo "$BRIDGES"
echo ""

# Get all the bridge interfaces from the system
blue "Listing system bridge interfaces..."
SYSTEM_BRIDGES=$(ip -j link show type bridge 2>/dev/null | jq -r '.[].ifname' 2>/dev/null || ip link | grep -E '^[0-9]+: br-' | cut -d':' -f2 | awk '{print $1}')
echo "$SYSTEM_BRIDGES"
echo ""

# Extract bridge IDs and check for conflicts
MAIN_BRIDGE_ID=""
CONFLICTING_BRIDGES=()
SUBNETS=()

# Get main project network info
MAIN_BRIDGE_INFO=$(docker network inspect $PROJECT_NETWORK_NAME 2>/dev/null || echo "")
if [ -n "$MAIN_BRIDGE_INFO" ]; then
  MAIN_BRIDGE_ID=$(echo "$MAIN_BRIDGE_INFO" | jq -r '.[0].Id' 2>/dev/null || echo "")
  MAIN_SUBNET=$(echo "$MAIN_BRIDGE_INFO" | jq -r '.[0].IPAM.Config[0].Subnet' 2>/dev/null || echo "")
  MAIN_GATEWAY=$(echo "$MAIN_BRIDGE_INFO" | jq -r '.[0].IPAM.Config[0].Gateway' 2>/dev/null || echo "")
  
  if [ -n "$MAIN_BRIDGE_ID" ] && [ -n "$MAIN_SUBNET" ]; then
    blue "✓ Found main project bridge: $PROJECT_NETWORK_NAME ($MAIN_BRIDGE_ID) - $MAIN_SUBNET"
    
    # Check for conflicts by listing all bridge interfaces with conflicting subnets
    for BRIDGE in $SYSTEM_BRIDGES; do
      if [[ "$BRIDGE" =~ br-([a-zA-Z0-9]+) ]]; then
        BR_SHORT_ID=${BASH_REMATCH[1]}
        
        # Skip if this is the main bridge
        if [[ "$BRIDGE" == "br-${MAIN_BRIDGE_ID:0:12}" ]]; then
          continue
        fi
        
        # Get IP address and subnet
        BR_IP=$(ip -j addr show dev $BRIDGE 2>/dev/null | jq -r '.[0].addr_info[] | select(.family=="inet") | .local' 2>/dev/null || ip addr show dev $BRIDGE | grep -w inet | awk '{print $2}' | cut -d'/' -f1)
        BR_SUBNET=$(ip -j addr show dev $BRIDGE 2>/dev/null | jq -r '.[0].addr_info[] | select(.family=="inet") | .prefixlen' 2>/dev/null || ip addr show dev $BRIDGE | grep -w inet | awk '{print $2}')
        
        # Check if the subnet matches our main network
        # For simplicity, we'll just check if the IP is in the 172.18.x.x range
        if [[ "$BR_IP" == 172.18.* ]]; then
          yellow "⚠️ Conflict detected: $BRIDGE has IP $BR_IP which conflicts with main network $MAIN_SUBNET"
          CONFLICTING_BRIDGES+=("$BRIDGE")
        fi
      fi
    done
    
    # Remove conflicting bridges
    if [ ${#CONFLICTING_BRIDGES[@]} -gt 0 ]; then
      blue "Resolving ${#CONFLICTING_BRIDGES[@]} network conflicts..."
      
      for BRIDGE in "${CONFLICTING_BRIDGES[@]}"; do
        yellow "Removing conflicting bridge: $BRIDGE"
        
        # Try to force bridge down
        ip link set dev $BRIDGE down 2>/dev/null || true
        
        # Attempt to delete the bridge interface
        if ip link delete dev $BRIDGE 2>/dev/null; then
          green "✅ Successfully removed conflicting bridge $BRIDGE"
        else
          yellow "⚠️ Could not remove bridge $BRIDGE immediately, will try more aggressive approach"
          
          # More aggressive approach
          ip link set dev $BRIDGE down 2>/dev/null || true
          sleep 1
          ip link delete dev $BRIDGE 2>/dev/null || true
          
          if ! ip link show dev $BRIDGE &>/dev/null; then
            green "✅ Successfully removed conflicting bridge $BRIDGE using aggressive approach"
          else
            red "❌ Could not remove bridge $BRIDGE, will try to work around this"
          fi
        fi
      done
      
      green "✅ Network conflict resolution complete"
    else
      green "✅ No network conflicts detected"
    fi
  else
    yellow "⚠️ Could not find main project network info, will create it during startup"
  fi
else
  yellow "⚠️ Project network not found, will be created during startup"
fi

echo $DIVIDER

# Ensure certbot is installed and SSL certificates are present
blue "🔒 Checking SSL/TLS certificates..."

if ! command -v certbot &> /dev/null; then
  yellow "⚠️ Certbot not found, installing..."
  if command -v apt-get &> /dev/null; then
    apt-get update
    apt-get install -y certbot python3-certbot-nginx
  elif command -v dnf &> /dev/null; then
    dnf install -y certbot python3-certbot-nginx
  else
    red "❌ Package manager not found. Please install certbot manually."
    exit 1
  fi
  green "✅ Certbot installed successfully"
fi

# Check for existing certificates
SSL_DIR="/etc/letsencrypt/live/latency.space"
if [ ! -d "$SSL_DIR" ]; then
  yellow "⚠️ SSL certificates not found, will attempt to obtain them"
  
  # Check if we're running interactively
  if [ -t 0 ]; then
    # Interactive run
    read -p "Do you want to run certbot to obtain SSL certificates? (y/n): " run_certbot
    if [[ "$run_certbot" == "y" ]]; then
      certbot --nginx -d latency.space -d www.latency.space -d status.latency.space \
        -d mars.latency.space -d venus.latency.space -d mercury.latency.space \
        -d jupiter.latency.space -d saturn.latency.space -d uranus.latency.space \
        -d neptune.latency.space -d pluto.latency.space
    fi
  else
    # Non-interactive run (GitHub Actions)
    yellow "⚠️ Non-interactive run, skipping certbot certificate request"
    yellow "⚠️ You'll need to run certbot manually to obtain certificates"
  fi
else
  green "✅ SSL certificates exist in $SSL_DIR"
  
  # Check certificate expiration
  CERT_DATE=$(openssl x509 -enddate -noout -in $SSL_DIR/fullchain.pem | cut -d= -f2)
  CERT_EXPIRY=$(date -d "$CERT_DATE" +%s)
  NOW=$(date +%s)
  DAYS_REMAINING=$(( ($CERT_EXPIRY - $NOW) / 86400 ))
  
  if [ $DAYS_REMAINING -lt 30 ]; then
    yellow "⚠️ SSL certificate expires in $DAYS_REMAINING days, attempting renewal"
    certbot renew --quiet
  else
    green "✅ SSL certificate valid for $DAYS_REMAINING more days"
  fi
fi

echo $DIVIDER

# System is using the standard Docker installation from packages
# Snap Docker has been removed due to containerd stability issues

# Check what's using the ports we need BEFORE stopping existing containers
blue "🔍 Checking for port conflicts with external services..."
REQUIRED_PORTS="8081 8444 9099 1080 1081 1082 1083 1084 1085 2081 2084 3001 3003 3080 3084 9092 9100 9101 9102 9103 9104 9105 9201 9204 9300 9304"

CONFLICTS_FOUND=false
for port in $REQUIRED_PORTS; do
  # Check if port is in use by non-docker processes
  # Use || true to prevent set -e from exiting when grep finds nothing
  PORT_INFO=$(ss -tlnp 2>/dev/null | grep ":$port " | grep -v docker-proxy || true)
  if [ -z "$PORT_INFO" ]; then
    # Try lsof if ss didn't find anything
    PORT_INFO=$(lsof -i :$port 2>/dev/null | grep -v docker-proxy || true)
  fi

  if [ -n "$PORT_INFO" ]; then
    yellow "⚠️  Port $port is in use by NON-DOCKER process:"
    echo "$PORT_INFO"
    CONFLICTS_FOUND=true
  fi
done

if [ "$CONFLICTS_FOUND" = true ]; then
  red "❌ Port conflicts detected with external services. Please resolve before deploying."
  exit 1
fi

green "✅ No external port conflicts detected"

# Restart the containers
blue "🔄 Restarting all containers..."
cd /opt/latency-space
docker compose down
blue "🏗️ Building all proxy images..."
docker compose build --no-cache
docker compose up -d

# Check container status with detailed output
echo ""
blue "📊 Checking container status..."
docker compose ps -a

# Count running containers
RUNNING=$(docker compose ps | grep -c "Up" || echo "0")
TOTAL=$(docker compose config --services | wc -l)

echo "Running: $RUNNING / Total services: $TOTAL"

if [ "$RUNNING" -eq 0 ]; then
  red "❌ No containers are running!"
  echo ""
  blue "Container logs for failed services:"
  docker compose logs --tail=50
  exit 1
elif [ "$RUNNING" -lt "$TOTAL" ]; then
  yellow "⚠️ Warning: Only $RUNNING out of $TOTAL containers are running"
  echo ""
  blue "Checking which containers failed:"
  docker compose ps -a | grep -v " Up "
  echo ""
  blue "Attempting to start failed containers individually to see error messages:"

  # Get list of containers that aren't running
  FAILED_SERVICES=$(docker compose ps -a --format json | jq -r 'select(.State != "running") | .Service' 2>/dev/null || docker compose ps -a | grep -v " Up " | awk '{print $1}' | grep latency-space | sed 's/latency-space-\(.*\)-[0-9]*/\1/')

  for service in $FAILED_SERVICES; do
    echo ""
    blue "=== Trying to start $service ==="
    docker compose up -d $service 2>&1 | tail -20
    sleep 2
    echo "Status after attempt:"
    docker compose ps $service 2>&1
    if docker ps | grep -q "$service"; then
      echo "Logs:"
      docker compose logs --tail=20 $service 2>&1
    fi
  done
else
  green "✅ All $TOTAL containers started successfully"
fi

# Update Nginx configuration with correct container IPs
blue "🔄 Updating Nginx configuration..."
if [ -f "deploy/update-nginx.sh" ]; then
  ./deploy/update-nginx.sh
else
  yellow "⚠️ update-nginx.sh script not found, skipping Nginx config update"
fi

# Create proper symlinks for shared data between packages
blue "🔄 Setting up symlinks for shared celestial object data..."
if [ -d "tools" ] && [ -d "proxy/src" ]; then
  # Remove any existing duplicated files
  rm -f tools/objects_data.go
  rm -f tools/models.go
  
  # Create symlinks to use the original data without duplication
  ln -sf ../proxy/src/models.go tools/models.go
  ln -sf ../proxy/src/objects_data.go tools/objects_data.go
  
  green "✅ Created symlinks for shared celestial object data"
else
  yellow "⚠️ tools or proxy directories not found, skipping symlink creation"
fi

# Check if everything is running
blue "🔍 Checking if containers are running..."
sleep 10
if docker compose ps | grep -q "Up"; then
  green "✅ Containers are running properly!"
else
  red "❌ Containers failed to start. Checking logs..."
  docker compose logs
  exit 1
fi

# Connectivity tests - using HTTPS when available
blue "🔌 Testing connectivity..."
echo $DIVIDER

# Check if SOCKS proxy is working
blue "🧦 Testing SOCKS proxy..."
if nc -z localhost 1080; then
  green "✅ SOCKS proxy is running on port 1080!"
else
  red "❌ SOCKS proxy is not running on port 1080!"
  exit 1
fi

# Check if HTTP proxy is working (try HTTPS first, then fallback to HTTP)
blue "🌐 Testing HTTP/HTTPS proxy..."
if curl -s --max-time 5 --insecure https://localhost:8444 &> /dev/null; then
  green "✅ HTTPS proxy is running on port 8444!"
elif curl -s --max-time 5 http://localhost:8081 &> /dev/null; then
  green "✅ HTTP proxy is running on port 8081!"
else
  red "❌ HTTP/HTTPS proxy is not running!"
  exit 1
fi

# Check if proxy is accessible through Nginx
blue "🌐 Testing Nginx proxy (HTTPS if available)..."
# Try HTTPS first, then fallback to HTTP
if curl -s --max-time 5 --insecure https://mars.latency.space &> /dev/null; then
  green "✅ HTTPS proxy is accessible through Nginx!"
elif curl -s --max-time 5 -H "Host: mars.latency.space" http://localhost:80 &> /dev/null; then
  green "✅ HTTP proxy is accessible through Nginx!"
else
  yellow "⚠️ Warning: Proxy may not be accessible through Nginx"
fi

echo $DIVIDER
green "✅ Deployment completed successfully!"
blue "🔍 If you encounter any issues, check the following:"
echo "  - Nginx configuration: /etc/nginx/sites-available/latency.space"
echo "  - Container logs: docker compose logs"
echo "  - Nginx logs: tail -f /var/log/nginx/error.log"
echo "  - SSL certificates: ls -la /etc/letsencrypt/live/latency.space"