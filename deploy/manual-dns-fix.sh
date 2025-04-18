#!/bin/bash
# Script to manually run the DNS update process that would normally run in CI/CD

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }

blue "🔧 Manually running DNS update process"

# Check if we're in the root directory of the repo
if [ ! -d "tools" ] || [ ! -d "proxy" ]; then
  red "Error: Script must be run from the root of the repository"
  red "Current directory: $(pwd)"
  red "Please run: cd /path/to/latency-space && ./deploy/manual-dns-fix.sh"
  exit 1
fi

# Check if required environment variables are set
if [ -z "$CF_API_TOKEN" ]; then
  red "Error: CF_API_TOKEN environment variable not set"
  echo "Please set your Cloudflare API token:"
  echo "export CF_API_TOKEN=your_token"
  
  # Check if secrets file exists
  if [ -f "/opt/latency-space/secrets/cloudflare.env" ]; then
    blue "Found Cloudflare secrets file. Loading..."
    source /opt/latency-space/secrets/cloudflare.env
  else
    read -p "Enter your Cloudflare API token: " CF_API_TOKEN
    if [ -z "$CF_API_TOKEN" ]; then
      red "No API token provided. Exiting."
      exit 1
    fi
    export CF_API_TOKEN
  fi
fi

if [ -z "$SERVER_IP" ]; then
  blue "SERVER_IP not set, attempting to detect automatically..."
  SERVER_IP=$(curl -s ifconfig.me || curl -s icanhazip.com || curl -s ipecho.net/plain)
  if [ -n "$SERVER_IP" ]; then
    blue "Automatically detected server IP: $SERVER_IP"
    export SERVER_IP
  else
    red "Could not automatically detect server IP."
    read -p "Enter your server IP address: " SERVER_IP
    if [ -z "$SERVER_IP" ]; then
      red "No server IP provided. Exiting."
      exit 1
    fi
    export SERVER_IP
  fi
fi

# Simulate the DNS update step from the CI/CD pipeline
blue "Setting up Go environment..."
CURRENT_DIR=$(pwd)

# Ensure Go is installed
if ! command -v go &> /dev/null; then
  red "Go is not installed. Attempting to install..."
  if command -v apt-get &> /dev/null; then
    apt-get update && apt-get install -y golang
  elif command -v yum &> /dev/null; then
    yum install -y golang
  else
    red "Could not install Go. Please install it manually and run this script again."
    exit 1
  fi
fi

blue "Preparing DNS update tool..."
cd tools

# Create symbolic links if they don't exist
if [ ! -L "config.go" ]; then
  ln -fs ../proxy/src/config.go .
  blue "Created symlink for config.go"
fi

if [ ! -L "models.go" ]; then
  ln -fs ../proxy/src/models.go .
  blue "Created symlink for models.go"
fi

blue "Running DNS update tool..."
go run . -token "$CF_API_TOKEN" -ip "$SERVER_IP"

# Go back to original directory
cd "$CURRENT_DIR"

# Check if our fix-dns.sh script exists and run it for additional diagnostics
if [ -f "deploy/fix-dns.sh" ]; then
  blue "Running detailed DNS fix script for additional diagnostics..."
  chmod +x deploy/fix-dns.sh
  ./deploy/fix-dns.sh
else
  blue "DNS fix script not found. Update completed with standard tool only."
fi

green "✅ DNS update process completed. If status.latency.space is still not resolving:"
echo "  1. Check the output above for any errors"
echo "  2. Wait for DNS propagation (can take up to 24 hours)"
echo "  3. Verify DNS settings in Cloudflare dashboard manually"
echo "  4. Try accessing http://status.latency.space directly"
echo ""
echo "To push local commits and trigger the full CI/CD pipeline:"
echo "git push origin main"