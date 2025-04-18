#!/bin/bash
# Script to fix the read-only filesystem issue for latency.space

# Print colored output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }

blue "Checking filesystem status..."
df -h

# Check which filesystems are mounted read-only
blue "Checking for read-only filesystems..."
mount | grep " ro,"

# Create a writable directory for our files
blue "Creating writable directories..."
WRITABLE_DIR="/tmp/latency-space"
mkdir -p $WRITABLE_DIR/html
mkdir -p $WRITABLE_DIR/nginx

# Create diagnostic files in writable location
blue "Creating diagnostic files..."
cat > $WRITABLE_DIR/html/test.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Latency Space Test</title>
</head>
<body>
    <h1>Filesystem Test Success</h1>
    <p>If you can see this page, the writable filesystem solution is working.</p>
    <p>Generated at: $(date)</p>
    <p>Hostname: $(hostname)</p>
</body>
</html>
EOF

# Create a modified Nginx configuration that uses the writable directory
blue "Creating modified Nginx configuration..."
cat > $WRITABLE_DIR/nginx/latency.space << 'EOC'
# Define rate limiting zones
limit_req_zone $binary_remote_addr zone=ip:10m rate=5r/m;
limit_conn_zone $binary_remote_addr zone=addr:10m;

# Server for base domain latency.space
server {
    listen 80;
    server_name latency.space www.latency.space;
    
    # Static content test location
    location /test.html {
        alias /tmp/latency-space/html/test.html;
        default_type text/html;
    }
    
    # Proxy to Docker containers for normal operation
    location / {
        proxy_pass http://proxy:80;
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
    
    location / {
        proxy_pass http://status:3000;
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
    
    location / {
        proxy_pass http://proxy:80;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
EOC

# Create symbolic link from Nginx sites directory to our writable config
blue "Updating Nginx configuration..."
if [ -f /etc/nginx/sites-available/latency.space ]; then
    mv /etc/nginx/sites-available/latency.space /etc/nginx/sites-available/latency.space.bak
fi

# Create a hard link or copy instead of symlink in case /etc is also read-only
cat $WRITABLE_DIR/nginx/latency.space > /etc/nginx/sites-available/latency.space || {
    red "ERROR: Failed to modify /etc/nginx/sites-available. It might be read-only too."
    blue "Trying alternative approach with nginx.conf..."
    
    # Alternative: Modify nginx.conf directly if it's writable
    NGINX_CONF="/etc/nginx/nginx.conf"
    if [ -w "$NGINX_CONF" ]; then
        cp $NGINX_CONF ${NGINX_CONF}.bak
        
        # Include our config directory in http section
        sed -i '/http {/a \    include /tmp/latency-space/nginx/*.conf;' $NGINX_CONF
        
        # Rename our config to ensure it's loaded
        mv $WRITABLE_DIR/nginx/latency.space $WRITABLE_DIR/nginx/latency.conf
        
        blue "Updated main Nginx configuration to include our directory."
    else
        red "ERROR: Cannot modify Nginx configuration. The filesystem is too restrictive."
        blue "Attempting to create a containerized solution..."
        
        # Create a container that serves our diagnostic page and forwards to other services
        cat > $WRITABLE_DIR/nginx/nginx.conf << 'EOF'
events {
    worker_connections 1024;
}

http {
    server {
        listen 80;
        
        location /test.html {
            root /usr/share/nginx/html;
        }
        
        location / {
            return 200 "Nginx container is running. The host filesystem appears to be read-only.";
            add_header Content-Type text/plain;
        }
    }
}
EOF
        
        # Copy our test file to a location the container will use
        cp $WRITABLE_DIR/html/test.html $WRITABLE_DIR/nginx/
        
        # Run a containerized Nginx
        docker stop nginx-emergency || true
        docker rm nginx-emergency || true
        docker run -d --name nginx-emergency -p 8080:80 -v $WRITABLE_DIR/nginx/nginx.conf:/etc/nginx/nginx.conf:ro -v $WRITABLE_DIR/nginx/test.html:/usr/share/nginx/html/test.html:ro nginx:alpine
        
        green "Emergency container started on port 8080. Access http://$(hostname -I | awk '{print $1}'):8080/test.html"
        exit 0
    fi
}

# Test and reload Nginx
blue "Testing Nginx configuration..."
nginx -t && {
    systemctl reload nginx
    green "Nginx configuration updated successfully!"
    green "Test page available at: http://latency.space/test.html"
} || {
    red "Nginx test failed. Using containerized solution..."
    
    # Run a containerized Nginx as fallback
    docker stop nginx-emergency || true
    docker rm nginx-emergency || true
    docker run -d --name nginx-emergency -p 8080:80 -v $WRITABLE_DIR/nginx/test.html:/usr/share/nginx/html/test.html:ro nginx:alpine
    
    green "Emergency container started on port 8080. Access http://$(hostname -I | awk '{print $1}'):8080/test.html"
}

blue "Checking Docker container status..."
docker ps

blue "Remember to check the Docker container logs for more details:"
echo "  docker logs latency-space-proxy-1"
echo "  docker logs latency-space-status-1"