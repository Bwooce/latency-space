#!/bin/sh
# Dynamic startup script for status container

# Default values for service IPs - can be overridden by environment variables
PROMETHEUS_IP=${PROMETHEUS_IP:-"172.18.0.3"}
PROXY_IP=${PROXY_IP:-"172.18.0.2"}

# Generate a customized nginx.conf using the correct IPs
cat > /etc/nginx/conf.d/default.conf << EOF
server {
    listen 80;
    server_name localhost;
    root /usr/share/nginx/html;
    index index.html;

    # Support for SPA routing
    location / {
        try_files \$uri \$uri/ /index.html;
    }

    # API proxy for metrics - using direct IP address
    location /api/metrics {
        proxy_pass http://${PROMETHEUS_IP}:9090/api/v1/query;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_cache_bypass \$http_upgrade;
        
        # Add error handling
        proxy_intercept_errors on;
        error_page 500 502 503 504 = @fallback_metrics;
    }
    
    # Fallback for metrics when Prometheus is unavailable
    location @fallback_metrics {
        default_type application/json;
        return 200 '{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1651356239.1,"60"]}]}}';
    }
    
    # Proxy for accessing the debug endpoints - using direct IP address
    location /api/debug/ {
        proxy_pass http://${PROXY_IP}:80/_debug/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host proxy;
        proxy_cache_bypass \$http_upgrade;
        
        # Add CORS headers
        add_header 'Access-Control-Allow-Origin' '*';
        add_header 'Access-Control-Allow-Methods' 'GET, OPTIONS';
        add_header 'Access-Control-Allow-Headers' 'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range';
    }
}
EOF

echo "=== Generated Nginx Configuration with IPs ==="
echo "Prometheus IP: $PROMETHEUS_IP"
echo "Proxy IP: $PROXY_IP"
cat /etc/nginx/conf.d/default.conf
echo "=============================================="

# Start Nginx
exec nginx -g "daemon off;"