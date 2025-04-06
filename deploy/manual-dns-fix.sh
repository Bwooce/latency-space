#!/bin/bash
# Simple script to fix DNS issues on systemd-resolved systems
# Run this directly on the server when SSH'd in

# Configure systemd-resolved
echo "Configuring systemd-resolved..."
cat > /etc/systemd/resolved.conf << 'EOF'
[Resolve]
DNS=8.8.8.8 8.8.4.4 1.1.1.1
FallbackDNS=9.9.9.9 149.112.112.112
DNSStubListener=yes
Cache=yes
EOF

# Restart systemd-resolved
systemctl restart systemd-resolved

# Configure Docker DNS
mkdir -p /etc/docker
cat > /etc/docker/daemon.json << 'EOF'
{
  "dns": ["8.8.8.8", "8.8.4.4", "1.1.1.1"]
}
EOF

# Restart Docker
systemctl restart docker

# Test the DNS resolution
echo "Testing DNS resolution..."
if ping -c 1 github.com &> /dev/null; then
  echo "Success! DNS is now working."
else
  echo "DNS still not working. Trying more aggressive fix..."
  
  # Breaking the symlink as a last resort
  if [ -L /etc/resolv.conf ]; then
    rm /etc/resolv.conf
    cat > /etc/resolv.conf << 'EOF'
nameserver 8.8.8.8
nameserver 8.8.4.4
nameserver 1.1.1.1
options timeout:2 attempts:5
EOF
  fi
  
  # Test again
  if ping -c 1 github.com &> /dev/null; then
    echo "Success! DNS is now working after aggressive fix."
  else
    echo "DNS still not working. Please contact your system administrator."
  fi
fi

# Try to run a git pull to verify
echo "Testing git pull..."
cd /opt/latency-space || mkdir -p /opt/latency-space
if [ -d ".git" ]; then
  if git pull; then
    echo "Git pull successful! DNS is working correctly."
  else
    echo "Git pull failed. DNS may still have issues."
  fi
else
  echo "Git repository not found. Cannot test git pull."
fi