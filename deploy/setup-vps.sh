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

# We're using the standard Docker installation, not snap
# The code for snap Docker has been removed as it's not recommended 
# due to stability issues with containerd

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
    docker-compose-plugin \
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

# Clone repository
echo "Cloning the repository..."
cd /opt/latency-space
if [ ! -d ".git" ]; then
  git clone https://github.com/Bwooce/latency-space.git .
else
  git pull
fi

echo "VPS setup completed!"
echo "Next steps:"
echo "1. Configure Nginx using: ./deploy/update-nginx.sh"
echo "2. Start the services with: docker compose up -d"