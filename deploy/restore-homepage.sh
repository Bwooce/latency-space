#!/bin/bash
# Script to restore the comprehensive static homepage for latency.space

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

blue "🌐 Restoring comprehensive latency.space homepage"
echo $DIVIDER

# Check if we're in the right directory
if [ ! -f "deploy/static/index.html" ]; then
  red "Static homepage file not found at deploy/static/index.html"
  echo "Please run this script from the latency-space directory"
  exit 1
fi

# Create directory for main site
mkdir -p /var/www/html/latency-space
chmod 755 /var/www/html/latency-space

# Copy the static homepage
blue "Copying static homepage to web root..."
cp deploy/static/index.html /var/www/html/latency-space/index.html
chmod 644 /var/www/html/latency-space/index.html

if [ -f "/var/www/html/latency-space/index.html" ]; then
  green "✅ Static homepage successfully copied to web root"
else
  red "❌ Failed to copy homepage"
  exit 1
fi

# Check if Nginx configuration has the correct root directive
NGINX_CONFIG="/etc/nginx/sites-enabled/latency.space"
if [ ! -f "$NGINX_CONFIG" ]; then
  yellow "⚠️ Nginx configuration not found at $NGINX_CONFIG"
  yellow "Using fix-nginx-clean.sh to set up Nginx configuration..."
  
  if [ -f "deploy/fix-nginx-clean.sh" ]; then
    bash deploy/fix-nginx-clean.sh
  else
    red "❌ Nginx fix script not found"
    exit 1
  fi
else
  # Check if the Nginx configuration is properly set up
  if grep -q "root /var/www/html/latency-space" "$NGINX_CONFIG"; then
    green "✅ Nginx configuration has correct root directive"
  else
    yellow "⚠️ Nginx configuration might not have the correct root directive"
    yellow "Using fix-nginx-clean.sh to ensure proper configuration..."
    
    if [ -f "deploy/fix-nginx-clean.sh" ]; then
      bash deploy/fix-nginx-clean.sh
    else
      red "❌ Nginx fix script not found"
      exit 1
    fi
  fi
  
  # Test and reload Nginx
  blue "Testing Nginx configuration..."
  if nginx -t; then
    green "✅ Nginx configuration is valid"
    blue "Reloading Nginx..."
    systemctl reload nginx
    green "✅ Nginx reloaded successfully"
  else
    red "❌ Nginx configuration test failed"
    exit 1
  fi
fi

# Final verification
blue "Verifying homepage access..."
if curl -s -I -m 5 http://localhost | grep -q "200 OK"; then
  green "✅ Homepage is accessible on localhost"
else
  yellow "⚠️ Homepage is not responding on localhost"
  echo "You may need to wait for Nginx to fully reload"
fi

# Summarize what was done
echo $DIVIDER
green "✓ Comprehensive homepage restoration complete!"
echo ""
echo "The static homepage contains:"
echo "- Complete list of all available celestial bodies"
echo "- Detailed usage instructions for both HTTP and SOCKS proxies"
echo "- Bandwidth and rate limit information"
echo "- Technical details about the latency simulation"
echo ""
echo "You can now access the homepage at:"
echo "  http://latency.space"
echo "  http://www.latency.space"
echo ""
echo "To verify the status service is working:"
echo "  http://status.latency.space"