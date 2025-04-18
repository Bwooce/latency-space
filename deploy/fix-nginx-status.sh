#!/bin/bash
# Script to fix the Nginx configuration for status.latency.space

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }
yellow() { echo -e "\033[0;33m$1\033[0m"; }

blue "🛠️ Fixing Nginx configuration for status.latency.space"

# Set paths
NGINX_DIR="/etc/nginx"
SITES_AVAILABLE="$NGINX_DIR/sites-available"
CONFIG_FILE="$SITES_AVAILABLE/latency.space"

# First, check if the file exists
if [ ! -f "$CONFIG_FILE" ]; then
  red "❌ Configuration file not found at $CONFIG_FILE"
  exit 1
fi

# Create a backup of the existing configuration
cp "$CONFIG_FILE" "${CONFIG_FILE}.bak.$(date +%s)"
blue "💾 Backup created"

# Fix the upstream resolution issue by adding resolver directive
# This tells Nginx to use Docker's DNS resolver
blue "🔧 Adding DNS resolver for Docker service names..."

# Check if the resolver is already in the file
if grep -q "resolver" "$CONFIG_FILE"; then
  yellow "⚠️  Resolver directive already exists, updating..."
  sed -i 's/resolver [^;]*;/resolver 127.0.0.11 valid=30s;/' "$CONFIG_FILE"
else
  # Add resolver at the top of the file
  sed -i '1s/^/# Define resolver for Docker DNS\nresolver 127.0.0.11 valid=30s;\n\n/' "$CONFIG_FILE"
fi

# Now fix the status.latency.space server block
# We need to use a variable for proxy_pass when using DNS resolution
blue "🔧 Updating status.latency.space server block..."

# Find the status.latency.space server block and modify it
if grep -q "server_name status.latency.space" "$CONFIG_FILE"; then
  # Find the proxy_pass line within the status block and replace it
  # This is a bit complex because we need to match within a specific server block
  sed -i '/server_name status.latency.space/,/}/s|proxy_pass http://status:3000;|set $upstream_status http://status:3000;\n        proxy_pass $upstream_status;|' "$CONFIG_FILE"
  green "✅ Updated proxy_pass directive to use variable with DNS resolution"
else
  yellow "⚠️  No status.latency.space server block found, creating it..."
  
  # Append a new server block
  cat >> "$CONFIG_FILE" << 'EOF'

# Server for status dashboard
server {
    listen 80;
    server_name status.latency.space;
    
    # Global rate limiting
    limit_req zone=ip burst=10 nodelay;
    limit_conn addr 5;
    
    location / {
        set $upstream_status http://status:3000;
        proxy_pass $upstream_status;
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
  green "✅ Created new status.latency.space server block with proper DNS resolution"
fi

# Now make sure all proxy_pass directives use variables
blue "🔧 Updating other proxy_pass directives for consistency..."

# For the main domain
sed -i '/server_name latency.space www.latency.space/,/}/s|proxy_pass http://proxy:80;|set $upstream_proxy http://proxy:80;\n        proxy_pass $upstream_proxy;|' "$CONFIG_FILE"

# For the wildcard subdomains
sed -i '/server_name ~\^/,/}/s|proxy_pass http://proxy:80;|set $upstream_proxy http://proxy:80;\n        proxy_pass $upstream_proxy;|' "$CONFIG_FILE"

# Test the Nginx configuration
blue "🔍 Testing the updated Nginx configuration..."
nginx -t
if [ $? -eq 0 ]; then
  blue "🔄 Reloading Nginx..."
  systemctl reload nginx
  if [ $? -eq 0 ]; then
    green "✅ Nginx reloaded successfully"
  else
    red "❌ Failed to reload Nginx"
    exit 1
  fi
else
  red "❌ Nginx configuration test failed"
  # Show the backup file that can be restored
  yellow "⚠️  You can restore the previous configuration from the backup:"
  ls -la ${CONFIG_FILE}.bak.*
  exit 1
fi

# Verify that status.latency.space is accessible
blue "🔍 Checking if status.latency.space is accessible..."
sleep 2 # Give Nginx a moment to start

HTTP_RESPONSE=$(curl -s -I -m 5 http://status.latency.space | head -1)
if [ -n "$HTTP_RESPONSE" ]; then
  green "✅ status.latency.space is accessible: $HTTP_RESPONSE"
else
  yellow "⚠️  Could not access status.latency.space yet"
  yellow "⚠️  This might be due to DNS propagation delays or other issues"
  
  # Check if status container is running
  blue "🔍 Checking status container..."
  if docker ps | grep -q status; then
    green "✅ Status container is running"
    echo "Container details:"
    docker ps | grep status
  else
    red "❌ Status container is not running"
    yellow "⚠️  Start the status container with: docker compose up -d status"
  fi
  
  # Check Nginx status
  blue "🔍 Checking Nginx status..."
  systemctl status nginx
fi

green "✅ Nginx configuration updated. If you still have issues:"
echo "  1. Check Nginx error logs: tail -f /var/log/nginx/error.log"
echo "  2. Check status container logs: docker logs latency-space-status-1"
echo "  3. Try accessing with the server IP directly: curl http://localhost/status"
echo "  4. Ensure DNS is properly resolving: dig status.latency.space"