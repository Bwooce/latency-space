#!/bin/bash
# Script to install Nginx configuration from repository

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }
yellow() { echo -e "\033[0;33m$1\033[0m"; }

# Check if script is run as root
if [ "$EUID" -ne 0 ]; then
  red "Please run this script as root"
  echo "Try: sudo $0"
  exit 1
fi

# Check which configuration to install
CONFIG_TYPE="standard"
if [ "$1" == "minimal" ]; then
  CONFIG_TYPE="minimal"
fi

blue "Installing $CONFIG_TYPE Nginx configuration from repository..."

# Check if we're in the right directory
if [ ! -d "config/nginx" ]; then
  red "Nginx config directory not found. Please run this script from the latency-space directory."
  exit 1
fi

# Determine which config file to use
if [ "$CONFIG_TYPE" == "minimal" ]; then
  CONFIG_FILE="config/nginx/minimal.conf"
  if [ ! -f "$CONFIG_FILE" ]; then
    red "Minimal configuration file not found at $CONFIG_FILE"
    exit 1
  fi
else
  CONFIG_FILE="config/nginx/latency.space.conf"
  if [ ! -f "$CONFIG_FILE" ]; then
    red "Standard configuration file not found at $CONFIG_FILE"
    exit 1
  fi
fi

# Create sites-available and sites-enabled directories if they don't exist
mkdir -p /etc/nginx/sites-available /etc/nginx/sites-enabled

# Copy the configuration
blue "Copying $CONFIG_FILE to /etc/nginx/sites-available/latency.space..."
cp "$CONFIG_FILE" /etc/nginx/sites-available/latency.space

# Create symlink if it doesn't exist
if [ ! -f "/etc/nginx/sites-enabled/latency.space" ]; then
  blue "Creating symlink in sites-enabled..."
  ln -sf /etc/nginx/sites-available/latency.space /etc/nginx/sites-enabled/
fi

# Remove default if it exists
if [ -f "/etc/nginx/sites-enabled/default" ]; then
  blue "Removing default Nginx configuration..."
  rm -f /etc/nginx/sites-enabled/default
fi

# Test Nginx configuration
blue "Testing Nginx configuration..."
if nginx -t; then
  green "✅ Nginx configuration is valid"
  
  blue "Reloading Nginx..."
  systemctl reload nginx
  green "✅ Nginx reloaded successfully"
else
  red "❌ Nginx configuration test failed"
  exit 1
fi

green "✅ Nginx configuration installed successfully!"
echo ""
echo "Service endpoints:"
echo "- Main website: http://latency.space"
echo "- Status dashboard: http://status.latency.space"
echo "- Proxy services: http://mars.latency.space, etc."

if [ "$CONFIG_TYPE" == "minimal" ]; then
  yellow "Note: Minimal configuration is installed. This expects services on localhost ports."
  echo "      - Status service on localhost:3000"
  echo "      - Proxy service on localhost:8080"
else
  echo "This configuration expects Docker containers named:"
  echo "- proxy: Service with interplanetary latencies"
  echo "- status: Status dashboard"
fi