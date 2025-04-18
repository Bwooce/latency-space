#!/bin/bash
# Script to fix Nginx configuration with correct Docker DNS resolver

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }

blue "🛠️ Fixing Nginx configuration for Docker service resolution"

# Backup the current configuration
CONFIG_FILE="/etc/nginx/sites-available/latency.space"
BACKUP_FILE="${CONFIG_FILE}.bak.$(date +%s)"

if [ -f "$CONFIG_FILE" ]; then
  cp "$CONFIG_FILE" "$BACKUP_FILE"
  blue "💾 Backup created at $BACKUP_FILE"
else
  blue "No existing configuration found, will create a new one"
fi

# Copy the fixed configuration
blue "📝 Installing fixed configuration..."

# Use fixed configuration from this script
cat > "$CONFIG_FILE" << 'EOF'
# Define rate limiting zones
limit_req_zone $binary_remote_addr zone=ip:10m rate=5r/m;
limit_conn_zone $binary_remote_addr zone=addr:10m;

# Define resolver for Docker DNS
resolver 127.0.0.11 valid=30s;

# Server for base domain latency.space
server {
    listen 80;
    server_name latency.space www.latency.space;
    
    # Handle Let's Encrypt validation challenges
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }
    
    # Allow direct access to diagnostic pages
    location ~ ^/(diagnostic\.html|status\.html)$ {
        root /var/www/html;
        try_files $uri =404;
    }
    
    # Less strict rate limiting for main site
    limit_req zone=ip burst=20 nodelay;
    limit_conn addr 10;
    
    # Serve the main site content
    root /var/www/html/latency-space;
    index index.html;
    
    # Try to serve static files, fallback to index.html for SPA
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    # Return 444 (no response) for suspicious requests
    location ~ \.(php|aspx|asp|cgi|jsp)$ {
        return 444;
    }
}

# Server for all other .latency.space subdomains
server {
    listen 80;
    server_name ~^[^.]+\.latency\.space$ ~^[^.]+\.[^.]+\.latency\.space$ ~^[^.]+\.[^.]+\.[^.]+\.latency\.space$;
    
    # Handle Let's Encrypt validation challenges
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }
    
    # Global rate limiting - very strict
    limit_req zone=ip burst=10 nodelay;
    limit_conn addr 5;
    
    # For all subdomains, serve over HTTP directly
    location / {
        # Only allow GET requests to prevent abuse
        limit_except GET {
            deny all;
        }
        
        set $upstream_proxy http://proxy:80;
        proxy_pass $upstream_proxy;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Destination $host;
        proxy_cache_bypass $http_upgrade;
        
        # Set timeouts to prevent hanging connections
        proxy_connect_timeout 10s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
    
    # Return 444 (no response) for suspicious requests
    location ~ \.(php|aspx|asp|cgi|jsp)$ {
        return 444;
    }
}

# Server for status dashboard
server {
    listen 80;
    server_name status.latency.space;
    
    # Global rate limiting - very strict
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

# Ensure symlink exists
if [ ! -f "/etc/nginx/sites-enabled/latency.space" ]; then
  ln -sf "$CONFIG_FILE" "/etc/nginx/sites-enabled/latency.space"
  blue "🔗 Created symlink in sites-enabled"
fi

# Test and reload Nginx
blue "🔍 Testing Nginx configuration..."
if nginx -t; then
  green "✅ Configuration is valid"
  blue "🔄 Reloading Nginx..."
  systemctl reload nginx
  green "✅ Nginx reloaded successfully"
else
  red "❌ Configuration test failed"
  red "Restoring backup..."
  cp "$BACKUP_FILE" "$CONFIG_FILE"
  nginx -t && systemctl reload nginx
  exit 1
fi

# Verify status.latency.space configuration
blue "🔍 Verifying status.latency.space configuration..."
if grep -q "server_name status.latency.space" "$CONFIG_FILE"; then
  green "✅ status.latency.space server block is properly configured"
else
  red "❌ status.latency.space configuration not found in the file"
  exit 1
fi

# Test status container
blue "🔍 Checking if status container is running..."
if docker ps | grep -q "status"; then
  green "✅ Status container is running"
else
  blue "Starting status container..."
  docker compose up -d status
  
  if docker ps | grep -q "status"; then
    green "✅ Status container started successfully"
  else
    red "❌ Failed to start status container"
    exit 1
  fi
fi

green "✅ Nginx configuration has been fixed!"
blue "🔍 Next steps:"
echo "  1. Try accessing status.latency.space in a browser"
echo "  2. If it still doesn't work, check DNS resolution:"
echo "     dig status.latency.space"
echo "  3. Check container logs:"
echo "     docker logs \$(docker ps | grep status | awk '{print \$1}')"
echo ""
echo "If you need to restore the previous configuration:"
echo "cp $BACKUP_FILE $CONFIG_FILE && systemctl reload nginx"