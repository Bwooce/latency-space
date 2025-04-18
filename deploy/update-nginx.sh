#!/bin/bash
# Script to update Nginx configuration with container IPs

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

blue "🔧 Updating Nginx Configuration"
echo $DIVIDER

# Get container IPs
blue "Getting container IPs..."
STATUS_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=status) 2>/dev/null)
PROXY_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=proxy) 2>/dev/null)
PROMETHEUS_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=prometheus) 2>/dev/null)
GRAFANA_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=grafana) 2>/dev/null)

# Print container IPs
echo "Status container IP: $STATUS_IP"
echo "Proxy container IP: $PROXY_IP"
echo "Prometheus container IP: $PROMETHEUS_IP"
echo "Grafana container IP: $GRAFANA_IP"

if [ -z "$STATUS_IP" ] || [ -z "$PROXY_IP" ]; then
  red "❌ Could not determine container IPs"
  echo "Make sure the containers are running. Try: docker compose up -d"
  exit 1
fi

# Create Nginx configuration with correct IPs
NGINX_CONF_DIR="/etc/nginx/sites-available"
NGINX_ENABLED_DIR="/etc/nginx/sites-enabled"
NGINX_CONF="$NGINX_CONF_DIR/latency.space"

# Make sure the directories exist
mkdir -p $NGINX_CONF_DIR
mkdir -p $NGINX_ENABLED_DIR

# Backup existing config if it exists
if [ -f "$NGINX_CONF" ]; then
  cp "$NGINX_CONF" "$NGINX_CONF.backup.$(date +%s)"
fi

blue "Creating Nginx configuration..."
cat > "$NGINX_CONF" << EOF
# Define rate limiting zones
limit_req_zone \$binary_remote_addr zone=ip:10m rate=5r/m;
limit_conn_zone \$binary_remote_addr zone=addr:10m;

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
        try_files \$uri =404;
    }
    
    # Less strict rate limiting for main site
    limit_req zone=ip burst=20 nodelay;
    limit_conn addr 10;
    
    # Serve the main site content
    root /opt/latency-space/static;
    index index.html;
    
    # Try to serve static files, fallback to index.html for SPA
    location / {
        try_files \$uri \$uri/ /index.html;
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
        
        # Docker service resolution - using direct IP instead of DNS
        proxy_pass http://${PROXY_IP}:80;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Forwarded-Host \$host;
        proxy_set_header X-Forwarded-For \$remote_addr;
        proxy_set_header X-Destination \$host;
        proxy_cache_bypass \$http_upgrade;
        
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
        # Docker service resolution - using direct IP instead of DNS
        proxy_pass http://${STATUS_IP}:80;
        
        # Standard proxy headers
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
        
        # Increase timeouts for dashboard operations
        proxy_connect_timeout 30s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}

# Server for _debug endpoints
server {
    listen 80;
    server_name latency.space;
    
    # Debug endpoints with higher priority
    location = /_debug/metrics {
        proxy_pass http://${PROXY_IP}:80;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }
    
    location = /_debug/distances {
        proxy_pass http://${PROXY_IP}:80;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }
    
    location = /_debug/status {
        proxy_pass http://${PROXY_IP}:80;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }
}
EOF

# Create a symlink to enable the site
ln -sf "$NGINX_CONF" "$NGINX_ENABLED_DIR/latency.space"

# Test and reload Nginx
blue "Testing Nginx configuration..."
nginx -t
if [ $? -ne 0 ]; then
  red "❌ Nginx configuration test failed"
  if [ -f "$NGINX_CONF.backup" ]; then
    yellow "Restoring previous configuration..."
    cp "$NGINX_CONF.backup" "$NGINX_CONF"
    nginx -t && systemctl reload nginx
  fi
  exit 1
else
  green "✅ Nginx configuration test passed"
  blue "Reloading Nginx..."
  systemctl reload nginx
  green "✅ Nginx reloaded successfully"
fi

# Skip adding host entries to containers since they lack permissions
blue "Skipping host entries for containers (permission issues)..."
yellow "⚠️ Cannot modify /etc/hosts in containers - using direct IPs in Nginx config instead"

# Add entries to host machine's /etc/hosts file instead
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

# Test connectivity
blue "Testing connectivity..."
echo $DIVIDER

echo "1. Testing status container directly:"
curl -I -s http://$STATUS_IP | head -1 || echo "Failed"

echo "2. Testing status.latency.space domain:"
curl -I -s -H "Host: status.latency.space" http://localhost | head -1 || echo "Failed"

echo "3. Testing proxy container directly:"
curl -I -s http://$PROXY_IP | head -1 || echo "Failed"

echo "4. Testing _debug/metrics endpoint:"
curl -I -s -H "Host: latency.space" http://localhost/_debug/metrics | head -1 || echo "Failed"

echo $DIVIDER

green "✅ Nginx configuration updated successfully!"
echo ""
echo "You should now be able to access:"
echo "- http://latency.space - Main site"
echo "- http://status.latency.space - Status dashboard"
echo "- http://mars.latency.space - Mars proxy"
echo "- http://latency.space/_debug/metrics - Debug metrics"