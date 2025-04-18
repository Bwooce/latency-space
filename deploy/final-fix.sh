#!/bin/bash
# Final fix script to resolve remaining issues

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }

# Create writable directory for our files
WRITABLE_DIR="/tmp/latency-space"
mkdir -p $WRITABLE_DIR/html

# Create an index.html file for the main site
cat > $WRITABLE_DIR/html/index.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Latency Space</title>
    <style>
        body { font-family: sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1 { color: #333; }
        .celestial-link { margin: 10px 0; padding: 10px; background: #f5f5f5; border-radius: 5px; }
        a { color: #0066cc; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <h1>Latency Space - Interplanetary Internet Simulator</h1>
    <p>This project simulates the latency of communication across the solar system.</p>
    
    <h2>Available Celestial Bodies:</h2>
    <div class="celestial-link">
        <a href="http://mercury.latency.space">Mercury</a>
    </div>
    <div class="celestial-link">
        <a href="http://venus.latency.space">Venus</a>
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

# Create simplified Nginx config with working static files
blue "Creating final fixed Nginx configuration..."
mkdir -p $WRITABLE_DIR/nginx

cat > $WRITABLE_DIR/nginx/latency.conf << 'EOC'
# Define rate limiting zones
limit_req_zone $binary_remote_addr zone=ip:10m rate=5r/m;
limit_conn_zone $binary_remote_addr zone=addr:10m;

# Server for base domain latency.space
server {
    listen 80;
    server_name latency.space www.latency.space;
    
    # Important: Add DNS resolver for Docker container names
    resolver 127.0.0.11 valid=30s;
    
    # Static content
    location = / {
        root /tmp/latency-space/html;
        index index.html;
    }
    
    location = /index.html {
        root /tmp/latency-space/html;
    }
    
    location = /test.html {
        root /tmp/latency-space/html;
    }
    
    # Proxy to proxy container
    location / {
        set $upstream_proxy http://proxy;
        proxy_pass $upstream_proxy:80;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}

# Server for status.latency.space
server {
    listen 80;
    server_name status.latency.space;
    
    # Important: Add DNS resolver for Docker container names
    resolver 127.0.0.11 valid=30s;
    
    location / {
        set $upstream_status http://status;
        proxy_pass $upstream_status:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}

# Server for all other .latency.space subdomains
server {
    listen 80;
    server_name ~^[^.]+\.latency\.space$;
    
    # Important: Add DNS resolver for Docker container names
    resolver 127.0.0.11 valid=30s;
    
    location / {
        set $upstream_proxy http://proxy;
        proxy_pass $upstream_proxy:80;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
EOC

# Create a test HTML file
blue "Creating test HTML file..."
cat > $WRITABLE_DIR/html/test.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Latency Space Test</title>
</head>
<body>
    <h1>Nginx Configuration Test</h1>
    <p>If you can see this page, Nginx is correctly serving static files from the writable directory.</p>
    <p>Generated at: $(date)</p>
    <p>Hostname: $(hostname)</p>
    
    <h2>Next Steps:</h2>
    <ul>
        <li>Try accessing <a href="/">the main site</a> to see if static file serving works</li>
        <li>Try accessing <a href="http://status.latency.space">status.latency.space</a> to check subdomain resolution</li>
        <li>Try accessing <a href="http://mars.latency.space">mars.latency.space</a> to check celestial body proxying</li>
    </ul>
</body>
</html>
EOF

# Update Nginx configuration
blue "Applying Nginx configuration..."
if [ -w "/etc/nginx/sites-available/latency.space" ]; then
    cat $WRITABLE_DIR/nginx/latency.conf > /etc/nginx/sites-available/latency.space
    
    # Test and reload
    blue "Testing Nginx configuration..."
    nginx -t && {
        systemctl reload nginx
        green "✓ Nginx configuration updated successfully!"
    } || {
        red "✗ Nginx configuration test failed."
    }
else
    # Alternative: Modify nginx.conf directly
    NGINX_CONF="/etc/nginx/nginx.conf"
    if [ -w "$NGINX_CONF" ]; then
        # Check if we already added the include
        if ! grep -q "/tmp/latency-space/nginx/\*.conf" $NGINX_CONF; then
            # Make a backup
            cp $NGINX_CONF ${NGINX_CONF}.bak
            
            # Include our config directory in http section
            sed -i '/http {/a \    include /tmp/latency-space/nginx/*.conf;' $NGINX_CONF
        fi
        
        blue "Testing Nginx configuration..."
        nginx -t && {
            systemctl reload nginx
            green "✓ Nginx configuration updated successfully!"
        } || {
            red "✗ Nginx configuration test failed."
        }
    else
        red "✗ Cannot modify Nginx configuration. The filesystem is too restrictive."
    fi
fi

# Restart containers
blue "Ensuring all containers are running..."
cd /opt/latency-space
docker compose down
docker compose up -d

# Wait for containers to start
sleep 5

# Check container status
blue "Checking container status..."
docker ps

# Try to fix DNS for status.latency.space
blue "Checking DNS for status.latency.space..."
host status.latency.space || {
    blue "Adding status.latency.space to /etc/hosts..."
    grep -q "status.latency.space" /etc/hosts || {
        echo "127.0.0.1 status.latency.space" >> /etc/hosts
    }
}

# Print final instructions
green "===================================================="
green "    Final Fix Complete - Next Steps                 "
green "===================================================="
echo ""
echo "Try accessing the following URLs:"
echo "  1. http://latency.space - Main site with static HTML"
echo "  2. http://latency.space/test.html - Test page"
echo "  3. http://status.latency.space - Status dashboard"
echo "  4. http://mars.latency.space - Mars proxy"
echo ""
echo "If you still have issues:"
echo "  - Check container logs: docker logs latency-space-proxy-1"
echo "  - Check container logs: docker logs latency-space-status-1"
echo "  - Verify Nginx is running: systemctl status nginx"
echo ""
echo "Remember: Your VM has a read-only filesystem at /var/www,"
echo "so we're using /tmp/latency-space for all files."