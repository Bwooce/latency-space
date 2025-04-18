#!/bin/bash
# Script to specifically fix the status.latency.space subdomain

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }
yellow() { echo -e "\033[0;33m$1\033[0m"; }

echo "====================================================================================="
blue "🛠️  STATUS.LATENCY.SPACE SUBDOMAIN FIX TOOL"
echo "====================================================================================="

# Check for required tools
check_command() {
  if ! command -v $1 &> /dev/null; then
    yellow "⚠️  $1 is not installed. Attempting to install..."
    if command -v apt-get &> /dev/null; then
      apt-get update && apt-get install -y $2 || {
        red "❌ Failed to install $1. Please install it manually."
        return 1
      }
    elif command -v yum &> /dev/null; then
      yum install -y $2 || {
        red "❌ Failed to install $1. Please install it manually."
        return 1
      }
    else
      red "❌ Could not install $1. Please install it manually."
      return 1
    fi
    green "✅ Successfully installed $1."
  fi
  return 0
}

check_command curl curl || exit 1
check_command jq jq || exit 1
check_command dig dnsutils || exit 1

# Server IP detection
get_server_ip() {
  local ip
  
  # First try environment variable
  if [ -n "$SERVER_IP" ]; then
    echo "$SERVER_IP"
    return
  fi
  
  # Try to get from file if it exists
  if [ -f "/opt/latency-space/server_ip.txt" ]; then
    ip=$(cat /opt/latency-space/server_ip.txt)
    if [ -n "$ip" ]; then
      echo "$ip"
      return
    fi
  fi
  
  # Try various IP detection services
  ip=$(curl -s ifconfig.me 2>/dev/null || 
       curl -s icanhazip.com 2>/dev/null || 
       curl -s ipecho.net/plain 2>/dev/null ||
       curl -s wtfismyip.com/text 2>/dev/null)
  
  if [ -n "$ip" ]; then
    echo "$ip"
    return
  fi
  
  # If all else fails, ask the user
  read -p "Enter your server's public IP address: " manual_ip
  echo "$manual_ip"
}

# Cloudflare token handling
get_cloudflare_token() {
  # First try environment variable
  if [ -n "$CF_API_TOKEN" ]; then
    echo "$CF_API_TOKEN"
    return
  fi
  
  # Try to load from secrets file
  for secret_file in "/opt/latency-space/secrets/cloudflare.env" "/opt/latency-space/.env" ".env" "secrets/cloudflare.env"; do
    if [ -f "$secret_file" ]; then
      blue "🔍 Found potential secrets file: $secret_file"
      source "$secret_file"
      if [ -n "$CF_API_TOKEN" ]; then
        echo "$CF_API_TOKEN"
        return
      fi
    fi
  done
  
  # If all else fails, ask the user
  read -p "Enter your Cloudflare API token: " manual_token
  echo "$manual_token"
}

# Main function to fix status.latency.space
fix_status_subdomain() {
  local server_ip=$1
  local cf_token=$2
  
  blue "🔍 Checking DNS status for status.latency.space..."
  local current_ip=$(dig +short status.latency.space)
  
  if [ "$current_ip" == "$server_ip" ]; then
    green "✅ status.latency.space already resolves to $server_ip"
    yellow "⚠️  If you're still having issues, it may be a propagation delay or Nginx configuration problem."
    echo "DNS propagation can take up to 24 hours, though typically just minutes with Cloudflare."
    return 0
  elif [ -n "$current_ip" ]; then
    yellow "⚠️  status.latency.space resolves to $current_ip, but should be $server_ip"
  else
    yellow "⚠️  status.latency.space does not resolve to any IP"
  fi
  
  blue "🔍 Getting Cloudflare zone ID for latency.space..."
  local zone_response=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones?name=latency.space" \
       -H "Authorization: Bearer $cf_token" \
       -H "Content-Type: application/json")
  
  local success=$(echo $zone_response | jq -r '.success')
  if [ "$success" != "true" ]; then
    red "❌ Failed to get zone ID. Cloudflare API error:"
    echo "$zone_response" | jq '.'
    return 1
  fi
  
  local zone_id=$(echo $zone_response | jq -r '.result[0].id')
  if [ -z "$zone_id" ] || [ "$zone_id" == "null" ]; then
    red "❌ Could not find zone ID for latency.space. Check your Cloudflare API token permissions."
    return 1
  fi
  
  green "✅ Found zone ID: $zone_id"
  
  # Check for existing record
  blue "🔍 Checking for existing status.latency.space record..."
  local record_response=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$zone_id/dns_records?type=A&name=status.latency.space" \
       -H "Authorization: Bearer $cf_token" \
       -H "Content-Type: application/json")
  
  local record_id=$(echo $record_response | jq -r '.result[0].id')
  local record_ip=$(echo $record_response | jq -r '.result[0].content')
  
  # Create or update the record
  if [ "$record_id" == "null" ] || [ -z "$record_id" ]; then
    blue "🆕 Creating new DNS record for status.latency.space..."
    local create_response=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones/$zone_id/dns_records" \
       -H "Authorization: Bearer $cf_token" \
       -H "Content-Type: application/json" \
       --data "{\"type\":\"A\",\"name\":\"status\",\"content\":\"$server_ip\",\"ttl\":1,\"proxied\":true}")
    
    if echo "$create_response" | jq -e '.success' > /dev/null; then
      green "✅ Successfully created DNS record for status.latency.space!"
      echo "New record details:"
      echo "$create_response" | jq '.result | {id, name, content, proxied, ttl}'
    else
      red "❌ Failed to create DNS record:"
      echo "$create_response" | jq '.'
      
      yellow "⚠️  Attempting alternative approach..."
      # Try creating with different parameters
      local create_response=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones/$zone_id/dns_records" \
         -H "Authorization: Bearer $cf_token" \
         -H "Content-Type: application/json" \
         --data "{\"type\":\"A\",\"name\":\"status.latency.space\",\"content\":\"$server_ip\",\"ttl\":1,\"proxied\":true}")
      
      if echo "$create_response" | jq -e '.success' > /dev/null; then
        green "✅ Successfully created DNS record with alternative approach!"
      else
        red "❌ Alternative approach also failed."
        
        blue "🔍 List of existing DNS records for latency.space:"
        curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$zone_id/dns_records" \
          -H "Authorization: Bearer $cf_token" \
          -H "Content-Type: application/json" | jq '.result[] | {id, name, type, content}'
        
        return 1
      fi
    fi
  else
    if [ "$record_ip" != "$server_ip" ]; then
      blue "🔄 Updating existing record for status.latency.space from $record_ip to $server_ip..."
      local update_response=$(curl -s -X PUT "https://api.cloudflare.com/client/v4/zones/$zone_id/dns_records/$record_id" \
         -H "Authorization: Bearer $cf_token" \
         -H "Content-Type: application/json" \
         --data "{\"type\":\"A\",\"name\":\"status\",\"content\":\"$server_ip\",\"ttl\":1,\"proxied\":true}")
      
      if echo "$update_response" | jq -e '.success' > /dev/null; then
        green "✅ Successfully updated DNS record for status.latency.space!"
      else
        red "❌ Failed to update DNS record:"
        echo "$update_response" | jq '.'
        return 1
      fi
    else
      green "✅ DNS record already exists with correct IP: $record_ip"
    fi
  fi
  
  # Verify DNS record creation/update was successful
  blue "🔍 Verifying DNS record update..."
  local verify_response=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$zone_id/dns_records?type=A&name=status.latency.space" \
       -H "Authorization: Bearer $cf_token" \
       -H "Content-Type: application/json")
  
  echo "Verification from Cloudflare API:"
  echo "$verify_response" | jq '.result[] | {id, name, content, proxied, ttl}'
  
  # Provide instructions for verification
  green "✅ DNS record operation completed successfully!"
  blue "🔍 To verify that the change is propagating, you can run:"
  echo "  dig status.latency.space"
  echo ""
  yellow "⚠️  DNS changes can take time to propagate worldwide (up to 24 hours)"
  echo "   However, with Cloudflare's proxy enabled, it's typically much faster (minutes)."
  echo ""
  
  # Verify the Nginx configuration
  verify_nginx_configuration
  return 0
}

# Function to verify Nginx configuration
verify_nginx_configuration() {
  blue "🔍 Checking Nginx configuration..."
  
  # Check if Nginx is installed
  if ! command -v nginx &> /dev/null; then
    yellow "⚠️  Nginx command not found. Skipping Nginx verification."
    return
  fi
  
  # Check for configuration file
  local nginx_conf_found=false
  local nginx_conf_files=(
    "/etc/nginx/sites-enabled/latency.space"
    "/etc/nginx/sites-available/latency.space"
    "/etc/nginx/conf.d/latency.space.conf"
  )
  
  for conf_file in "${nginx_conf_files[@]}"; do
    if [ -f "$conf_file" ]; then
      nginx_conf_found=true
      blue "🔍 Found Nginx configuration file: $conf_file"
      
      # Check if status subdomain is configured
      if grep -q "server_name.*status\.latency\.space" "$conf_file"; then
        green "✅ Status subdomain is configured in Nginx"
      else
        red "❌ Status subdomain not found in Nginx configuration"
        yellow "⚠️  Your Nginx configuration may need to be updated to handle status.latency.space"
      fi
      
      # Check for proxy configuration
      if grep -q "proxy_pass.*http://status:3000" "$conf_file"; then
        green "✅ Proxy pass to status container is configured correctly"
      else
        yellow "⚠️  No proper proxy_pass for status container found"
        echo "   Your Nginx should have a configuration like:"
        echo '   server {'
        echo '       listen 80;'
        echo '       server_name status.latency.space;'
        echo '       location / {'
        echo '           proxy_pass http://status:3000;'
        echo '           proxy_http_version 1.1;'
        echo '           proxy_set_header Upgrade $http_upgrade;'
        echo '           proxy_set_header Connection "upgrade";'
        echo '           proxy_set_header Host $host;'
        echo '       }'
        echo '   }'
      fi
      
      break
    fi
  done
  
  if [ "$nginx_conf_found" = false ]; then
    yellow "⚠️  No Nginx configuration file found for latency.space"
    echo "   Please ensure Nginx is properly configured for status.latency.space"
  fi
  
  # Check Docker containers
  blue "🔍 Checking Docker status container..."
  if command -v docker &> /dev/null; then
    if docker ps | grep -q "status"; then
      green "✅ Status container is running"
    else
      red "❌ Status container is not running"
      yellow "⚠️  Try starting it with: docker compose up -d status"
    fi
  else
    yellow "⚠️  Docker command not found. Skipping container check."
  fi
}

# Main script execution
echo "🚀 Starting status.latency.space DNS fix"

# Get server IP and Cloudflare token
SERVER_IP=$(get_server_ip)
if [ -z "$SERVER_IP" ]; then
  red "❌ Failed to determine server IP address. Exiting."
  exit 1
fi

CF_API_TOKEN=$(get_cloudflare_token)
if [ -z "$CF_API_TOKEN" ]; then
  red "❌ Failed to get Cloudflare API token. Exiting."
  exit 1
fi

blue "🌐 Using server IP: $SERVER_IP"
blue "🔑 Using Cloudflare API token: ${CF_API_TOKEN:0:4}...${CF_API_TOKEN: -4}"

# Run the fix function
fix_status_subdomain "$SERVER_IP" "$CF_API_TOKEN"
exit_code=$?

if [ $exit_code -eq 0 ]; then
  green "✅ status.latency.space DNS fix completed successfully!"
else
  red "❌ status.latency.space DNS fix failed with errors."
fi

echo "====================================================================================="
exit $exit_code