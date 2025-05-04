#!/bin/bash
# Script to fix DNS records for all latency.space subdomains
# This script bypasses Cloudflare proxy for all functional subdomains 
# (planets, moons, spacecraft, status) while keeping it for the main domain

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }
yellow() { echo -e "\033[0;33m$1\033[0m"; }
DIVIDER="----------------------------------------"

blue "🌐 Fixing latency.space DNS records"
echo $DIVIDER

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

# Get server IP - explicitly requesting IPv4
SERVER_IP=$(curl -s -4 ifconfig.me || curl -s -4 icanhazip.com || curl -s -4 ipecho.net/plain)
if [ -z "$SERVER_IP" ]; then
  red "Could not automatically detect IPv4 address."
  read -p "Enter your server IPv4 address: " SERVER_IP
  if [ -z "$SERVER_IP" ]; then
    red "No IPv4 address provided. Exiting."
    exit 1
  fi
fi

# Validate that we have an IPv4 address
if ! echo "$SERVER_IP" | grep -E "^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$" > /dev/null; then
  red "Error: '$SERVER_IP' does not appear to be a valid IPv4 address."
  red "This script only supports IPv4 addresses for DNS A records."
  exit 1
fi

blue "Using server IP: $SERVER_IP"
echo $DIVIDER

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
echo $DIVIDER

# Get all DNS records for latency.space
blue "Fetching all DNS records for latency.space..."
ALL_RECORDS=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records?per_page=100" \
     -H "Authorization: Bearer $CF_API_TOKEN" \
     -H "Content-Type: application/json")

RECORD_COUNT=$(echo "$ALL_RECORDS" | jq '.result | length')
green "Found $RECORD_COUNT DNS records"

# Process all A records
blue "Processing DNS records..."
echo "$ALL_RECORDS" | jq -c '.result[] | select(.type=="A")' | while read -r record; do
  NAME=$(echo "$record" | jq -r '.name')
  CONTENT=$(echo "$record" | jq -r '.content')
  PROXIED=$(echo "$record" | jq -r '.proxied')
  RECORD_ID=$(echo "$record" | jq -r '.id')
  
  # Check if this is the main domain or a subdomain
  if [[ "$NAME" == "latency.space" || "$NAME" == "www.latency.space" ]]; then
    # Main domain should be proxied
    if [[ "$PROXIED" == "false" ]]; then
      yellow "⚠️  Main domain $NAME is not proxied. Enabling Cloudflare proxy..."
      UPDATE_RESPONSE=$(curl -s -X PUT "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records/$RECORD_ID" \
         -H "Authorization: Bearer $CF_API_TOKEN" \
         -H "Content-Type: application/json" \
         --data "{\"type\":\"A\",\"name\":\"$NAME\",\"content\":\"$CONTENT\",\"ttl\":1,\"proxied\":true}")
      
      if [[ $(echo "$UPDATE_RESPONSE" | jq '.success') == "true" ]]; then
        green "✓ Successfully enabled Cloudflare proxy for $NAME"
      else
        red "✗ Failed to update $NAME:"
        echo "$UPDATE_RESPONSE" | jq '.errors'
      fi
    else
      green "✓ Main domain $NAME is correctly proxied through Cloudflare"
    fi
  else
    # Subdomain should NOT be proxied for interplanetary latency
    NEEDS_UPDATE=false
    UPDATE_IP=false
    
    if [[ "$PROXIED" == "true" ]]; then
      yellow "⚠️  Subdomain $NAME is proxied through Cloudflare. Disabling proxy..."
      NEEDS_UPDATE=true
    fi
    
    if [[ "$CONTENT" != "$SERVER_IP" ]]; then
      yellow "⚠️  Subdomain $NAME has incorrect IP: $CONTENT. Updating to $SERVER_IP..."
      UPDATE_IP=true
      NEEDS_UPDATE=true
    fi
    
    if [[ "$NEEDS_UPDATE" == "true" ]]; then
      UPDATE_RESPONSE=$(curl -s -X PUT "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records/$RECORD_ID" \
         -H "Authorization: Bearer $CF_API_TOKEN" \
         -H "Content-Type: application/json" \
         --data "{\"type\":\"A\",\"name\":\"$NAME\",\"content\":\"$SERVER_IP\",\"ttl\":1,\"proxied\":false}")
      
      if [[ $(echo "$UPDATE_RESPONSE" | jq '.success') == "true" ]]; then
        if [[ "$PROXIED" == "true" && "$UPDATE_IP" == "true" ]]; then
          green "✓ Successfully disabled Cloudflare proxy and updated IP for $NAME"
        elif [[ "$PROXIED" == "true" ]]; then
          green "✓ Successfully disabled Cloudflare proxy for $NAME"
        else
          green "✓ Successfully updated IP for $NAME to $SERVER_IP"
        fi
      else
        red "✗ Failed to update $NAME:"
        echo "$UPDATE_RESPONSE" | jq '.errors'
      fi
    else
      green "✓ Subdomain $NAME is correctly configured (not proxied, correct IP)"
    fi
  fi
  
  echo $DIVIDER
done

# Look for important subdomains that might be missing
important_subdomains=("status" "mars" "jupiter" "saturn" "earth" "venus" "mercury" "uranus" "neptune" "pluto" "moon.earth" "voyager1" "voyager2")

for subdomain in "${important_subdomains[@]}"; do
  full_name="${subdomain}.latency.space"
  
  # Check if record exists
  if ! echo "$ALL_RECORDS" | jq -e ".result[] | select(.name==\"$full_name\")" > /dev/null; then
    yellow "⚠️  Important subdomain $full_name is missing. Creating..."
    
    CREATE_RESPONSE=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records" \
       -H "Authorization: Bearer $CF_API_TOKEN" \
       -H "Content-Type: application/json" \
       --data "{\"type\":\"A\",\"name\":\"$subdomain\",\"content\":\"$SERVER_IP\",\"ttl\":1,\"proxied\":false}")
    
    if [[ $(echo "$CREATE_RESPONSE" | jq '.success') == "true" ]]; then
      green "✓ Successfully created record for $full_name pointing directly to $SERVER_IP"
    else
      red "✗ Failed to create record for $full_name:"
      echo "$CREATE_RESPONSE" | jq '.errors'
    fi
    
    echo $DIVIDER
  fi
done

blue "All DNS changes have been applied!"
yellow "⚠️ DNS changes may take time to propagate (typically 5-30 minutes)"
echo "To check the current DNS resolution:"
echo "  dig +short mars.latency.space"
echo "  dig +short status.latency.space"
echo ""
echo "To verify the changes took effect, the DNS should resolve directly to your server IP: $SERVER_IP"
echo "To test SOCKS proxy functionality:"
echo "  curl --socks5 mars.latency.space:1080 https://example.com"

# Automated SSL certificate request with certbot
echo $DIVIDER
blue "🔒 Checking SSL certificate configuration"

# Ensure certbot is installed
if ! command -v certbot &> /dev/null; then
  yellow "⚠️ Certbot not found, installing..."
  
  if command -v apt-get &> /dev/null; then
    apt-get update
    apt-get install -y certbot python3-certbot-nginx
  elif command -v dnf &> /dev/null; then
    dnf install -y certbot python3-certbot-nginx
  else
    red "❌ Package manager not found. Please install certbot manually."
    exit 1
  fi
  
  green "✅ Certbot installed successfully"
fi

# Check if existing certificates are present
SSL_DIR="/etc/letsencrypt/live/latency.space"
if [ -d "$SSL_DIR" ]; then
  blue "Checking existing SSL certificates..."
  
  # Check certificate expiration
  CERT_DATE=$(openssl x509 -enddate -noout -in $SSL_DIR/fullchain.pem | cut -d= -f2)
  CERT_EXPIRY=$(date -d "$CERT_DATE" +%s)
  NOW=$(date +%s)
  DAYS_REMAINING=$(( ($CERT_EXPIRY - $NOW) / 86400 ))
  
  if [ $DAYS_REMAINING -lt 30 ]; then
    yellow "⚠️ SSL certificate expires in $DAYS_REMAINING days, attempting renewal"
    
    # Renew certificates
    certbot renew --quiet
    if [ $? -eq 0 ]; then
      green "✅ Successfully renewed SSL certificates"
    else
      red "❌ Failed to renew SSL certificates"
    fi
  else
    green "✅ SSL certificates are valid for $DAYS_REMAINING more days"
  fi
else
  yellow "⚠️ SSL certificates not found, will attempt to obtain them"
  
  # Create an array of domains from the important_subdomains
  DOMAIN_ARGS=("-d" "latency.space" "-d" "www.latency.space")
  
  # Add all important subdomains to the certificate request
  for subdomain in "${important_subdomains[@]}"; do
    DOMAIN_ARGS+=("-d" "${subdomain}.latency.space")
  done
  
  # Run certbot in non-interactive mode (suitable for automated scripts)
  blue "Requesting SSL certificates for latency.space and all subdomains..."
  
  # Check if we're running interactively
  if [ -t 0 ]; then
    # Interactive run - ask user first
    read -p "Do you want to request SSL certificates for all domains now? (y/n): " run_certbot
    if [[ "$run_certbot" == "y" ]]; then
      certbot --nginx "${DOMAIN_ARGS[@]}" --redirect
    else
      yellow "⚠️ SSL certificate request skipped."
      echo "To request certificates manually, run:"
      echo "certbot --nginx ${DOMAIN_ARGS[@]} --redirect"
    fi
  else
    # Non-interactive run - determine if webserver is ready
    if systemctl is-active --quiet nginx; then
      # Check if port 80 is available (required for HTTP-01 validation)
      if curl -s http://localhost:80 &>/dev/null; then
        certbot --nginx "${DOMAIN_ARGS[@]}" --redirect --non-interactive --agree-tos --email admin@latency.space
        
        if [ $? -eq 0 ]; then
          green "✅ Successfully obtained SSL certificates for all domains"
        else
          red "❌ Failed to obtain SSL certificates"
          yellow "⚠️ You'll need to run certbot manually to obtain certificates"
        fi
      else
        yellow "⚠️ Nginx is running but port 80 might not be accessible"
        yellow "⚠️ Skipping automatic certificate request"
      fi
    else
      yellow "⚠️ Nginx is not running, skipping automatic certificate request"
      yellow "⚠️ Please ensure Nginx is properly configured before requesting certificates"
    fi
  fi
fi

echo $DIVIDER
green "✅ DNS and SSL configuration complete!"