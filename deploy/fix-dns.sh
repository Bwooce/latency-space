#!/bin/bash
# deploy/fix-dns.sh - Fix DNS resolution issues for systemd-resolved systems

set -e

echo "🔍 Checking DNS resolution..."

# Test DNS resolution
if ping -c 1 github.com &> /dev/null; then
  echo "✅ DNS resolution is working properly."
else
  echo "❌ DNS resolution failed. Attempting to fix..."
  
  # Check if using systemd-resolved (resolv.conf is a symlink)
  if [ -L /etc/resolv.conf ]; then
    echo "📋 System is using systemd-resolved"
    
    # Configure systemd-resolved directly
    cat > /etc/systemd/resolved.conf << 'EOF'
[Resolve]
DNS=8.8.8.8 8.8.4.4 1.1.1.1
FallbackDNS=9.9.9.9 149.112.112.112
DNSStubListener=yes
Cache=yes
EOF
    
    # Restart systemd-resolved
    systemctl restart systemd-resolved
    
    echo "🔄 Updated systemd-resolved configuration"
  else
    # Direct resolv.conf modification
    cp /etc/resolv.conf /etc/resolv.conf.backup
    echo "📁 Backed up current resolv.conf"
    
    # Add Google and Cloudflare DNS servers
    cat > /etc/resolv.conf << 'EOF'
nameserver 8.8.8.8
nameserver 8.8.4.4
nameserver 1.1.1.1
EOF
    
    echo "🔄 Updated resolv.conf with public DNS servers"
  fi
  
  # Test DNS resolution again
  if ping -c 1 github.com &> /dev/null; then
    echo "✅ DNS resolution fixed successfully!"
  else
    echo "❌ DNS resolution still failing. Trying alternative approach..."
    
    # Create a custom resolv.conf and remove symlink
    if [ -L /etc/resolv.conf ]; then
      echo "📋 Removing resolv.conf symlink and creating direct file"
      rm /etc/resolv.conf
      cat > /etc/resolv.conf << 'EOF'
nameserver 8.8.8.8
nameserver 8.8.4.4
nameserver 1.1.1.1
options timeout:2 attempts:5
EOF
      
      # Check one more time
      if ping -c 1 github.com &> /dev/null; then
        echo "✅ DNS resolution fixed with direct resolv.conf file!"
      else
        echo "❌ DNS resolution still failing after direct file creation."
      fi
    fi
    
    # Network connectivity check
    echo "🔍 Checking general network connectivity..."
    if ping -c 1 8.8.8.8 &> /dev/null; then
      echo "✅ Network connectivity to 8.8.8.8 is working."
      echo "❌ The issue is specific to DNS resolution."
    else
      echo "❌ No network connectivity. Check network interfaces and routing."
    fi
  fi
fi

# Check Docker DNS configuration
echo "🔍 Checking Docker DNS configuration..."
mkdir -p /etc/docker

if [ -f /etc/docker/daemon.json ]; then
  echo "📁 Current Docker daemon configuration:"
  cat /etc/docker/daemon.json
  
  # Check if DNS is configured in daemon.json
  if grep -q "dns" /etc/docker/daemon.json; then
    echo "✅ Docker DNS configuration exists."
  else
    echo "⚠️ No DNS configuration found in Docker daemon.json."
    echo "📝 Creating a new Docker daemon configuration with public DNS servers..."
    
    # Create or update Docker DNS configuration
    if [ -s /etc/docker/daemon.json ]; then
      # File exists and is not empty - need to merge
      TMP=$(mktemp)
      if command -v jq &> /dev/null; then
        jq '. + {"dns": ["8.8.8.8", "8.8.4.4", "1.1.1.1"]}' /etc/docker/daemon.json > "$TMP" && mv "$TMP" /etc/docker/daemon.json
      else
        # Fallback if jq not available
        echo '{"dns": ["8.8.8.8", "8.8.4.4", "1.1.1.1"]}' > /etc/docker/daemon.json
      fi
    else
      # File doesn't exist or is empty
      echo '{"dns": ["8.8.8.8", "8.8.4.4", "1.1.1.1"]}' > /etc/docker/daemon.json
    fi
    
    echo "🔄 Restarting Docker service..."
    systemctl restart docker
  fi
else
  echo "📁 Creating Docker daemon configuration with public DNS servers..."
  echo '{"dns": ["8.8.8.8", "8.8.4.4", "1.1.1.1"]}' > /etc/docker/daemon.json
  
  echo "🔄 Restarting Docker service..."
  systemctl restart docker
fi

echo "✅ DNS configuration check completed."