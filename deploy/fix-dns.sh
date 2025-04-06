#!/bin/bash
# deploy/fix-dns.sh - Fix DNS resolution issues

set -e

echo "🔍 Checking DNS resolution..."

# Test DNS resolution
if ping -c 1 github.com &> /dev/null; then
  echo "✅ DNS resolution is working properly."
else
  echo "❌ DNS resolution failed. Attempting to fix..."
  
  # Backup current resolv.conf
  if [ -f /etc/resolv.conf ]; then
    cp /etc/resolv.conf /etc/resolv.conf.backup
    echo "📁 Backed up current resolv.conf"
  fi
  
  # Add Google and Cloudflare DNS servers
  echo "nameserver 8.8.8.8" > /etc/resolv.conf
  echo "nameserver 8.8.4.4" >> /etc/resolv.conf
  echo "nameserver 1.1.1.1" >> /etc/resolv.conf
  
  echo "🔄 Updated resolv.conf with public DNS servers"
  
  # Test DNS resolution again
  if ping -c 1 github.com &> /dev/null; then
    echo "✅ DNS resolution fixed successfully!"
  else
    echo "❌ DNS resolution still failing. Additional troubleshooting required."
    
    # Check if systemd-resolved is running
    if systemctl is-active systemd-resolved &> /dev/null; then
      echo "🔍 systemd-resolved is running. Restarting service..."
      systemctl restart systemd-resolved
      
      # Test again after restart
      if ping -c 1 github.com &> /dev/null; then
        echo "✅ DNS resolution fixed after restarting systemd-resolved!"
      else
        echo "❌ DNS resolution still failing after service restart."
      fi
    else
      echo "ℹ️ systemd-resolved is not active."
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
      jq '. + {"dns": ["8.8.8.8", "8.8.4.4", "1.1.1.1"]}' /etc/docker/daemon.json > "$TMP" && mv "$TMP" /etc/docker/daemon.json
    else
      # File doesn't exist or is empty
      echo '{"dns": ["8.8.8.8", "8.8.4.4", "1.1.1.1"]}' > /etc/docker/daemon.json
    fi
    
    echo "🔄 Restarting Docker service..."
    systemctl restart docker
  fi
else
  echo "📁 Creating Docker daemon configuration with public DNS servers..."
  mkdir -p /etc/docker
  echo '{"dns": ["8.8.8.8", "8.8.4.4", "1.1.1.1"]}' > /etc/docker/daemon.json
  
  echo "🔄 Restarting Docker service..."
  systemctl restart docker
fi

echo "✅ DNS configuration check completed."