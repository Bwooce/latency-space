#!/bin/bash
# Script to correct Nginx configuration and test

echo "Creating diagnostic page..."
mkdir -p /var/www/html

# Write a simple diagnostic HTML file
cat > /var/www/html/server-debug-report.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Latency Space Server Status</title>
    <style>
        body { font-family: sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        pre { background: #f5f5f5; padding: 10px; overflow-x: auto; }
        .section { margin-bottom: 20px; }
    </style>
</head>
<body>
    <h1>Server is Responding</h1>
    <p>This page confirms Nginx is serving direct static files.</p>
    
    <div class="section">
        <h2>1. System Status</h2>
        <pre>
Hostname: $(hostname)
Date: $(date)
Uptime: $(uptime)
        </pre>
    </div>
    
    <div class="section">
        <h2>2. Nginx Configuration</h2>
        <pre>
Nginx Version: $(nginx -v 2>&1)
Config Test: $(nginx -t 2>&1)
Active Sites:
$(ls -la /etc/nginx/sites-enabled/)
        </pre>
    </div>
    
    <div class="section">
        <h2>3. Container Status</h2>
        <pre>
$(docker ps -a)
        </pre>
    </div>
    
    <div class="section">
        <h2>4. Network Configuration</h2>
        <pre>
Latency.space IP: $(getent hosts latency.space || echo "Not found")
Docker Networks:
$(docker network ls)
        </pre>
    </div>
    
    <div class="section">
        <h2>5. Docker Volumes</h2>
        <pre>
$(docker volume ls)
        </pre>
    </div>
</body>
</html>
EOF

echo "Creating backup of original Nginx configuration..."
if [ -f /etc/nginx/sites-available/latency.space ]; then
    cp /etc/nginx/sites-available/latency.space /etc/nginx/sites-available/latency.space.bak
fi

echo "Creating simplified Nginx config..."
cat > /etc/nginx/sites-available/latency.space << 'EOF'
# Define rate limiting zones
limit_req_zone $binary_remote_addr zone=ip:10m rate=5r/m;
limit_conn_zone $binary_remote_addr zone=addr:10m;

# Server for base domain latency.space
server {
    listen 80;
    server_name latency.space www.latency.space;
    
    # Allow static file access
    location ~ \.(html|css|js|jpg|jpeg|png|gif|ico)$ {
        root /var/www/html;
        try_files $uri =404;
    }
    
    # Special diagnostics path
    location = /server-debug-report.html {
        root /var/www/html;
        try_files $uri =404;
    }
    
    # Server the main site content
    location / {
        root /var/www/html/latency-space;
        try_files $uri $uri/ /index.html;
    }
}

# Server for status.latency.space
server {
    listen 80;
    server_name status.latency.space;
    
    location / {
        return 200 "Status subdomain is working. Nginx configuration is operational.";
        add_header Content-Type text/plain;
    }
}

# Server for all other .latency.space subdomains
server {
    listen 80;
    server_name ~^[^.]+\.latency\.space$;
    
    location / {
        return 200 "Subdomain is working. Nginx configuration is operational.";
        add_header Content-Type text/plain;
    }
}
EOF

echo "Testing Nginx configuration..."
nginx -t

echo "Reloading Nginx..."
systemctl reload nginx

echo "Creating the Docker test container..."
docker run --name nginx-test -d -p 8888:80 -v /var/www/html:/usr/share/nginx/html nginx:alpine

echo "Diagnostic report available at: http://latency.space/server-debug-report.html"
echo "Also available through test container at: http://$(hostname -I | awk '{print $1}'):8888/server-debug-report.html"
echo "Try accessing both URLs to diagnose if Nginx is properly serving files."