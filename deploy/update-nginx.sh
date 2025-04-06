#!/bin/bash
# Quick script to update Nginx configuration

set -e

echo "🔄 Updating Nginx configuration..."

# Check if the nginx-proxy.conf file exists
if [ ! -f "deploy/nginx-proxy.conf" ]; then
  echo "❌ Configuration file not found"
  exit 1
fi

# Copy the configuration to Nginx
cp deploy/nginx-proxy.conf /etc/nginx/sites-available/latency.space

# Create symlink if it doesn't exist
if [ ! -f "/etc/nginx/sites-enabled/latency.space" ]; then
  ln -sf /etc/nginx/sites-available/latency.space /etc/nginx/sites-enabled/
fi

# Remove default if it exists
if [ -f "/etc/nginx/sites-enabled/default" ]; then
  rm -f /etc/nginx/sites-enabled/default
fi

# Test Nginx configuration
echo "🔍 Testing Nginx configuration..."
if nginx -t; then
  echo "✅ Configuration is valid"
  echo "🔄 Reloading Nginx..."
  systemctl reload nginx
  echo "✅ Nginx reloaded successfully"
else
  echo "❌ Configuration is invalid"
  exit 1
fi

# Create test directory for Let's Encrypt validation
mkdir -p /var/www/html/.well-known/acme-challenge

echo "🌐 Nginx is now configured to proxy requests to:"
echo "  - All subdomains on HTTP (port 80) → http://localhost:8080"
echo "  - All single-level subdomains on HTTPS (port 443) → http://localhost:8080"
echo "  - status.latency.space → http://localhost:3000"
echo ""
echo "You should now be able to access moon.earth.latency.space successfully."