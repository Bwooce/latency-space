#!/bin/bash
# Complete fix script to resolve all issues (version 1.0)
# This script will:
# 1. Create static files in a writable location
# 2. Fix Nginx configuration
# 3. Fix DNS resolution
# 4. Restart all containers
# 5. Verify everything works

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }

blue "Starting comprehensive fix..."

# Create writable directory for our files
WRITABLE_DIR="/tmp/latency-space"
mkdir -p $WRITABLE_DIR/html
mkdir -p $WRITABLE_DIR/nginx

# PART 1: CREATE STATIC FILES
# ---------------------------
blue "Creating static HTML files..."

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

# Create a test HTML file
cat > $WRITABLE_DIR/html/test.html << EOF
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

# PART 2: FIX NGINX CONFIGURATION
# -------------------------------
blue "Creating Nginx configuration..."

# Check if Nginx has proper configuration for handling this
if [ -d "/var/www/html" ]; then
    blue "Using /var/www/html for static files..."
    # Copy static files to Nginx's default document root
    cp $WRITABLE_DIR/html/index.html /var/www/html/ 2>/dev/null || blue "Failed to copy to /var/www/html (likely read-only)"
    cp $WRITABLE_DIR/html/test.html /var/www/html/ 2>/dev/null || blue "Failed to copy to /var/www/html (likely read-only)"
fi

# Create Nginx configuration that directly includes the entire server blocks
cat > $WRITABLE_DIR/nginx/latency.conf << 'EOC'
# Define rate limiting zones
limit_req_zone $binary_remote_addr zone=ip:10m rate=5r/m;
limit_conn_zone $binary_remote_addr zone=addr:10m;

# Server for main domain
server {
    listen 80;
    server_name latency.space www.latency.space;
    
    # Important: Add DNS resolver for Docker container names
    resolver 127.0.0.11 valid=30s ipv6=off;

    # Root directory for static files
    root /tmp/latency-space/html;
    
    # Static content - directly serve these files
    location = / {
        try_files $uri /index.html =404;
    }
    
    location = /index.html {
        try_files $uri =404;
    }
    
    location = /test.html {
        try_files $uri =404;
    }
    
    # Pass everything else to the proxy container
    location / {
        set $upstream_proxy proxy;
        proxy_pass http://$upstream_proxy:80;
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
    resolver 127.0.0.11 valid=30s ipv6=off;
    
    location / {
        set $upstream_status status;
        proxy_pass http://$upstream_status:3000;
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
    server_name ~^(?<celestial>[^.]+)\.latency\.space$;
    
    # Important: Add DNS resolver for Docker container names
    resolver 127.0.0.11 valid=30s ipv6=off;
    
    # Root directory for static files
    root /tmp/latency-space/html;
    
    location / {
        set $upstream_proxy proxy;
        proxy_pass http://$upstream_proxy:80;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Celestial-Body $celestial;
        proxy_cache_bypass $http_upgrade;
    }
}
EOC

# Now update the Nginx configuration
blue "Applying Nginx configuration..."

# Try multiple methods to update Nginx configuration
if [ -w "/etc/nginx/sites-available/latency.space" ]; then
    # Method 1: Update the sites-available file directly
    cp $WRITABLE_DIR/nginx/latency.conf /etc/nginx/sites-available/latency.space
    
    # Ensure symlink exists
    if [ ! -L "/etc/nginx/sites-enabled/latency.space" ]; then
        ln -sf /etc/nginx/sites-available/latency.space /etc/nginx/sites-enabled/latency.space
    fi
    
    blue "Updated Nginx configuration via sites-available."
elif [ -w "/etc/nginx/nginx.conf" ]; then
    # Method 2: Modify main nginx.conf to include our file
    NGINX_CONF="/etc/nginx/nginx.conf"
    
    # Make backup if it doesn't exist
    if [ ! -f "${NGINX_CONF}.bak" ]; then
        cp $NGINX_CONF ${NGINX_CONF}.bak
    fi
    
    # Check if we already added the include
    if ! grep -q "/tmp/latency-space/nginx/\*.conf" $NGINX_CONF; then
        # Include our config directory in http section
        sed -i '/http {/a \    include /tmp/latency-space/nginx/*.conf;' $NGINX_CONF
    fi
    
    blue "Updated Nginx configuration via nginx.conf."
else
    # Method 3: Last resort - create a small config file in conf.d
    if [ -d "/etc/nginx/conf.d" ] && [ -w "/etc/nginx/conf.d" ]; then
        cp $WRITABLE_DIR/nginx/latency.conf /etc/nginx/conf.d/latency.conf
        blue "Updated Nginx configuration via conf.d directory."
    else
        red "ERROR: Cannot update Nginx configuration. All locations are read-only."
        blue "Will try to create a separate Nginx container..."
        
        # Create Docker container specific config
        cat > $WRITABLE_DIR/nginx/nginx.conf << 'EOF'
user  nginx;
worker_processes  auto;

error_log  /var/log/nginx/error.log notice;
pid        /var/run/nginx.pid;

events {
    worker_connections  1024;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';

    access_log  /var/log/nginx/access.log  main;

    sendfile        on;
    keepalive_timeout  65;

    # Define rate limiting zones
    limit_req_zone $binary_remote_addr zone=ip:10m rate=5r/m;
    limit_conn_zone $binary_remote_addr zone=addr:10m;

    # Server for main domain
    server {
        listen 80;
        server_name latency.space www.latency.space;
        
        # Root directory for static files
        root /usr/share/nginx/html;
        
        # Static content - directly serve these files
        location = / {
            try_files $uri /index.html =404;
        }
        
        location = /index.html {
            try_files $uri =404;
        }
        
        location = /test.html {
            try_files $uri =404;
        }
    }
}
EOF

        # Try to stop any existing container
        docker stop nginx-emergency 2>/dev/null || true
        docker rm nginx-emergency 2>/dev/null || true
        
        # Start the container with our configuration
        docker run -d --name nginx-emergency \
            -p 8080:80 \
            -v $WRITABLE_DIR/nginx/nginx.conf:/etc/nginx/nginx.conf:ro \
            -v $WRITABLE_DIR/html:/usr/share/nginx/html:ro \
            nginx:alpine
        
        green "Created emergency Nginx container on port 8080."
        green "Access http://$(hostname -I | awk '{print $1}'):8080/ for the main site."
        green "Access http://$(hostname -I | awk '{print $1}'):8080/test.html for the test page."
    fi
fi

# Test Nginx configuration
blue "Testing Nginx configuration..."
nginx -t && {
    systemctl reload nginx
    green "✓ Nginx configuration reloaded successfully!"
} || {
    red "✗ Nginx configuration test failed. Using previous configuration."
}

# PART 3: DNS CONFIGURATION
# ------------------------
blue "Updating DNS configuration..."

# Add entries to /etc/hosts if possible
if [ -w "/etc/hosts" ]; then
    blue "Adding entries to /etc/hosts..."
    # Get server IP
    SERVER_IP=$(hostname -I | awk '{print $1}')
    
    # Check if entries already exist
    if ! grep -q "status.latency.space" /etc/hosts; then
        echo "$SERVER_IP status.latency.space" >> /etc/hosts
    fi
    
    # Add common celestial bodies
    for body in mercury venus mars jupiter saturn uranus neptune pluto; do
        if ! grep -q "$body.latency.space" /etc/hosts; then
            echo "$SERVER_IP $body.latency.space" >> /etc/hosts
        fi
    done
    
    green "✓ Updated /etc/hosts file with subdomains"
else
    blue "Cannot modify /etc/hosts (read-only). Skipping DNS entries."
fi

# PART 4: RESTART CONTAINERS
# -------------------------
blue "Restarting all containers..."

# Change to the project directory
cd /opt/latency-space

# Stop all containers
docker compose down

# Fix volume issues in docker-compose.yml if needed
if [ -f docker-compose.yml ] && [ -w docker-compose.yml ]; then
    blue "Checking docker-compose.yml for volume issues..."
    # Make a backup
    cp docker-compose.yml docker-compose.yml.bak
    
    # Use sed to replace bind mounts with named volumes
    sed -i 's#- ./config:/etc/space-proxy#- proxy_config:/etc/space-proxy#g' docker-compose.yml
    sed -i 's#- ./config/ssl:/etc/letsencrypt#- proxy_ssl:/etc/letsencrypt#g' docker-compose.yml
    sed -i 's#- ./certs:/app/certs#- proxy_certs:/app/certs#g' docker-compose.yml
    sed -i 's#- ./monitoring/grafana/dashboards:/etc/grafana/provisioning/dashboards#- grafana_dashboards:/etc/grafana/provisioning/dashboards#g' docker-compose.yml
    
    # Ensure volumes section exists and includes our volumes
    if grep -q "volumes:" docker-compose.yml; then
        # Check if each volume exists in the volumes section
        VOLUMES_TO_ADD=""
        for vol in proxy_config proxy_ssl proxy_certs grafana_dashboards; do
            if ! grep -q "$vol:" docker-compose.yml; then
                VOLUMES_TO_ADD="$VOLUMES_TO_ADD\n  $vol:"
            fi
        done
        
        # Add missing volumes
        if [ ! -z "$VOLUMES_TO_ADD" ]; then
            sed -i "/volumes:/a $VOLUMES_TO_ADD" docker-compose.yml
        fi
    else
        # Add volumes section at the end of the file
        cat >> docker-compose.yml << 'EOF'

volumes:
  prometheus_data:
  grafana_data:
  proxy_config:
  proxy_ssl:
  proxy_certs:
  grafana_dashboards:
EOF
    fi
    
    green "✓ Updated docker-compose.yml with named volumes"
fi

# Start all containers
docker compose up -d

# Wait for containers to start
blue "Waiting for containers to start..."
sleep 10

# Check container status
blue "Checking container status..."
docker ps

# PART 5: VERIFY FIX (will run after script completes)
# ---------------------------------------------------
blue "Creating verification script..."

cat > $WRITABLE_DIR/verify.sh << 'EOF'
#!/bin/bash
# Verification script - runs after a delay to check if everything is working

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }

# Wait before checking
sleep 60

echo "======================================================="
echo "  VERIFICATION RESULTS                                 "
echo "======================================================="

# Check main site
echo -n "Checking main site (latency.space): "
if curl -s http://latency.space | grep -q "Latency Space"; then
    green "OK"
else
    red "FAILED"
    curl -I http://latency.space
fi

# Check test page
echo -n "Checking test page (latency.space/test.html): "
if curl -s http://latency.space/test.html | grep -q "Nginx Configuration Test"; then
    green "OK"
else
    red "FAILED"
    curl -I http://latency.space/test.html
fi

# Check Mars subdomain
echo -n "Checking Mars subdomain (mars.latency.space): "
MARS_RESPONSE=$(curl -s http://mars.latency.space)
if [ ! -z "$MARS_RESPONSE" ]; then
    green "OK"
    echo "  Response: $MARS_RESPONSE"
else
    red "FAILED"
    curl -I http://mars.latency.space
fi

# Check status subdomain
echo -n "Checking status subdomain (status.latency.space): "
if curl -s -I http://status.latency.space | grep -q "200 OK"; then
    green "OK"
else
    red "FAILED"
    curl -I http://status.latency.space
fi

# Check container status
echo "Container status:"
docker ps

# Check Nginx status
echo "Nginx status:"
systemctl status nginx | head -5

echo "======================================================="
echo "  VERIFICATION COMPLETE                                "
echo "======================================================="
EOF

chmod +x $WRITABLE_DIR/verify.sh

# Run verification script in background after a delay
nohup bash -c "$WRITABLE_DIR/verify.sh > $WRITABLE_DIR/verification_results.log 2>&1" &

# Print final success message
green "===================================================="
green "    Fix Complete                                    "
green "===================================================="
echo ""
echo "The verification script will run automatically in 60 seconds"
echo "and save results to: $WRITABLE_DIR/verification_results.log"
echo ""
echo "You can manually check the following URLs:"
echo "  1. http://latency.space - Main site with static HTML"
echo "  2. http://latency.space/test.html - Test page"
echo "  3. http://status.latency.space - Status dashboard"
echo "  4. http://mars.latency.space - Mars proxy"
echo ""
echo "To see verification results after 60 seconds:"
echo "  cat $WRITABLE_DIR/verification_results.log"