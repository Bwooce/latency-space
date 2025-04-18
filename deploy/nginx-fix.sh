#!/bin/bash
# Script to fix Nginx configuration error

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }

# Create a simple, working Nginx configuration
WRITABLE_DIR="/tmp/latency-space"
mkdir -p $WRITABLE_DIR/nginx
mkdir -p $WRITABLE_DIR/html

# Create a test HTML file in the writable directory
cat > $WRITABLE_DIR/html/index.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Latency Space</title>
    <style>
        body { font-family: sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1 { color: #333; }
        .section { margin-bottom: 20px; }
    </style>
</head>
<body>
    <h1>Latency Space</h1>
    
    <div class="section">
        <h2>Celestial Bodies:</h2>
        <ul>
            <li><a href="http://mars.latency.space">Mars</a></li>
            <li><a href="http://jupiter.latency.space">Jupiter</a></li>
            <li><a href="http://saturn.latency.space">Saturn</a></li>
        </ul>
    </div>
    
    <div class="section">
        <h2>Status:</h2>
        <p><a href="http://status.latency.space">Status Dashboard</a></p>
    </div>
</body>
</html>
EOF

cat > $WRITABLE_DIR/html/test.html << 'EOF'
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

# Create a minimal, working Nginx configuration
cat > $WRITABLE_DIR/nginx/latency.conf << 'EOF'
# Main server
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

# Status server
server {
    listen 80;
    server_name status.latency.space;
    
    # Use Docker DNS resolver
    resolver 127.0.0.11 valid=30s;
    
    location / {
        proxy_pass http://status:3000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}

# Subdomains
server {
    listen 80;
    server_name ~^[^.]+\.latency\.space$;
    
    # Use Docker DNS resolver
    resolver 127.0.0.11 valid=30s;
    
    location / {
        proxy_pass http://proxy:80;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
EOF

# Update Nginx configuration
blue "Updating Nginx configuration..."

if [ -w "/etc/nginx/sites-available/latency.space" ]; then
    # Replace the existing configuration
    mv /etc/nginx/sites-available/latency.space /etc/nginx/sites-available/latency.space.old
    cp $WRITABLE_DIR/nginx/latency.conf /etc/nginx/sites-available/latency.space
    
    # Ensure symlink exists
    if [ ! -L "/etc/nginx/sites-enabled/latency.space" ]; then
        ln -sf /etc/nginx/sites-available/latency.space /etc/nginx/sites-enabled/latency.space
    fi
    
    # Test and reload Nginx
    blue "Testing Nginx configuration..."
    nginx -t
    if [ $? -eq 0 ]; then
        systemctl reload nginx
        green "✓ Nginx configuration updated successfully"
    else
        red "✗ Nginx configuration test failed"
        blue "Restoring original configuration..."
        mv /etc/nginx/sites-available/latency.space.old /etc/nginx/sites-available/latency.space
        systemctl reload nginx
    fi
else
    red "✗ Cannot write to /etc/nginx/sites-available/latency.space"
fi

# Copy static files to a location Nginx can access
if [ -d "/var/www/html" ]; then
    if [ -w "/var/www/html" ]; then
        cp $WRITABLE_DIR/html/index.html /var/www/html/
        cp $WRITABLE_DIR/html/test.html /var/www/html/
        green "✓ Copied static files to /var/www/html"
    fi
fi

# Test URLs
blue "Testing URLs (we'll wait 5 seconds for Nginx to reload)..."
sleep 5

echo -n "Testing latency.space: "
curl -s -o /dev/null -w "%{http_code}" http://latency.space/ && echo " OK" || echo " Failed"

echo -n "Testing latency.space/test.html: "
curl -s -o /dev/null -w "%{http_code}" http://latency.space/test.html && echo " OK" || echo " Failed"

echo -n "Testing mars.latency.space: "
curl -s -o /dev/null -w "%{http_code}" http://mars.latency.space/ && echo " OK" || echo " Failed"

green "Nginx fix complete!"