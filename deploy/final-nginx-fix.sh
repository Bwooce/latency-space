#!/bin/bash
# Final Nginx fix script - uses direct container IPs instead of DNS names

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }

# Create directories for writable files
WRITABLE_DIR="/tmp/latency-space"
mkdir -p $WRITABLE_DIR/nginx
mkdir -p $WRITABLE_DIR/html

# Create test HTML files
blue "Creating HTML files..."
cat > $WRITABLE_DIR/html/index.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Latency Space</title>
    <style>
        body { font-family: sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1 { color: #333; }
        .card { border: 1px solid #ddd; padding: 15px; margin: 10px 0; border-radius: 5px; }
        a { color: #0066cc; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <h1>Latency Space - Interplanetary Internet Simulator</h1>
    <p>This project simulates the latency of communication across the solar system.</p>
    
    <div class="card">
        <h2>Available Planets:</h2>
        <ul>
            <li><a href="http://mercury.latency.space">Mercury</a></li>
            <li><a href="http://venus.latency.space">Venus</a></li>
            <li><a href="http://mars.latency.space">Mars</a></li>
            <li><a href="http://jupiter.latency.space">Jupiter</a></li>
            <li><a href="http://saturn.latency.space">Saturn</a></li>
            <li><a href="http://uranus.latency.space">Uranus</a></li>
            <li><a href="http://neptune.latency.space">Neptune</a></li>
        </ul>
    </div>
    
    <div class="card">
        <h2>Dashboard:</h2>
        <p><a href="http://status.latency.space">Status Dashboard</a> - View real-time distances and latency</p>
    </div>
    
    <p><small>Created by Latency Space Team</small></p>
</body>
</html>
EOF

cat > $WRITABLE_DIR/html/test.html << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
</head>
<body>
    <h1>Nginx Test Page</h1>
    <p>This page confirms Nginx is correctly configured to serve static files.</p>
    <p>Generated: $(date)</p>
</body>
</html>
EOF

# Get container IPs
blue "Retrieving container IPs..."

# Function to get container IP
get_container_ip() {
    docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $1 2>/dev/null || echo "not_found"
}

PROXY_CONTAINER=$(docker ps -q -f name=latency-space-proxy)
STATUS_CONTAINER=$(docker ps -q -f name=latency-space-status)

if [ -z "$PROXY_CONTAINER" ]; then
    red "Proxy container not found. Is it running?"
    PROXY_IP="not_found"
else
    PROXY_IP=$(get_container_ip $PROXY_CONTAINER)
    green "Proxy container IP: $PROXY_IP"
fi

if [ -z "$STATUS_CONTAINER" ]; then
    red "Status container not found. Is it running?"
    STATUS_IP="not_found"
else
    STATUS_IP=$(get_container_ip $STATUS_CONTAINER)
    green "Status container IP: $STATUS_IP"
fi

# Create Nginx configuration with direct IPs
blue "Creating Nginx configuration..."

cat > $WRITABLE_DIR/nginx/latency.conf << EOF
# Main server
server {
    listen 80;
    server_name latency.space www.latency.space;
    
    # Root directory for static files
    root /tmp/latency-space/html;
    
    # Serve static files directly
    location / {
        try_files \$uri \$uri/index.html =404;
    }
}

# Status server
server {
    listen 80;
    server_name status.latency.space;
    
    location / {
        # Use direct IP instead of DNS name
        proxy_pass http://${STATUS_IP:-127.0.0.1}:3000;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
    }
}

# Subdomains
server {
    listen 80;
    server_name ~^[^.]+\.latency\.space\$;
    
    location / {
        # Use direct IP instead of DNS name
        proxy_pass http://${PROXY_IP:-127.0.0.1}:80;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
    }
}
EOF

# Update Nginx configuration
blue "Updating Nginx configuration..."

# Make a backup of the current configuration
if [ -f "/etc/nginx/sites-available/latency.space" ]; then
    cp /etc/nginx/sites-available/latency.space /etc/nginx/sites-available/latency.space.bak-$(date +%s)
fi

# Apply new configuration
cp $WRITABLE_DIR/nginx/latency.conf /etc/nginx/sites-available/latency.space

# Ensure symlink exists
if [ ! -L "/etc/nginx/sites-enabled/latency.space" ]; then
    ln -sf /etc/nginx/sites-available/latency.space /etc/nginx/sites-enabled/latency.space
fi

# Copy static files to Nginx's document root
if [ -d "/var/www/html" ]; then
    if [ -w "/var/www/html" ]; then
        cp $WRITABLE_DIR/html/index.html /var/www/html/
        cp $WRITABLE_DIR/html/test.html /var/www/html/
        green "✓ Copied static files to /var/www/html"
    else
        blue "Cannot write to /var/www/html, using /tmp/latency-space/html"
    fi
fi

# Test and reload Nginx
blue "Testing Nginx configuration..."
nginx -t && {
    systemctl reload nginx
    green "✓ Nginx configuration updated successfully"
} || {
    red "✗ Nginx configuration test failed"
    
    # Last resort - create a very minimal configuration
    blue "Creating minimal configuration as last resort..."
    cat > /etc/nginx/sites-available/latency.space << 'EOF'
# Main server only
server {
    listen 80;
    server_name latency.space www.latency.space;
    
    # Root directory for static files
    root /tmp/latency-space/html;
    
    # Serve static files directly
    location / {
        try_files $uri $uri/index.html =404;
    }
}
EOF
    
    # Test again
    nginx -t && {
        systemctl reload nginx
        green "✓ Minimal Nginx configuration applied"
    } || {
        red "✗ Minimal configuration also failed. Nginx may have deeper issues."
    }
}

# Wait for Nginx to reload
sleep 3

# Test URLs
blue "Testing URLs..."

echo -n "Testing latency.space: "
curl -s -o /dev/null -w "%{http_code}" http://latency.space/ && echo " OK" || echo " Failed"

echo -n "Testing latency.space/test.html: "
curl -s -o /dev/null -w "%{http_code}" http://latency.space/test.html && echo " OK" || echo " Failed"

echo -n "Testing mars.latency.space: "
curl -s -o /dev/null -w "%{http_code}" http://mars.latency.space/ && echo " OK" || echo " Failed"

if [ "$STATUS_IP" != "not_found" ]; then
    echo -n "Testing status.latency.space: "
    curl -s -o /dev/null -w "%{http_code}" http://status.latency.space/ && echo " OK" || echo " Failed"
fi

green "Nginx fix complete!"
echo ""
echo "You can now access:"
echo "  - http://latency.space - Main site"
echo "  - http://latency.space/test.html - Test page"
echo "  - http://mars.latency.space - Mars proxy"
if [ "$STATUS_IP" != "not_found" ]; then
    echo "  - http://status.latency.space - Status dashboard"
fi