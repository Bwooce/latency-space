#!/bin/bash
# Script to fix DNS issues for status.latency.space

# Check if required environment variables are set
if [ -z "$CF_API_TOKEN" ]; then
  echo "Error: CF_API_TOKEN environment variable not set"
  echo "Usage: CF_API_TOKEN=your_token SERVER_IP=your_server_ip ./fix-dns.sh"
  exit 1
fi

if [ -z "$SERVER_IP" ]; then
  echo "Error: SERVER_IP environment variable not set"
  echo "Usage: CF_API_TOKEN=your_token SERVER_IP=your_server_ip ./fix-dns.sh"
  exit 1
fi

echo "Checking DNS record for status.latency.space..."

# Get zone ID for latency.space
ZONE_ID=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones?name=latency.space" \
     -H "Authorization: Bearer $CF_API_TOKEN" \
     -H "Content-Type: application/json" | jq -r '.result[0].id')

if [ -z "$ZONE_ID" ] || [ "$ZONE_ID" == "null" ]; then
  echo "Error: Could not find zone ID for latency.space"
  exit 1
fi

echo "Found zone ID: $ZONE_ID"

# Check if status.latency.space record exists
RECORD_INFO=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records?type=A&name=status.latency.space" \
     -H "Authorization: Bearer $CF_API_TOKEN" \
     -H "Content-Type: application/json")

RECORD_ID=$(echo $RECORD_INFO | jq -r '.result[0].id')
RECORD_IP=$(echo $RECORD_INFO | jq -r '.result[0].content')

if [ -z "$RECORD_ID" ] || [ "$RECORD_ID" == "null" ]; then
  echo "Record for status.latency.space not found. Creating..."
  
  # Create new record
  CREATE_RESPONSE=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records" \
     -H "Authorization: Bearer $CF_API_TOKEN" \
     -H "Content-Type: application/json" \
     --data "{\"type\":\"A\",\"name\":\"status\",\"content\":\"$SERVER_IP\",\"ttl\":1,\"proxied\":true}")
     
  if echo "$CREATE_RESPONSE" | jq -e '.success' > /dev/null; then
    echo "Successfully created DNS record for status.latency.space pointing to $SERVER_IP"
  else
    echo "Error creating DNS record:"
    echo "$CREATE_RESPONSE" | jq '.'
    exit 1
  fi
else
  echo "Record found for status.latency.space: $RECORD_ID (current IP: $RECORD_IP)"
  
  if [ "$RECORD_IP" != "$SERVER_IP" ]; then
    echo "IP address mismatch. Updating record to point to $SERVER_IP..."
    
    # Update existing record
    UPDATE_RESPONSE=$(curl -s -X PUT "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records/$RECORD_ID" \
       -H "Authorization: Bearer $CF_API_TOKEN" \
       -H "Content-Type: application/json" \
       --data "{\"type\":\"A\",\"name\":\"status\",\"content\":\"$SERVER_IP\",\"ttl\":1,\"proxied\":true}")
       
    if echo "$UPDATE_RESPONSE" | jq -e '.success' > /dev/null; then
      echo "Successfully updated DNS record for status.latency.space to $SERVER_IP"
    else
      echo "Error updating DNS record:"
      echo "$UPDATE_RESPONSE" | jq '.'
      exit 1
    fi
  else
    echo "Record is already correctly pointing to $SERVER_IP"
  fi
fi

echo ""
echo "Verifying DNS resolution..."
echo "Current local resolution:"
dig +short status.latency.space

echo ""
echo "To use this script:"
echo "1. Set your Cloudflare API token: export CF_API_TOKEN=your_token"
echo "2. Set your server IP: export SERVER_IP=your_server_ip"
echo "3. Run the script: ./fix-dns.sh"
echo ""
echo "Note: DNS changes may take up to 24 hours to fully propagate, though usually much faster."