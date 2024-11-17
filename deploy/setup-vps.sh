#!/bin/bash
# deploy/setup-vps.sh

# Update system
apt-get update && apt-get upgrade -y

# Install required packages
apt-get install -y \
    docker.io \
    docker-compose \
    nginx \
    certbot \
    python3-certbot-nginx \
    ufw

# Configure firewall
ufw allow 22
ufw allow 80
ufw allow 443
ufw allow 53/udp
ufw --force enable

# Create application directory
mkdir -p /opt/latency-space
chown -R $USER:$USER /opt/latency-space

# Install Docker
systemctl start docker
systemctl enable docker

# Create docker network
docker network create space-net

# Setup SSL directory
mkdir -p /etc/letsencrypt

# Setup Nginx
cat > /etc/nginx/sites-available/latency.space << 'EOF'
server {
    listen 80;
    server_name *.latency.space;
    
    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
EOF

ln -s /etc/nginx/sites-available/latency.space /etc/nginx/sites-enabled/
rm /etc/nginx/sites-enabled/default

# Reload Nginx
systemctl reload nginx

# Setup SSL
certbot --nginx -d latency.space -d '*.latency.space' --agree-tos -m $SSL_EMAIL -n

echo "VPS setup completed!"

