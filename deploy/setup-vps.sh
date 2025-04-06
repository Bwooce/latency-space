#!/bin/bash
# deploy/setup-vps.sh

# Ensure the script exits on any error
set -e

# Configure proper DNS for systemd-resolved systems
echo "Configuring DNS..."
if [ -L /etc/resolv.conf ]; then
  echo "System using systemd-resolved, configuring it properly..."
  
  # Configure systemd-resolved
  cat > /etc/systemd/resolved.conf << 'EOF'
[Resolve]
DNS=8.8.8.8 8.8.4.4 1.1.1.1
FallbackDNS=9.9.9.9 149.112.112.112
DNSStubListener=yes
Cache=yes
EOF

  # Restart systemd-resolved
  systemctl restart systemd-resolved
else
  # Direct modification for systems not using systemd-resolved
  cp /etc/resolv.conf /etc/resolv.conf.backup
  cat > /etc/resolv.conf << 'EOF'
nameserver 8.8.8.8
nameserver 8.8.4.4
nameserver 1.1.1.1
EOF
fi

# Configure Docker DNS
mkdir -p /etc/docker
cat > /etc/docker/daemon.json << 'EOF'
{
  "dns": ["8.8.8.8", "8.8.4.4", "1.1.1.1"]
}
EOF

# Verify DNS resolution
echo "Verifying DNS resolution..."
if ! ping -c 1 github.com &> /dev/null; then
  echo "Warning: DNS resolution still failing. Trying alternative approach..."
  
  # Try breaking the symlink as a last resort
  if [ -L /etc/resolv.conf ]; then
    echo "Removing symlink and creating direct resolv.conf file..."
    rm /etc/resolv.conf
    cat > /etc/resolv.conf << 'EOF'
nameserver 8.8.8.8
nameserver 8.8.4.4
nameserver 1.1.1.1
options timeout:2 attempts:5
EOF
  fi
  
  # Check if that fixed it
  if ! ping -c 1 github.com &> /dev/null; then
    echo "Warning: DNS resolution still failing. Continuing anyway..."
  fi
fi

# Update system
echo "Updating system packages..."
apt-get update && apt-get upgrade -y

# Install required packages
echo "Installing required packages..."
apt-get install -y \
    docker.io \
    docker-compose \
    nginx \
    certbot \
    python3-certbot-nginx \
    ufw \
    jq \
    curl \
    dnsutils \
    net-tools

# Configure firewall
echo "Configuring firewall..."
ufw allow 22
ufw allow 80
ufw allow 443
ufw allow 53/udp
ufw allow 1080 # Allow SOCKS proxy
ufw allow 9090 # Prometheus metrics
ufw --force enable

# Create application directory
echo "Creating application directory..."
mkdir -p /opt/latency-space
chown -R $USER:$USER /opt/latency-space

# Install Docker
echo "Starting Docker service..."
systemctl start docker
systemctl enable docker
systemctl restart docker # Apply DNS settings

# Create docker network
echo "Creating Docker network..."
docker network create space-net || echo "Network may already exist, continuing..."

# Setup SSL directory
echo "Setting up SSL directory..."
mkdir -p /etc/letsencrypt

# Setup Nginx
echo "Configuring Nginx..."
cat > /etc/nginx/sites-available/latency.space << 'EOF'
# HTTP server for all subdomains (including multi-level ones)
server {
    listen 80;
    server_name .latency.space;
    
    # Handle Let's Encrypt validation challenges
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }
    
    # For known subdomains with valid certificates, redirect to HTTPS
    if ($host ~* ^([^.]+)\.latency\.space$) {
        return 301 https://$host$request_uri;
    }
    
    # For other subdomains (multi-level ones), serve over HTTP directly
    location / {
        proxy_pass http://localhost:8080;  # Changed from 3000 to 8080 (proxy container)
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_cache_bypass $http_upgrade;
    }
}

# HTTPS server for first-level subdomains
server {
    listen 443 ssl;
    server_name latency.space *.latency.space;
    
    # SSL certificates will be added by certbot
    
    location / {
        proxy_pass http://localhost:8080;  # Changed from 3000 to 8080 (proxy container)
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}

# Server for status dashboard
server {
    listen 80;
    server_name status.latency.space;
    
    location / {
        proxy_pass http://localhost:3000;  # Status service
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
EOF

# Create directory for Let's Encrypt validation
mkdir -p /var/www/html/.well-known/acme-challenge

ln -sf /etc/nginx/sites-available/latency.space /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default

# Reload Nginx
echo "Reloading Nginx..."
systemctl reload nginx

# Setup SSL for base domain and first-level subdomains
echo "Setting up SSL certificates..."
certbot --nginx -d latency.space -d '*.latency.space' --agree-tos -m $SSL_EMAIL -n || echo "SSL setup failed, please run manually after DNS is resolved"

# Clone repository
echo "Cloning the repository..."
cd /opt/latency-space
if [ ! -d ".git" ]; then
  git clone https://github.com/Bwooce/latency-space.git .
else
  git pull
fi

# Copy fix-dns script for easy access
cp deploy/fix-dns.sh /usr/local/bin/fix-dns
chmod +x /usr/local/bin/fix-dns

echo "VPS setup completed!"
echo "You can run 'fix-dns' at any time to troubleshoot DNS issues."