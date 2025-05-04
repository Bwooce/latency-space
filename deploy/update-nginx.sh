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

# Get container IPs - using more specific network inspection
blue "Getting container IPs..."
STATUS_CONTAINER_ID=$(docker ps -q -f name=status)
PROXY_CONTAINER_ID=$(docker ps -q -f name=proxy)
PROMETHEUS_CONTAINER_ID=$(docker ps -q -f name=prometheus)
GRAFANA_CONTAINER_ID=$(docker ps -q -f name=grafana)

# Get IPs from all networks, prioritizing space-net
get_container_ip() {
  local container_id=$1
  local network_name="latency-space_space-net"
  
  # First try the main network
  local ip=$(docker inspect -f "{{range \$k, \$v := .NetworkSettings.Networks}}{{if eq \$k \"$network_name\"}}{{\$v.IPAddress}}{{end}}{{end}}" $container_id 2>/dev/null)
  
  # If not found, try any network
  if [ -z "$ip" ]; then
    ip=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $container_id 2>/dev/null)
  fi
  
  echo $ip
}

STATUS_IP=$(get_container_ip $STATUS_CONTAINER_ID)
PROXY_IP=$(get_container_ip $PROXY_CONTAINER_ID)
PROMETHEUS_IP=$(get_container_ip $PROMETHEUS_CONTAINER_ID)
GRAFANA_IP=$(get_container_ip $GRAFANA_CONTAINER_ID)

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

# Server for base domain latency.space
server {
    listen 80;
    server_name latency.space www.latency.space;
    
    # Handle Let's Encrypt validation challenges by proxying to the backend Go app
    location /.well-known/acme-challenge/ {
        proxy_pass http://$PROXY_IP:8080; # Target the Go app's internal HTTP port (Note: Using $PROXY_IP variable from script)
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
    
    # Allow direct access to diagnostic pages
    location ~ ^/(diagnostic\.html|status\.html)$ {
        root /var/www/html;
        try_files \$uri =404;
    }
    
    # Less strict rate limiting for main site
    limit_req zone=ip burst=20 nodelay;
    limit_conn addr 10;

    # API requests - Proxy to backend Go service
    location /api/ {
        # Using the PROXY_IP variable defined in the script
        proxy_pass http://$PROXY_IP:80; 
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Forwarded-Host \$host;
        proxy_set_header X-Forwarded-For \$remote_addr;
        proxy_cache_bypass \$http_upgrade;

        # Set reasonable timeouts for API calls
        proxy_connect_timeout 10s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }

    # API status data requests - Proxy to backend Go service
    location /api/status-data {
        # Using the PROXY_IP variable defined in the script
        proxy_pass http://$PROXY_IP:80;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Forwarded-Host \$host; # Keep consistency
        proxy_set_header X-Real-IP \$remote_addr; # Use X-Real-IP like other API/status blocks
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for; # Use combined forwarded-for
        proxy_set_header X-Forwarded-Proto \$scheme; # Include scheme
        proxy_cache_bypass \$http_upgrade;

        # Use timeouts similar to other API calls
        proxy_connect_timeout 10s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }

    # Debug endpoints with higher priority (merged from separate server block)
    location = /_debug/metrics {
        proxy_pass http://$PROXY_IP:80;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }

    location = /_debug/distances {
        proxy_pass http://$PROXY_IP:80;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }

    location = /_debug/status {
        proxy_pass http://$PROXY_IP:80;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }

    # Add a generic /_debug/ catch-all pointing to PROXY_IP
    location ^~ /_debug/ {
         proxy_pass http://${PROXY_IP}:80;
         proxy_http_version 1.1;
         proxy_set_header Host \$host;
         proxy_set_header X-Real-IP \$remote_addr;
         proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }

    # Proxy root to the status service
    location / {
        # Explicitly exclude debug paths if they are handled by other servers/locations
        # if (\$uri ~* "^/_debug") {
        #     return 404; # Or handle appropriately
        # }

        # Proxy to the status service (using STATUS_IP variable)
        proxy_pass http://$STATUS_IP:80;

        # Standard proxy headers (copied from status.latency.space block)
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;

        # Set timeouts (copied from status.latency.space block)
        proxy_connect_timeout 30s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Return 444 (no response) for suspicious requests
    location ~ \.(php|aspx|asp|cgi|jsp)$ {
        return 444;
    }
}

# Server block for HTTP -> HTTPS redirect for subdomains
server {
    listen 80;
    listen [::]:80;
    server_name ~^[^.]+\.latency\.space$ ~^[^.]+\.[^.]+\.latency\.space$ ~^[^.]+\.[^.]+\.[^.]+\.latency\.space$;

    # Handle Let's Encrypt validation challenges by proxying to the backend Go app
    location /.well-known/acme-challenge/ {
        proxy_pass http://$PROXY_IP:8080; # Target the Go app's internal HTTP port (Note: Using $PROXY_IP variable from script)
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    # Redirect all other HTTP requests to HTTPS
    location / {
        return 301 https://\$host\$request_uri;
    }
}


# Server for all other .latency.space subdomains
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name ~^[^.]+\.latency\.space$ ~^[^.]+\.[^.]+\.latency\.space$ ~^[^.]+\.[^.]+\.[^.]+\.latency\.space$;

    ssl_certificate /etc/letsencrypt/live/latency.space/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/latency.space/privkey.pem;
    # Include recommended settings from certbot/nginx guide (or similar standard practice)
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;
    ssl_session_tickets off;

    # Handle Let's Encrypt validation challenges by proxying to the backend Go app
    location /.well-known/acme-challenge/ {
        proxy_pass http://$PROXY_IP:8080; # Target the Go app's internal HTTP port (Note: Using $PROXY_IP variable from script)
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
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
        proxy_pass http://$PROXY_IP:80;
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
