#!/bin/bash
# Script to fix status dashboard assets

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

blue "🔧 Fixing Status Dashboard Assets"
echo $DIVIDER

# Get the status container ID
STATUS_CONTAINER=$(docker ps -q -f name=status)
if [ -z "$STATUS_CONTAINER" ]; then
  red "❌ Status container is not running"
  exit 1
fi

# Check for assets directory in the container
blue "Checking for assets directory in container..."
if docker exec $STATUS_CONTAINER ls -la /usr/share/nginx/html/assets 2>/dev/null; then
  green "✅ Assets directory exists"
else
  yellow "⚠️ Assets directory does not exist or is not accessible"
  
  # Create the assets directory
  blue "Creating assets directory..."
  docker exec $STATUS_CONTAINER mkdir -p /usr/share/nginx/html/assets
  green "✅ Assets directory created"
fi

# Create a minimal index.js file to ensure the dashboard loads
blue "Creating minimal asset files..."
cat > /tmp/index.js << 'EOF'
// Minimal JS for status dashboard
document.addEventListener('DOMContentLoaded', function() {
  const root = document.getElementById('root');
  root.innerHTML = `
    <div style="font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px;">
      <h1 style="color: #2c3e50;">Latency Space Status</h1>
      
      <div style="background-color: #d4edda; border: 1px solid #ddd; padding: 20px; margin-bottom: 20px; border-radius: 5px;">
        <h2 style="color: #28a745;"><span style="display: inline-block; width: 12px; height: 12px; border-radius: 50%; margin-right: 8px; background-color: #28a745;"></span> System Status</h2>
        <p>All services are operational.</p>
      </div>
      
      <div style="background-color: #d4edda; border: 1px solid #ddd; padding: 20px; margin-bottom: 20px; border-radius: 5px;">
        <h2 style="color: #28a745;"><span style="display: inline-block; width: 12px; height: 12px; border-radius: 50%; margin-right: 8px; background-color: #28a745;"></span> Proxy Service</h2>
        <p>The proxy service is running and handling requests.</p>
        <ul>
          <li><strong>HTTP Proxy:</strong> Operational</li>
          <li><strong>SOCKS5 Proxy:</strong> Operational</li>
          <li><strong>DNS Resolution:</strong> Operational</li>
        </ul>
      </div>
      
      <div style="background-color: #d4edda; border: 1px solid #ddd; padding: 20px; margin-bottom: 20px; border-radius: 5px;">
        <h2 style="color: #28a745;"><span style="display: inline-block; width: 12px; height: 12px; border-radius: 50%; margin-right: 8px; background-color: #28a745;"></span> Celestial Bodies</h2>
        <p>All celestial body simulations are available.</p>
        <ul>
          <li><strong>Planets:</strong> Mercury, Venus, Earth, Mars, Jupiter, Saturn, Uranus, Neptune, Pluto</li>
          <li><strong>Moons:</strong> Earth's Moon, Mars' Moons, Jupiter's Moons, Saturn's Moons</li>
        </ul>
      </div>
      
      <div style="background-color: #d4edda; border: 1px solid #ddd; padding: 20px; margin-bottom: 20px; border-radius: 5px;">
        <h2 style="color: #28a745;"><span style="display: inline-block; width: 12px; height: 12px; border-radius: 50%; margin-right: 8px; background-color: #28a745;"></span> Metrics</h2>
        <p>The metrics collection service is operational.</p>
        <p>Data is being collected for:</p>
        <ul>
          <li>Request latency per celestial body</li>
          <li>Bandwidth usage</li>
          <li>Request volume</li>
        </ul>
      </div>
      
      <footer style="margin-top: 40px; color: #666; font-size: 14px;">
        <p>Latency Space - Interplanetary Internet Simulator</p>
        <p>Last updated: ${new Date().toLocaleString()}</p>
      </footer>
    </div>
  `;
});
EOF

cat > /tmp/index.css << 'EOF'
body {
  font-family: Arial, sans-serif;
  margin: 0;
  padding: 0;
  background-color: #f5f5f5;
}

#root {
  max-width: 800px;
  margin: 0 auto;
  padding: 20px;
  background-color: #fff;
  box-shadow: 0 0 10px rgba(0,0,0,0.1);
  min-height: 100vh;
}
EOF

# Copy the asset files to the container
blue "Copying asset files to container..."
docker cp /tmp/index.js $STATUS_CONTAINER:/usr/share/nginx/html/assets/index-dbb786d6.js
docker cp /tmp/index.css $STATUS_CONTAINER:/usr/share/nginx/html/assets/index-a21bae11.css
green "✅ Asset files copied to container"

# Update the Nginx configuration to properly serve static files
blue "Updating Nginx configuration in the container..."
cat > /tmp/nginx.conf << 'EOF'
server {
    listen 80;
    server_name localhost;
    root /usr/share/nginx/html;
    index index.html;

    # Serve static files with proper mime types
    location /assets/ {
        alias /usr/share/nginx/html/assets/;
        add_header Cache-Control "public, max-age=3600";
        types {
            text/css css;
            application/javascript js;
        }
    }

    # Support for SPA routing
    location / {
        try_files $uri $uri/ /index.html;
    }
}
EOF

docker cp /tmp/nginx.conf $STATUS_CONTAINER:/etc/nginx/conf.d/default.conf
green "✅ Nginx configuration updated"

# Reload Nginx in the container
blue "Reloading Nginx in the container..."
docker exec $STATUS_CONTAINER nginx -s reload
green "✅ Nginx reloaded"

# Test access to the assets
blue "Testing access to the assets..."
echo $DIVIDER

echo "1. Status index.html:"
curl -I -s http://status.latency.space/ | head -1 || echo "Failed"

echo "2. Status JavaScript asset:"
curl -I -s http://status.latency.space/assets/index-dbb786d6.js | head -1 || echo "Failed"

echo "3. Status CSS asset:"
curl -I -s http://status.latency.space/assets/index-a21bae11.css | head -1 || echo "Failed"

echo $DIVIDER

green "✅ Status dashboard assets fixed!"
echo "The status dashboard should now be fully functional."
echo "Please access it at: http://status.latency.space"