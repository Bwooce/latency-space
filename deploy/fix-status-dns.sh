#!/bin/bash
# Script to directly fix the status.latency.space DNS record
# This script bypasses Cloudflare proxy for the status subdomain

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }
yellow() { echo -e "\033[0;33m$1\033[0m"; }

blue "🌐 Fixing status.latency.space DNS record"

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

# Get server IP
SERVER_IP=$(curl -s ifconfig.me || curl -s icanhazip.com || curl -s ipecho.net/plain)
if [ -z "$SERVER_IP" ]; then
  red "Could not automatically detect server IP."
  read -p "Enter your server IP address: " SERVER_IP
  if [ -z "$SERVER_IP" ]; then
    red "No server IP provided. Exiting."
    exit 1
  fi
fi

blue "Using server IP: $SERVER_IP"

# Check for jq
if ! command -v jq &> /dev/null; then
  blue "Installing jq..."
  apt-get update && apt-get install -y jq
fi

# Get zone ID for latency.space
blue "Getting Cloudflare zone ID for latency.space..."
ZONE_RESPONSE=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones?name=latency.space" \
     -H "Authorization: Bearer $CF_API_TOKEN" \
     -H "Content-Type: application/json")

ZONE_ID=$(echo $ZONE_RESPONSE | jq -r '.result[0].id')
if [ -z "$ZONE_ID" ] || [ "$ZONE_ID" == "null" ]; then
  red "Could not find zone ID for latency.space."
  echo "API Response: $ZONE_RESPONSE"
  exit 1
fi

green "Found zone ID: $ZONE_ID"

# Get the current status.latency.space record
blue "Checking for existing status.latency.space record..."
RECORD_RESPONSE=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records?type=A&name=status.latency.space" \
     -H "Authorization: Bearer $CF_API_TOKEN" \
     -H "Content-Type: application/json")

echo "API Response details:"
echo "$RECORD_RESPONSE" | jq '.result_info'

RECORD_COUNT=$(echo "$RECORD_RESPONSE" | jq '.result | length')
if [ "$RECORD_COUNT" -eq "0" ]; then
  blue "No status.latency.space record found. Creating a new one..."
  
  # Create new record with proxied=false to bypass Cloudflare
  CREATE_RESPONSE=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records" \
     -H "Authorization: Bearer $CF_API_TOKEN" \
     -H "Content-Type: application/json" \
     --data "{\"type\":\"A\",\"name\":\"status\",\"content\":\"$SERVER_IP\",\"ttl\":1,\"proxied\":false}")
  
  CREATE_SUCCESS=$(echo "$CREATE_RESPONSE" | jq '.success')
  if [ "$CREATE_SUCCESS" == "true" ]; then
    green "✓ Successfully created DNS record for status.latency.space pointing directly to $SERVER_IP"
    echo "Created record details:"
    echo "$CREATE_RESPONSE" | jq '.result | {id, name, content, proxied}'
  else
    red "✗ Failed to create DNS record:"
    echo "$CREATE_RESPONSE" | jq '.errors'
    exit 1
  fi
else
  RECORD_ID=$(echo "$RECORD_RESPONSE" | jq -r '.result[0].id')
  RECORD_IP=$(echo "$RECORD_RESPONSE" | jq -r '.result[0].content')
  RECORD_PROXIED=$(echo "$RECORD_RESPONSE" | jq -r '.result[0].proxied')
  
  if [ "$RECORD_PROXIED" == "true" ]; then
    yellow "⚠️ Record is currently proxied through Cloudflare. This is causing the incorrect IP address."
    blue "Updating record to bypass Cloudflare proxy..."
    
    UPDATE_RESPONSE=$(curl -s -X PUT "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records/$RECORD_ID" \
       -H "Authorization: Bearer $CF_API_TOKEN" \
       -H "Content-Type: application/json" \
       --data "{\"type\":\"A\",\"name\":\"status\",\"content\":\"$SERVER_IP\",\"ttl\":1,\"proxied\":false}")
    
    UPDATE_SUCCESS=$(echo "$UPDATE_RESPONSE" | jq '.success')
    if [ "$UPDATE_SUCCESS" == "true" ]; then
      green "✓ Successfully updated DNS record for status.latency.space to point directly to $SERVER_IP"
      echo "Updated record details:"
      echo "$UPDATE_RESPONSE" | jq '.result | {id, name, content, proxied}'
    else
      red "✗ Failed to update DNS record:"
      echo "$UPDATE_RESPONSE" | jq '.errors'
      exit 1
    fi
  elif [ "$RECORD_IP" != "$SERVER_IP" ]; then
    blue "Updating record's IP address from $RECORD_IP to $SERVER_IP..."
    
    UPDATE_RESPONSE=$(curl -s -X PUT "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records/$RECORD_ID" \
       -H "Authorization: Bearer $CF_API_TOKEN" \
       -H "Content-Type: application/json" \
       --data "{\"type\":\"A\",\"name\":\"status\",\"content\":\"$SERVER_IP\",\"ttl\":1,\"proxied\":false}")
    
    UPDATE_SUCCESS=$(echo "$UPDATE_RESPONSE" | jq '.success')
    if [ "$UPDATE_SUCCESS" == "true" ]; then
      green "✓ Successfully updated DNS record for status.latency.space to $SERVER_IP"
      echo "Updated record details:"
      echo "$UPDATE_RESPONSE" | jq '.result | {id, name, content, proxied}'
    else
      red "✗ Failed to update DNS record:"
      echo "$UPDATE_RESPONSE" | jq '.errors'
      exit 1
    fi
  else
    green "✓ DNS record is already correctly configured to point directly to $SERVER_IP"
  fi
fi

# Verify the changes were applied
blue "Verifying DNS record update..."
VERIFY_RESPONSE=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records?type=A&name=status.latency.space" \
     -H "Authorization: Bearer $CF_API_TOKEN" \
     -H "Content-Type: application/json")

echo "Current record details:"
echo "$VERIFY_RESPONSE" | jq '.result[0] | {id, name, content, proxied}'

# Check if Cloudflare proxy is disabled
VERIFY_PROXIED=$(echo "$VERIFY_RESPONSE" | jq -r '.result[0].proxied')
if [ "$VERIFY_PROXIED" == "false" ]; then
  green "✓ Cloudflare proxy is correctly disabled for status.latency.space"
else
  red "✗ Cloudflare proxy is still enabled. This may take time to update."
fi

blue "Status.latency.space DNS changes have been applied!"
yellow "⚠️ DNS changes may take time to propagate (typically 5-30 minutes)"
echo "To check the current DNS resolution:"
echo "  dig status.latency.space"
echo ""
echo "To test the server directly:"
echo "  curl -H 'Host: status.latency.space' http://$SERVER_IP"