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

# Create directory for main site and ensure permissions
mkdir -p /var/www/html/latency-space
chmod 755 /var/www/html/latency-space

# Create a comprehensive landing page
echo "Creating landing page at /var/www/html/latency-space/index.html"

# Copy the comprehensive static index.html if available
if [ -f "deploy/static/index.html" ]; then
  cp deploy/static/index.html /var/www/html/latency-space/index.html
  echo "✅ Copied comprehensive index.html to web root"
else
  # Fall back to creating a simple landing page
  cat > /var/www/html/latency-space/index.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Latency Space</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
        }
        h1 {
            color: #333;
        }
        .celestial-link {
            display: inline-block;
            margin: 10px;
            padding: 10px;
            background-color: #f0f0f0;
            border-radius: 5px;
        }
    </style>
</head>
<body>
    <h1>Welcome to Latency Space</h1>
    <p>Experience the latency of interplanetary communication!</p>
    
    <h2>Available Celestial Bodies:</h2>
    
    <div class="celestial-link">
        <a href="http://mercury.latency.space">Mercury</a>
    </div>
    <div class="celestial-link">
        <a href="http://venus.latency.space">Venus</a>
    </div>
    <div class="celestial-link">
        <a href="http://earth.latency.space">Earth</a>
    </div>
    <div class="celestial-link">
        <a href="http://mars.latency.space">Mars</a>
    </div>
    <div class="celestial-link">
        <a href="http://jupiter.latency.space">Jupiter</a>
    </div>
    <div class="celestial-link">
        <a href="http://saturn.latency.space">Saturn</a>
    </div>
    <div class="celestial-link">
        <a href="http://uranus.latency.space">Uranus</a>
    </div>
    <div class="celestial-link">
        <a href="http://neptune.latency.space">Neptune</a>
    </div>
    <div class="celestial-link">
        <a href="http://pluto.latency.space">Pluto</a>
    </div>
    
    <h2>Status Dashboard:</h2>
    <p>Check the <a href="http://status.latency.space">Status Dashboard</a> for real-time distances and latency information.</p>
    
    <hr>
    <p><small>Latency Space - Simulating interplanetary communication delays</small></p>
</body>
</html>
EOF
fi

# Set proper permissions on the HTML file
chmod 644 /var/www/html/latency-space/index.html
echo "Landing page created successfully"

# Create directory for Let's Encrypt validation
mkdir -p /var/www/html/.well-known/acme-challenge

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

echo "🌐 Nginx is now configured to:"
echo "  - Serve the main website at latency.space and www.latency.space"
echo "  - Proxy all celestial-body subdomains to http://proxy:80"
echo "  - Proxy status.latency.space to http://status:3000"
echo ""
echo "You should now be able to access the main website and all subdomain proxies."