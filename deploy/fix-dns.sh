#!/bin/bash
# Script to fix DNS issues for status.latency.space

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }

blue "🔍 Checking status.latency.space DNS record..."

# Check if jq is installed
if ! command -v jq &> /dev/null; then
  red "Error: jq is not installed. Installing..."
  apt-get update && apt-get install -y jq || {
    red "Failed to install jq. Please install it manually and run this script again."
    exit 1
  }
fi

# Check if required environment variables are set
if [ -z "$CF_API_TOKEN" ]; then
  red "Error: CF_API_TOKEN environment variable not set"
  echo "  Usage: CF_API_TOKEN=your_token SERVER_IP=your_server_ip ./fix-dns.sh"
  
  # Check if secrets file exists
  if [ -f "/opt/latency-space/secrets/cloudflare.env" ]; then
    blue "Found Cloudflare secrets file. Loading..."
    source /opt/latency-space/secrets/cloudflare.env
  else
    exit 1
  fi
fi

if [ -z "$SERVER_IP" ]; then
  red "Error: SERVER_IP environment variable not set"
  echo "  Usage: CF_API_TOKEN=your_token SERVER_IP=your_server_ip ./fix-dns.sh"
  
  # Try to automatically determine server IP
  SERVER_IP=$(curl -s ifconfig.me || curl -s icanhazip.com || curl -s ipecho.net/plain)
  if [ -n "$SERVER_IP" ]; then
    blue "Automatically detected server IP: $SERVER_IP"
  else
    red "Could not automatically detect server IP. Please specify SERVER_IP manually."
    exit 1
  fi
fi

# Diagnostic check for current DNS resolution
blue "📊 Current DNS status for latency.space domains:"
echo "Main domain (latency.space):"
dig +short latency.space

echo "Status subdomain (status.latency.space):"
dig +short status.latency.space

echo "Mars subdomain (mars.latency.space):"
dig +short mars.latency.space

# Get zone ID for latency.space
blue "🔍 Getting Cloudflare zone ID for latency.space..."
ZONE_ID=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones?name=latency.space" \
     -H "Authorization: Bearer $CF_API_TOKEN" \
     -H "Content-Type: application/json" | jq -r '.result[0].id')

if [ -z "$ZONE_ID" ] || [ "$ZONE_ID" == "null" ]; then
  red "❌ Error: Could not find zone ID for latency.space"
  red "Please check your Cloudflare API token has sufficient permissions"
  exit 1
fi

green "✅ Found zone ID: $ZONE_ID"

# Check if status.latency.space record exists
blue "🔍 Checking for existing status.latency.space record..."
RECORD_INFO=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records?type=A&name=status.latency.space" \
     -H "Authorization: Bearer $CF_API_TOKEN" \
     -H "Content-Type: application/json")

# Display debugging info
echo "API Response details:"
echo "$RECORD_INFO" | jq '.result_info'

RECORD_ID=$(echo $RECORD_INFO | jq -r '.result[0].id')
RECORD_IP=$(echo $RECORD_INFO | jq -r '.result[0].content')

if [ -z "$RECORD_ID" ] || [ "$RECORD_ID" == "null" ]; then
  blue "🆕 Record for status.latency.space not found. Creating..."
  
  # Create new record
  CREATE_RESPONSE=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records" \
     -H "Authorization: Bearer $CF_API_TOKEN" \
     -H "Content-Type: application/json" \
     --data "{\"type\":\"A\",\"name\":\"status\",\"content\":\"$SERVER_IP\",\"ttl\":1,\"proxied\":true}")
     
  if echo "$CREATE_RESPONSE" | jq -e '.success' > /dev/null; then
    green "✅ Successfully created DNS record for status.latency.space pointing to $SERVER_IP"
    echo "Created record details:"
    echo "$CREATE_RESPONSE" | jq '.result | {id, name, content, proxied}'
  else
    red "❌ Error creating DNS record:"
    echo "$CREATE_RESPONSE" | jq '.'
    
    blue "🔍 Checking all DNS records for latency.space domain..."
    ALL_RECORDS=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records" \
      -H "Authorization: Bearer $CF_API_TOKEN" \
      -H "Content-Type: application/json")
    
    echo "All DNS records:"
    echo "$ALL_RECORDS" | jq '.result[] | {id, name, type, content}'
    exit 1
  fi
else
  green "✅ Record found for status.latency.space: $RECORD_ID (current IP: $RECORD_IP)"
  
  if [ "$RECORD_IP" != "$SERVER_IP" ]; then
    blue "🔄 IP address mismatch. Updating record to point to $SERVER_IP..."
    
    # Update existing record
    UPDATE_RESPONSE=$(curl -s -X PUT "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records/$RECORD_ID" \
       -H "Authorization: Bearer $CF_API_TOKEN" \
       -H "Content-Type: application/json" \
       --data "{\"type\":\"A\",\"name\":\"status\",\"content\":\"$SERVER_IP\",\"ttl\":1,\"proxied\":true}")
       
    if echo "$UPDATE_RESPONSE" | jq -e '.success' > /dev/null; then
      green "✅ Successfully updated DNS record for status.latency.space to $SERVER_IP"
      echo "Updated record details:"
      echo "$UPDATE_RESPONSE" | jq '.result | {id, name, content, proxied}'
    else
      red "❌ Error updating DNS record:"
      echo "$UPDATE_RESPONSE" | jq '.'
      exit 1
    fi
  else
    green "✅ Record is already correctly pointing to $SERVER_IP"
  fi
fi

# Verify DNS changes are being applied
blue "🔍 Verifying DNS changes are being applied..."
blue "Note: DNS changes may take time to propagate globally"

# Check Cloudflare DNS records after changes
VERIFY_RECORDS=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records?type=A&name=status.latency.space" \
  -H "Authorization: Bearer $CF_API_TOKEN" \
  -H "Content-Type: application/json")

echo "Verification results from Cloudflare API:"
echo "$VERIFY_RECORDS" | jq '.result[] | {id, name, content, proxied}'

echo ""
blue "📊 Local DNS resolution status:"
echo "Main domain (latency.space):"
dig +short latency.space

echo "Status subdomain (status.latency.space):"
dig +short status.latency.space

# Curl check for status.latency.space
blue "🔍 Attempting HTTP request to status.latency.space..."
STATUS_RESPONSE=$(curl -s -I -m 5 http://status.latency.space | head -1)
if [ -n "$STATUS_RESPONSE" ]; then
  green "✅ HTTP response from status.latency.space: $STATUS_RESPONSE"
else
  red "❌ No HTTP response from status.latency.space"
fi

# Check if Nginx is properly configured
blue "🔍 Checking Nginx configuration for status.latency.space..."
if grep -q "server_name status.latency.space" /etc/nginx/sites-enabled/* 2>/dev/null; then
  green "✅ Nginx is configured for status.latency.space"
else
  red "❌ No Nginx configuration found for status.latency.space"
  echo "Nginx site configurations:"
  ls -la /etc/nginx/sites-enabled/
fi

# Check Docker status container
blue "🔍 Checking status container..."
if docker ps | grep -q status; then
  green "✅ Status container is running"
  echo "Container details:"
  docker ps | grep status
else
  red "❌ Status container is not running"
  echo "All running containers:"
  docker ps
fi

green "✅ DNS check completed. If status.latency.space is still not resolving:"
echo "  1. Confirm your Cloudflare API token has sufficient permissions"
echo "  2. Wait for DNS propagation (can take up to 24 hours)"
echo "  3. Check if your server IP ($SERVER_IP) is correct"
echo "  4. Verify Nginx is properly routing requests to the status container"
echo "  5. Ensure your firewall allows incoming traffic on port 80/443"