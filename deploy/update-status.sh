#!/bin/bash
# Script to fix status.latency.space setup

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }
yellow() { echo -e "\033[0;33m$1\033[0m"; }

blue "🛠️ Setting up status.latency.space"

# 1. First ensure the status container is running
blue "🔄 Starting status container..."
docker compose up -d status
if [ $? -ne 0 ]; then
  red "❌ Failed to start status container"
  exit 1
fi

# 2. Create and configure Nginx for status.latency.space
blue "📝 Configuring Nginx for status.latency.space..."

# Create the Nginx configuration file if it doesn't exist
NGINX_DIR="/etc/nginx"
SITES_AVAILABLE="$NGINX_DIR/sites-available"
SITES_ENABLED="$NGINX_DIR/sites-enabled"
CONFIG_FILE="$SITES_AVAILABLE/latency.space"

# Check if file exists and make a backup if it does
if [ -f "$CONFIG_FILE" ]; then
  blue "💾 Making backup of existing Nginx configuration..."
  cp "$CONFIG_FILE" "$CONFIG_FILE.bak"
  green "✅ Backup created at $CONFIG_FILE.bak"
fi

# Check if the file already contains status.latency.space configuration
if grep -q "server_name status.latency.space" "$CONFIG_FILE" 2>/dev/null; then
  blue "🔍 Found existing status.latency.space configuration in Nginx"
else
  blue "➕ Adding status.latency.space configuration to Nginx..."
  
  # Get the existing server configuration
  if [ -f "$CONFIG_FILE" ]; then
    # Check if the status.latency.space server block already exists
    if ! grep -q "server_name status.latency.space" "$CONFIG_FILE"; then
      # Append status server block to the end of the file
      cat >> "$CONFIG_FILE" << 'EOF'

# Server for status dashboard
server {
    listen 80;
    server_name status.latency.space;
    
    # Global rate limiting
    limit_req zone=ip burst=10 nodelay;
    limit_conn addr 5;
    
    location / {
        proxy_pass http://status:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        
        # Increase timeouts for dashboard operations
        proxy_connect_timeout 10s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}
EOF
      green "✅ Added status.latency.space server block to Nginx configuration"
    fi
  else
    red "❌ Nginx configuration file not found at $CONFIG_FILE"
    yellow "⚠️ Creating a new configuration file..."
    
    # Create basic configuration with status server
    mkdir -p "$SITES_AVAILABLE"
    cat > "$CONFIG_FILE" << 'EOF'
# Define rate limiting zones
limit_req_zone $binary_remote_addr zone=ip:10m rate=5r/m;
limit_conn_zone $binary_remote_addr zone=addr:10m;

# Server for base domain latency.space
server {
    listen 80;
    server_name latency.space www.latency.space;
    
    # Global rate limiting
    limit_req zone=ip burst=20 nodelay;
    limit_conn addr 10;
    
    location / {
        proxy_pass http://proxy:80;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Destination $host;
        proxy_cache_bypass $http_upgrade;
    }
}

# Server for all other .latency.space subdomains
server {
    listen 80;
    server_name ~^[^.]+\.latency\.space$ ~^[^.]+\.[^.]+\.latency\.space$ ~^[^.]+\.[^.]+\.[^.]+\.latency\.space$;
    
    # Global rate limiting
    limit_req zone=ip burst=10 nodelay;
    limit_conn addr 5;
    
    location / {
        proxy_pass http://proxy:80;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Destination $host;
        proxy_cache_bypass $http_upgrade;
    }
}

# Server for status dashboard
server {
    listen 80;
    server_name status.latency.space;
    
    # Global rate limiting
    limit_req zone=ip burst=10 nodelay;
    limit_conn addr 5;
    
    location / {
        proxy_pass http://status:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        
        # Increase timeouts for dashboard operations
        proxy_connect_timeout 10s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}
EOF
    green "✅ Created new Nginx configuration with status.latency.space server block"
  fi
fi

# Create symbolic link in sites-enabled if needed
if [ ! -L "$SITES_ENABLED/latency.space" ]; then
  blue "🔗 Creating symbolic link in sites-enabled..."
  mkdir -p "$SITES_ENABLED"
  ln -sf "$CONFIG_FILE" "$SITES_ENABLED/latency.space"
  green "✅ Created symbolic link"
fi

# Test Nginx configuration and reload
blue "🔍 Testing Nginx configuration..."
nginx -t
if [ $? -eq 0 ]; then
  blue "🔄 Reloading Nginx..."
  systemctl reload nginx
  green "✅ Nginx reloaded successfully"
else
  red "❌ Nginx configuration test failed"
  exit 1
fi

# 3. Fix local DNS resolution
blue "🔧 Fixing local DNS resolution..."

# Check if systemd-resolved is in use
if [ -L "/etc/resolv.conf" ]; then
  blue "🔍 System is using systemd-resolved"
  
  # Create a better systemd-resolved config
  cat > /etc/systemd/resolved.conf << 'EOF'
[Resolve]
DNS=8.8.8.8 8.8.4.4 1.1.1.1
FallbackDNS=9.9.9.9 149.112.112.112
DNSStubListener=yes
Cache=yes
EOF
  
  # Restart systemd-resolved
  systemctl restart systemd-resolved
  green "✅ systemd-resolved restarted with new configuration"
else
  # Direct modification of resolv.conf
  blue "🔍 Updating /etc/resolv.conf directly"
  echo "nameserver 8.8.8.8" > /etc/resolv.conf
  echo "nameserver 8.8.4.4" >> /etc/resolv.conf
  echo "nameserver 1.1.1.1" >> /etc/resolv.conf
  green "✅ resolv.conf updated with public DNS servers"
fi

# 4. Fix /etc/hosts if it's using localhost
if grep -q "status.latency.space" /etc/hosts; then
  blue "🔄 Removing status.latency.space from /etc/hosts..."
  sed -i '/status\.latency\.space/d' /etc/hosts
  green "✅ Removed localhost entries for status.latency.space"
fi

# 5. Add status.latency.space to Docker DNS configuration
blue "🐳 Updating Docker DNS configuration..."

DOCKER_CONF="/etc/docker/daemon.json"
if [ -f "$DOCKER_CONF" ]; then
  # Backup the existing file
  cp "$DOCKER_CONF" "${DOCKER_CONF}.bak"
  
  # If the file already has DNS settings, ensure they're correct
  if grep -q "dns" "$DOCKER_CONF"; then
    blue "🔍 Found existing DNS configuration in Docker"
    # Use jq to update if available, otherwise manual modification
    if command -v jq >/dev/null 2>&1; then
      jq '.dns = ["8.8.8.8", "8.8.4.4", "1.1.1.1"]' "$DOCKER_CONF" > "${DOCKER_CONF}.new"
      mv "${DOCKER_CONF}.new" "$DOCKER_CONF"
    else
      # Manual regex replacement as a fallback
      sed -i 's/"dns"\s*:\s*\[[^]]*\]/"dns": ["8.8.8.8", "8.8.4.4", "1.1.1.1"]/' "$DOCKER_CONF"
    fi
  else
    # Add DNS configuration if not present
    if command -v jq >/dev/null 2>&1; then
      jq '. + {"dns": ["8.8.8.8", "8.8.4.4", "1.1.1.1"]}' "$DOCKER_CONF" > "${DOCKER_CONF}.new"
      mv "${DOCKER_CONF}.new" "$DOCKER_CONF"
    else
      # Add DNS config with sed as a fallback
      sed -i 's/{/{\"dns\":[\"8.8.8.8\",\"8.8.4.4\",\"1.1.1.1\"],/' "$DOCKER_CONF"
    fi
  fi
else
  # Create new docker daemon.json
  mkdir -p /etc/docker
  cat > "$DOCKER_CONF" << 'EOF'
{
  "dns": ["8.8.8.8", "8.8.4.4", "1.1.1.1"]
}
EOF
fi

blue "🔄 Restarting Docker to apply DNS changes..."
systemctl restart docker
if [ $? -eq 0 ]; then
  green "✅ Docker restarted successfully"
else
  red "❌ Failed to restart Docker"
  yellow "⚠️ You may need to restart Docker manually"
fi

# 6. Restart the containers
blue "🔄 Restarting all containers..."
cd /opt/latency-space
docker compose down
docker compose up -d
if [ $? -eq 0 ]; then
  green "✅ All containers restarted successfully"
else
  red "❌ Failed to restart containers"
  exit 1
fi

# 7. Final verification
blue "🔍 Verifying status.latency.space setup..."

# Wait for containers to start
sleep 5

# Check status container
if docker ps | grep -q status; then
  green "✅ Status container is running"
else
  red "❌ Status container is not running"
fi

# Check Nginx configuration
if grep -q "server_name status.latency.space" "$CONFIG_FILE"; then
  green "✅ Nginx configuration for status.latency.space is in place"
else
  red "❌ Nginx configuration is missing status.latency.space"
fi

# Check DNS resolution locally
RESOLVED_IP=$(dig +short status.latency.space)
SERVER_IP=$(curl -s ifconfig.me || curl -s icanhazip.com || curl -s ipecho.net/plain)

if [ -n "$RESOLVED_IP" ]; then
  if [[ "$RESOLVED_IP" == *"127.0.0.1"* ]] || [[ "$RESOLVED_IP" == *"localhost"* ]]; then
    red "❌ status.latency.space still resolves to localhost"
    yellow "⚠️ DNS changes may take time to propagate"
  else
    green "✅ status.latency.space resolves to: $RESOLVED_IP"
  fi
else
  yellow "⚠️ status.latency.space does not resolve to any IP yet"
  yellow "⚠️ DNS changes may take time to propagate"
fi

# Check HTTP connection
HTTP_RESPONSE=$(curl -s -I -m 5 http://status.latency.space | head -1)
if [ -n "$HTTP_RESPONSE" ]; then
  green "✅ HTTP response from status.latency.space: $HTTP_RESPONSE"
else
  red "❌ No HTTP response from status.latency.space"
fi

green "✅ All setup steps completed!"
yellow "⚠️ If status.latency.space is still not working correctly:"
echo "  1. DNS changes may take time to propagate (try: 'dig status.latency.space')"
echo "  2. You may need to restart the server if DNS changes aren't taking effect"
echo "  3. Check Docker logs for any issues: 'docker logs latency-space-status-1'"
echo "  4. Check Nginx error logs: 'tail -f /var/log/nginx/error.log'"