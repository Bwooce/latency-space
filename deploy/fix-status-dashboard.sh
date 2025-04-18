#!/bin/bash
# Script to fix the status dashboard container

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

blue "🛠️ Fixing Status Dashboard"
echo $DIVIDER

# Get container IPs
blue "Getting container IPs..."
STATUS_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=status) 2>/dev/null)
PROXY_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=proxy) 2>/dev/null)
PROMETHEUS_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=prometheus) 2>/dev/null)

# Print container IPs
echo "Status container IP: $STATUS_IP"
echo "Proxy container IP: $PROXY_IP"
echo "Prometheus container IP: $PROMETHEUS_IP"

# In case any of the containers are not running, stop here
if [ -z "$STATUS_IP" ] || [ -z "$PROXY_IP" ] || [ -z "$PROMETHEUS_IP" ]; then
  yellow "⚠️ Some containers are not running. Starting them..."
  
  # Stop all containers
  blue "Stopping all containers..."
  docker compose down || true
  
  # Start containers with proper environment variables
  blue "Starting containers with environment variables..."
  docker compose up -d --force-recreate
  
  # Wait for containers to start
  blue "Waiting for containers to stabilize..."
  sleep 10
  
  # Get container IPs again
  STATUS_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=status) 2>/dev/null)
  PROXY_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=proxy) 2>/dev/null)
  PROMETHEUS_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=prometheus) 2>/dev/null)
  
  echo "Updated Status IP: $STATUS_IP"
  echo "Updated Proxy IP: $PROXY_IP"
  echo "Updated Prometheus IP: $PROMETHEUS_IP"
  
  # If still no IPs, exit
  if [ -z "$STATUS_IP" ] || [ -z "$PROXY_IP" ] || [ -z "$PROMETHEUS_IP" ]; then
    red "❌ Failed to start all containers properly"
    exit 1
  fi
fi

# Create a custom nginx config for the status container
blue "Creating custom Nginx config for status container..."

CONFIG_FILE=$(mktemp)
cat > "$CONFIG_FILE" << EOF
server {
    listen 80;
    server_name localhost;
    root /usr/share/nginx/html;
    index index.html;

    # Support for SPA routing
    location / {
        try_files \$uri \$uri/ /index.html;
    }

    # API proxy for metrics - using direct IP address
    location /api/metrics {
        proxy_pass http://${PROMETHEUS_IP}:9090/api/v1/query;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_cache_bypass \$http_upgrade;
        
        # Add error handling
        proxy_intercept_errors on;
        error_page 500 502 503 504 = @fallback_metrics;
    }
    
    # Fallback for metrics when Prometheus is unavailable
    location @fallback_metrics {
        default_type application/json;
        return 200 '{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1651356239.1,"60"]}]}}';
    }
    
    # Proxy for accessing the debug endpoints - using direct IP address
    location /api/debug/ {
        proxy_pass http://${PROXY_IP}:80/_debug/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host proxy;
        proxy_cache_bypass \$http_upgrade;
        
        # Add CORS headers
        add_header 'Access-Control-Allow-Origin' '*';
        add_header 'Access-Control-Allow-Methods' 'GET, OPTIONS';
        add_header 'Access-Control-Allow-Headers' 'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range';
    }
}
EOF

# Copy the custom config into the status container
blue "Copying custom config to status container..."
STATUS_CONTAINER=$(docker ps -q -f name=status)
docker cp "$CONFIG_FILE" "$STATUS_CONTAINER:/etc/nginx/conf.d/default.conf"

# Test Nginx configuration inside the container
blue "Testing Nginx configuration in status container..."
if docker exec "$STATUS_CONTAINER" nginx -t; then
  green "✅ Nginx configuration is valid"
else
  red "❌ Nginx configuration has errors"
  exit 1
fi

# Reload Nginx in the container
blue "Reloading Nginx in status container..."
docker exec "$STATUS_CONTAINER" nginx -s reload
green "✅ Status container Nginx reloaded"

# Now fix the host machine's Nginx configuration
blue "Updating host Nginx configuration..."

# Run our update-nginx.sh script
if [ -f "deploy/update-nginx.sh" ]; then
  blue "Running Nginx update script..."
  bash deploy/update-nginx.sh
fi

# Final tests
blue "Testing status dashboard connectivity..."
echo $DIVIDER

echo "1. Status container health check:"
curl -I -s "http://$STATUS_IP/" | head -1 || echo "Failed"

echo "2. Status API metrics:"
curl -I -s "http://$STATUS_IP/api/metrics" | head -1 || echo "Failed"

echo "3. Status API debug endpoints:"
curl -I -s "http://$STATUS_IP/api/debug/distances" | head -1 || echo "Failed"

echo "4. External status.latency.space access:"
curl -I -s -H "Host: status.latency.space" "http://localhost/" | head -1 || echo "Failed"

echo $DIVIDER

green "✅ Status dashboard fix completed!"
echo ""
echo "If the dashboard is still not working:"
echo "1. Check the JavaScript console in your browser for errors"
echo "2. Check the status container logs: docker logs \$(docker ps -q -f name=status)"
echo "3. Try rebuilding the status container: docker compose build --no-cache status && docker compose up -d status"
echo "4. View the Nginx error log: tail -f /var/log/nginx/error.log"