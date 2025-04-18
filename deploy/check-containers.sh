#!/bin/bash
# Script to check container logs and fix Nginx configuration

# Print colored output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }

blue "Checking container status..."
docker ps -a

blue "Checking running containers' network setup..."
docker network inspect space-net

blue "Checking proxy container logs..."
docker logs latency-space-proxy-1

blue "Creating fixed Nginx configuration with proper resolver..."
WRITABLE_DIR="/tmp/latency-space"
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
    
    # Static content test location
    location /test.html {
        alias /tmp/latency-space/html/test.html;
        default_type text/html;
    }
    
    # Proxy to Docker containers for normal operation
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

blue "Creating test HTML file..."
mkdir -p $WRITABLE_DIR/html
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
        <li>Try accessing <a href="/">the main site</a> to see if proxy forwarding works</li>
        <li>Try accessing <a href="http://status.latency.space">status.latency.space</a> to check subdomain resolution</li>
        <li>Try accessing <a href="http://mars.latency.space">mars.latency.space</a> to check celestial body proxying</li>
    </ul>
</body>
</html>
EOF

blue "Updating Nginx configuration..."
if [ -w "/etc/nginx/sites-available/latency.space" ]; then
    cat $WRITABLE_DIR/nginx/latency.conf > /etc/nginx/sites-available/latency.space
    
    blue "Testing Nginx configuration..."
    nginx -t && {
        systemctl reload nginx
        green "Nginx configuration updated successfully!"
        green "Test page available at: http://latency.space/test.html"
    } || {
        red "Nginx test failed. Trying alternative approach..."
        
        # Alternative: Modify nginx.conf directly if it's writable
        NGINX_CONF="/etc/nginx/nginx.conf"
        if [ -w "$NGINX_CONF" ]; then
            cp $NGINX_CONF ${NGINX_CONF}.bak
            
            # Check if we already added the include
            if ! grep -q "/tmp/latency-space/nginx/\*.conf" $NGINX_CONF; then
                # Include our config directory in http section
                sed -i '/http {/a \    include /tmp/latency-space/nginx/*.conf;' $NGINX_CONF
            fi
            
            blue "Updated main Nginx configuration to include our directory."
            
            # Test and reload
            nginx -t && {
                systemctl reload nginx
                green "Nginx configuration updated successfully!"
                green "Test page available at: http://latency.space/test.html"
            } || {
                red "Nginx test still failed."
            }
        else
            red "ERROR: Cannot modify Nginx configuration. The filesystem is too restrictive."
        fi
    }
else
    red "ERROR: Cannot modify /etc/nginx/sites-available. It might be read-only."
    
    # Alternative: Modify nginx.conf directly if it's writable
    NGINX_CONF="/etc/nginx/nginx.conf"
    if [ -w "$NGINX_CONF" ]; then
        cp $NGINX_CONF ${NGINX_CONF}.bak
        
        # Check if we already added the include
        if ! grep -q "/tmp/latency-space/nginx/\*.conf" $NGINX_CONF; then
            # Include our config directory in http section
            sed -i '/http {/a \    include /tmp/latency-space/nginx/*.conf;' $NGINX_CONF
        fi
        
        blue "Updated main Nginx configuration to include our directory."
        
        # Test and reload
        nginx -t && {
            systemctl reload nginx
            green "Nginx configuration updated successfully!"
            green "Test page available at: http://latency.space/test.html"
        } || {
            red "Nginx test still failed."
        }
    else
        red "ERROR: Cannot modify Nginx configuration. The filesystem is too restrictive."
    fi
fi

blue "Checking if all required containers are running..."
containers=("proxy" "status" "prometheus" "grafana")
for container in "${containers[@]}"; do
    container_id=$(docker ps -q -f name="latency-space-$container")
    if [ -n "$container_id" ]; then
        green "✓ $container container is running"
    else
        red "✗ $container container is not running. Starting it..."
        docker compose up -d $container
    fi
done

blue "Diagnostic information complete."
echo "Access http://latency.space/test.html to test static file serving."
echo "Once that works, try accessing the main site and subdomains:"
echo "  - http://latency.space"
echo "  - http://status.latency.space"
echo "  - http://mars.latency.space"