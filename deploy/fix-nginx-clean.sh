#!/bin/bash
# Complete Nginx configuration cleanup and reinstallation script

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }
yellow() { echo -e "\033[0;33m$1\033[0m"; }

blue "🧹 Starting COMPLETE Nginx configuration cleanup and reinstallation"

# Backup directory
BACKUP_DIR="/tmp/nginx-backup-$(date +%s)"
mkdir -p "$BACKUP_DIR"
blue "💾 Created backup directory at $BACKUP_DIR"

# Backup all existing Nginx configurations
blue "💾 Backing up all existing Nginx configurations..."
cp -r /etc/nginx/sites-available "$BACKUP_DIR/" 2>/dev/null || true
cp -r /etc/nginx/sites-enabled "$BACKUP_DIR/" 2>/dev/null || true
cp -r /etc/nginx/conf.d "$BACKUP_DIR/" 2>/dev/null || true
green "✅ Backup completed"

# Complete cleanup of all site configurations
blue "🧹 Cleaning up all site configurations..."
# Disable all existing sites
rm -f /etc/nginx/sites-enabled/* 2>/dev/null || true
# Remove all existing site configurations
rm -f /etc/nginx/sites-available/* 2>/dev/null || true
# Ensure directories exist
mkdir -p /etc/nginx/sites-available /etc/nginx/sites-enabled
green "✅ All previous configurations removed"

# Create fresh configuration file
CONFIG_FILE="/etc/nginx/sites-available/latency.space"
blue "📝 Creating fresh configuration file at $CONFIG_FILE..."

# Write configuration file
cat > "$CONFIG_FILE" << 'EOF'
# Define rate limiting zones
limit_req_zone $binary_remote_addr zone=ip:10m rate=5r/m;
limit_conn_zone $binary_remote_addr zone=addr:10m;

# Define resolver for Docker DNS - critical for resolution of service names
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
        
        # Docker service resolution with variable for DNS resolution
        set $proxy_upstream http://proxy:80;
        proxy_pass $proxy_upstream;
        
        # Standard proxy headers
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
        # Docker service resolution with variable for DNS resolution
        set $status_upstream http://status:3000;
        proxy_pass $status_upstream;
        
        # Standard proxy headers
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

# Create symlink
blue "🔗 Creating symlink for the configuration..."
ln -sf "$CONFIG_FILE" "/etc/nginx/sites-enabled/latency.space"
green "✅ Symlink created"

# Test and reload Nginx
blue "🔍 Testing Nginx configuration..."
if nginx -t; then
  green "✅ Configuration is valid"
  blue "🔄 Reloading Nginx..."
  systemctl reload nginx
  green "✅ Nginx reloaded successfully"
else
  red "❌ Configuration test failed"
  yellow "⚠️ Restoring from backup is not possible since we did a complete cleanup"
  yellow "⚠️ You can find your original configurations in $BACKUP_DIR"
  exit 1
fi

# Prepare directories
blue "📁 Ensuring web directories exist..."
mkdir -p /var/www/html/latency-space
chmod 755 /var/www/html/latency-space
mkdir -p /var/www/html/.well-known/acme-challenge
chmod 755 /var/www/html/.well-known

# Create an index file if it doesn't exist
if [ ! -f "/var/www/html/latency-space/index.html" ]; then
  blue "📝 Creating simple index file..."
  cat > /var/www/html/latency-space/index.html << 'EOF'
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
  chmod 644 /var/www/html/latency-space/index.html
  green "✅ Created index file"
fi

# Test status container
blue "🔍 Checking if status container is running..."
if docker ps | grep -q "status"; then
  green "✅ Status container is running"
else
  blue "Starting status container..."
  cd /opt/latency-space
  docker compose up -d status
  
  if docker ps | grep -q "status"; then
    green "✅ Status container started successfully"
  else
    red "❌ Failed to start status container"
    exit 1
  fi
fi

# Test DNS resolution
blue "🔍 Testing DNS resolution for status.latency.space..."
if host status.latency.space 127.0.0.11 &>/dev/null || dig +short status.latency.space; then
  green "✅ DNS resolution working for status.latency.space"
else
  yellow "⚠️ DNS resolution for status.latency.space is not working yet"
  yellow "⚠️ This might be normal if DNS changes haven't propagated yet"
fi

green "✅ Nginx configuration has been completely reset and fixed!"
blue "🔍 Next steps:"
echo "  1. Try accessing status.latency.space in a browser"
echo "  2. If it still doesn't work, check DNS resolution:"
echo "     dig status.latency.space"
echo "  3. Try directly through the IP: curl -H 'Host: status.latency.space' http://$(hostname -I | awk '{print $1}')"
echo "  4. Check container logs:"
echo "     docker logs \$(docker ps | grep status | awk '{print \$1}')"
echo ""
echo "Your original Nginx configurations were backed up to: $BACKUP_DIR"