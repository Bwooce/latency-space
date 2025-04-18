#!/bin/bash
# Debug script to investigate why _debug pages aren't working
# This script collects Nginx and proxy logs, tests connectivity and configuration

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }
yellow() { echo -e "\033[0;33m$1\033[0m"; }
DIVIDER="----------------------------------------"

# Check if script is run as root
if [ "$EUID" -ne 0 ]; then
  yellow "⚠️  Running without root privileges. Some tests may fail."
  echo "For complete results, run with: sudo $0"
fi

# Create output directory
OUTPUT_DIR="/tmp/latency-debug"
mkdir -p "$OUTPUT_DIR"
LOG_FILE="$OUTPUT_DIR/debug-logs.txt"

# Start logging
echo "Debug script started at $(date)" | tee "$LOG_FILE"
echo $DIVIDER | tee -a "$LOG_FILE"

# Check Nginx configuration
blue "Checking Nginx configuration..." | tee -a "$LOG_FILE"
if [ -f "/etc/nginx/sites-enabled/latency.space" ]; then
  echo "Nginx configuration exists at /etc/nginx/sites-enabled/latency.space" | tee -a "$LOG_FILE"
  
  # Check for _debug location blocks
  if grep -q "location.*_debug" /etc/nginx/sites-enabled/latency.space; then
    echo "Found _debug location block in Nginx config:" | tee -a "$LOG_FILE"
    grep -A 15 "location.*_debug" /etc/nginx/sites-enabled/latency.space | tee -a "$LOG_FILE"
  else
    red "❌ No _debug location block found in Nginx config" | tee -a "$LOG_FILE"
  fi
  
  # Check server blocks
  echo -e "\nServer blocks in Nginx config:" | tee -a "$LOG_FILE"
  grep -n "server {" /etc/nginx/sites-enabled/latency.space | tee -a "$LOG_FILE"
  
  # Save full Nginx configuration
  echo -e "\nSaving full Nginx configuration to $OUTPUT_DIR/nginx-config.txt" | tee -a "$LOG_FILE"
  cat /etc/nginx/sites-enabled/latency.space > "$OUTPUT_DIR/nginx-config.txt"
else
  red "❌ Nginx configuration not found at /etc/nginx/sites-enabled/latency.space" | tee -a "$LOG_FILE"
fi

echo $DIVIDER | tee -a "$LOG_FILE"

# Check Nginx error logs
blue "Checking Nginx error logs..." | tee -a "$LOG_FILE"
if [ -f "/var/log/nginx/error.log" ]; then
  echo "Last 20 lines of Nginx error log:" | tee -a "$LOG_FILE"
  tail -n 20 /var/log/nginx/error.log | tee -a "$LOG_FILE"
  
  # Check for debug endpoint in access logs
  echo -e "\nSearching for _debug requests in access.log:" | tee -a "$LOG_FILE"
  grep "_debug" /var/log/nginx/access.log | tail -n 10 | tee -a "$LOG_FILE" || echo "No _debug requests found in access log" | tee -a "$LOG_FILE"
else
  red "❌ Nginx error log not found at /var/log/nginx/error.log" | tee -a "$LOG_FILE"
fi

echo $DIVIDER | tee -a "$LOG_FILE"

# Check Docker containers
blue "Checking Docker containers..." | tee -a "$LOG_FILE"
docker ps | tee -a "$LOG_FILE"

# Check proxy container
blue "Checking proxy container..." | tee -a "$LOG_FILE"
PROXY_CONTAINER=$(docker ps -q -f name=proxy)
if [ -n "$PROXY_CONTAINER" ]; then
  echo "Proxy container is running with ID: $PROXY_CONTAINER" | tee -a "$LOG_FILE"
  
  # Check proxy logs
  echo -e "\nLast 20 lines of proxy container logs:" | tee -a "$LOG_FILE"
  docker logs $PROXY_CONTAINER --tail 20 | tee -a "$LOG_FILE"
  
  # Check if proxy container has _debug endpoints
  echo -e "\nChecking if proxy container has _debug handlers:" | tee -a "$LOG_FILE"
  docker exec $PROXY_CONTAINER find /app -type f -name "*.go" -exec grep -l "_debug" {} \; | tee -a "$LOG_FILE" || echo "No _debug handlers found in Go files" | tee -a "$LOG_FILE"
  
  # Internal curl test (if available)
  echo -e "\nTesting _debug endpoint inside proxy container:" | tee -a "$LOG_FILE"
  if docker exec $PROXY_CONTAINER which curl &>/dev/null; then
    docker exec $PROXY_CONTAINER curl -v http://localhost/_debug/metrics 2>&1 | tee -a "$LOG_FILE"
  else
    echo "curl not available in container, can't test internally" | tee -a "$LOG_FILE"
  fi
else
  red "❌ Proxy container is not running" | tee -a "$LOG_FILE"
fi

echo $DIVIDER | tee -a "$LOG_FILE"

# Test external connectivity
blue "Testing external connectivity..." | tee -a "$LOG_FILE"

# Test main domain
echo "Testing main site (latency.space):" | tee -a "$LOG_FILE"
curl -s -I -m 5 http://latency.space | head -1 | tee -a "$LOG_FILE" || echo "Failed to connect" | tee -a "$LOG_FILE"

# Test _debug endpoints
echo -e "\nTesting _debug endpoint with verbose output:" | tee -a "$LOG_FILE"
curl -v http://latency.space/_debug/metrics 2>&1 | tee -a "$LOG_FILE"

# Test with direct IP
echo -e "\nTesting _debug endpoint with direct IP and Host header:" | tee -a "$LOG_FILE"
SERVER_IP=$(curl -s -4 ifconfig.me || curl -s -4 icanhazip.com || hostname -I | awk '{print $1}')
curl -v -H "Host: latency.space" http://$SERVER_IP/_debug/metrics 2>&1 | tee -a "$LOG_FILE"

# Test endpoint from inside the proxy container directly
echo -e "\nTesting _debug endpoint inside proxy container (if curl available):" | tee -a "$LOG_FILE"
if [ -n "$PROXY_CONTAINER" ] && docker exec $PROXY_CONTAINER which curl &>/dev/null; then
  docker exec $PROXY_CONTAINER curl -v http://localhost/_debug/metrics 2>&1 | tee -a "$LOG_FILE"
else
  echo "Cannot test inside proxy container (container not running or curl not available)" | tee -a "$LOG_FILE"
fi

echo $DIVIDER | tee -a "$LOG_FILE"

# Check proxy container code
blue "Analyzing proxy container code..." | tee -a "$LOG_FILE"
if [ -d "/opt/latency-space/proxy" ]; then
  echo "Found proxy code directory" | tee -a "$LOG_FILE"
  
  # Look for debug handlers in Go code
  echo -e "\nSearching for _debug handlers in Go code:" | tee -a "$LOG_FILE"
  grep -r "_debug" /opt/latency-space/proxy --include="*.go" | tee -a "$LOG_FILE" || echo "No _debug handlers found in code" | tee -a "$LOG_FILE"
  
  # Check main.go for route setup
  echo -e "\nChecking main.go for route setup:" | tee -a "$LOG_FILE"
  cat /opt/latency-space/proxy/src/main.go 2>/dev/null | grep -A 20 "func main" | tee -a "$LOG_FILE" || echo "Could not read main.go" | tee -a "$LOG_FILE"
else
  echo "Proxy code directory not found at /opt/latency-space/proxy" | tee -a "$LOG_FILE"
fi

echo $DIVIDER | tee -a "$LOG_FILE"

# Try installing tcpdump and running it
blue "Attempting to capture traffic with tcpdump (if available)..." | tee -a "$LOG_FILE"
if which tcpdump &>/dev/null; then
  echo "tcpdump is available, capturing traffic..." | tee -a "$LOG_FILE"
  
  # Run tcpdump in background
  OUTPUT_PCAP="$OUTPUT_DIR/debug-traffic.pcap"
  tcpdump -i any -c 100 -w "$OUTPUT_PCAP" port 80 or port 8080 or port 3000 &
  TCPDUMP_PID=$!
  
  # Make a test request while capturing
  echo "Making test request while capturing traffic..." | tee -a "$LOG_FILE"
  curl -s http://latency.space/_debug/metrics > /dev/null
  
  # Wait for capture to complete
  sleep 5
  kill $TCPDUMP_PID 2>/dev/null
  
  echo "Traffic capture saved to $OUTPUT_PCAP" | tee -a "$LOG_FILE"
  
  # Show summary of captured packets
  echo "Packet summary:" | tee -a "$LOG_FILE"
  tcpdump -r "$OUTPUT_PCAP" -n | head -10 | tee -a "$LOG_FILE"
else
  echo "tcpdump not available, skipping packet capture" | tee -a "$LOG_FILE"
fi

echo $DIVIDER | tee -a "$LOG_FILE"

# Fix attempts
blue "Attempting potential fixes..." | tee -a "$LOG_FILE"

# Check if there's an issue with Nginx configuration priority
if [ -f "/etc/nginx/sites-enabled/latency.space" ]; then
  if grep -q "location \/_debug\/" /etc/nginx/sites-enabled/latency.space; then
    yellow "⚠️ Found basic location block for _debug, might need higher priority" | tee -a "$LOG_FILE"
    
    # Check if it has ^~ prefix for higher priority
    if ! grep -q "location \^~ \/_debug\/" /etc/nginx/sites-enabled/latency.space; then
      yellow "⚠️ _debug location is missing ^~ prefix for higher priority" | tee -a "$LOG_FILE"
      
      # Make a backup
      cp /etc/nginx/sites-enabled/latency.space /etc/nginx/sites-enabled/latency.space.bak.$(date +%s)
      
      # Try to fix the priority
      echo "Attempting to fix location priority..." | tee -a "$LOG_FILE"
      sed -i 's/location \(\/_debug\/\)/location ^~ \1/g' /etc/nginx/sites-enabled/latency.space
      
      # Test and reload
      if nginx -t; then
        echo "Nginx configuration test passed, reloading..." | tee -a "$LOG_FILE"
        systemctl reload nginx
        green "✅ Nginx reloaded with updated configuration" | tee -a "$LOG_FILE"
      else
        red "❌ Nginx configuration test failed, reverting changes" | tee -a "$LOG_FILE"
        cp /etc/nginx/sites-enabled/latency.space.bak.$(date +%s) /etc/nginx/sites-enabled/latency.space
      fi
    fi
  fi
fi

# Recommendations
blue "Recommendations based on analysis:" | tee -a "$LOG_FILE"

# Create a fix script
cat > "$OUTPUT_DIR/fix-debug-endpoints.sh" << 'EOF'
#!/bin/bash
# Script to fix _debug endpoints

# Add ^~ prefix to _debug location for higher priority
sed -i 's/location[[:space:]]\+\(\/_debug\/\)/location ^~ \1/g' /etc/nginx/sites-enabled/latency.space

# Add explicit location for /_debug/distances
cat << 'NGINX_BLOCK' >> /etc/nginx/sites-enabled/latency.space

# Explicit location for /_debug/distances endpoint
location = /_debug/distances {
    set $upstream_proxy http://proxy:80;
    proxy_pass $upstream_proxy;
    
    # Standard proxy headers
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection 'upgrade';
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Host $host;
    proxy_set_header X-Forwarded-For $remote_addr;
    proxy_set_header X-Destination $host;
    proxy_cache_bypass $http_upgrade;
    
    # Set timeouts for debug endpoints
    proxy_connect_timeout 300s;
    proxy_send_timeout 300s;
    proxy_read_timeout 300s;
}
NGINX_BLOCK

# Test and reload Nginx
nginx -t && systemctl reload nginx

# Restart proxy container
docker restart $(docker ps -q -f name=proxy)
EOF

chmod +x "$OUTPUT_DIR/fix-debug-endpoints.sh"
echo "Created potential fix script at $OUTPUT_DIR/fix-debug-endpoints.sh" | tee -a "$LOG_FILE"

echo $DIVIDER | tee -a "$LOG_FILE"

# Final summary
blue "Debug complete! Important findings:" | tee -a "$LOG_FILE"
echo "1. Debug logs and analysis are saved to $OUTPUT_DIR" | tee -a "$LOG_FILE"
echo "2. A potential fix script is available at $OUTPUT_DIR/fix-debug-endpoints.sh" | tee -a "$LOG_FILE"
echo "3. If the _debug location block exists but isn't working, it might need higher priority" | tee -a "$LOG_FILE"
echo "4. You can run the fix script with: sudo $OUTPUT_DIR/fix-debug-endpoints.sh" | tee -a "$LOG_FILE"
echo "5. After running the fix, test again with: curl http://latency.space/_debug/metrics" | tee -a "$LOG_FILE"

echo $DIVIDER | tee -a "$LOG_FILE"
echo "Debug script completed at $(date)" | tee -a "$LOG_FILE"