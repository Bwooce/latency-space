#!/bin/bash
# Debug script to investigate why status container isn't receiving traffic
# This script collects detailed information about the status dashboard component

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
OUTPUT_DIR="/tmp/status-debug"
mkdir -p "$OUTPUT_DIR"
LOG_FILE="$OUTPUT_DIR/status-debug.log"

# Start logging
echo "Status container debug script started at $(date)" | tee "$LOG_FILE"
echo $DIVIDER | tee -a "$LOG_FILE"

# Check DNS resolution for status.latency.space
blue "Checking DNS resolution for status.latency.space..." | tee -a "$LOG_FILE"
SERVER_IP=$(curl -s -4 ifconfig.me || curl -s -4 icanhazip.com || hostname -I | awk '{print $1}')
echo "Server IP: $SERVER_IP" | tee -a "$LOG_FILE"

STATUS_IP=$(host status.latency.space 2>/dev/null | grep "has address" | awk '{print $4}')
if [ -n "$STATUS_IP" ]; then
  echo "status.latency.space resolves to: $STATUS_IP" | tee -a "$LOG_FILE"
  
  if [ "$STATUS_IP" = "$SERVER_IP" ]; then
    green "✅ DNS is correctly pointing to this server" | tee -a "$LOG_FILE"
  else
    red "❌ DNS is pointing to $STATUS_IP, not to this server ($SERVER_IP)" | tee -a "$LOG_FILE"
    echo "This must be fixed for status.latency.space to work properly" | tee -a "$LOG_FILE"
    echo "Run: sudo ./deploy/fix-all-dns.sh to update DNS records" | tee -a "$LOG_FILE"
  fi
else
  red "❌ Failed to resolve status.latency.space" | tee -a "$LOG_FILE"
fi

echo $DIVIDER | tee -a "$LOG_FILE"

# Check status container
blue "Checking status container..." | tee -a "$LOG_FILE"
STATUS_CONTAINER=$(docker ps -q -f name=status)

if [ -n "$STATUS_CONTAINER" ]; then
  green "✅ Status container is running with ID: $STATUS_CONTAINER" | tee -a "$LOG_FILE"
  
  # Check container networking
  echo -e "\nContainer network details:" | tee -a "$LOG_FILE"
  docker inspect --format 'IP: {{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}} | Network: {{range $key, $value := .NetworkSettings.Networks}}{{$key}}{{end}}' $STATUS_CONTAINER | tee -a "$LOG_FILE"
  
  echo -e "\nContainer port mappings:" | tee -a "$LOG_FILE"
  docker port $STATUS_CONTAINER | tee -a "$LOG_FILE"
  
  echo -e "\nContainer restart count:" | tee -a "$LOG_FILE"
  docker inspect --format 'Restarts: {{.RestartCount}}' $STATUS_CONTAINER | tee -a "$LOG_FILE"
  
  echo -e "\nLast 20 lines of container logs:" | tee -a "$LOG_FILE"
  docker logs --tail 20 $STATUS_CONTAINER | tee -a "$LOG_FILE"
else
  red "❌ Status container is not running" | tee -a "$LOG_FILE"
  
  # Check if container exists but is stopped
  STOPPED_CONTAINER=$(docker ps -a -q -f name=status)
  if [ -n "$STOPPED_CONTAINER" ]; then
    yellow "⚠️ Status container exists but is stopped (ID: $STOPPED_CONTAINER)" | tee -a "$LOG_FILE"
    
    echo -e "\nContainer exit reason:" | tee -a "$LOG_FILE"
    docker inspect $STOPPED_CONTAINER --format '{{.State.Status}}: {{.State.Error}}' | tee -a "$LOG_FILE"
    
    echo -e "\nLast 20 lines of container logs:" | tee -a "$LOG_FILE"
    docker logs --tail 20 $STOPPED_CONTAINER | tee -a "$LOG_FILE"
  else
    red "❌ Status container doesn't exist at all" | tee -a "$LOG_FILE"
  fi
  
  # Try to start status container
  blue "Attempting to start status container..." | tee -a "$LOG_FILE"
  docker compose up -d status
  
  if docker ps | grep -q status; then
    green "✅ Successfully started status container" | tee -a "$LOG_FILE"
  else
    red "❌ Failed to start status container" | tee -a "$LOG_FILE"
  fi
fi

echo $DIVIDER | tee -a "$LOG_FILE"

# Check Nginx configuration
blue "Checking Nginx configuration for status.latency.space..." | tee -a "$LOG_FILE"
if [ -f "/etc/nginx/sites-enabled/latency.space" ]; then
  if grep -q "server_name status.latency.space" /etc/nginx/sites-enabled/latency.space; then
    green "✅ Found server block for status.latency.space" | tee -a "$LOG_FILE"
    
    # Check proxy_pass configuration
    echo -e "\nProxy pass configuration:" | tee -a "$LOG_FILE"
    grep -A 20 "server_name status.latency.space" /etc/nginx/sites-enabled/latency.space | grep "proxy_pass" | tee -a "$LOG_FILE"
    
    # Check if it's using the correct port
    if grep -A 20 "server_name status.latency.space" /etc/nginx/sites-enabled/latency.space | grep -q "proxy_pass.*status:80"; then
      green "✅ Configuration is using correct internal port (status:80)" | tee -a "$LOG_FILE"
    else
      red "❌ Configuration is not using the correct port" | tee -a "$LOG_FILE"
      echo "Saving full server block to $OUTPUT_DIR/status-server-block.txt" | tee -a "$LOG_FILE"
      grep -A 50 "server_name status.latency.space" /etc/nginx/sites-enabled/latency.space > "$OUTPUT_DIR/status-server-block.txt"
    fi
  else
    red "❌ No server block found for status.latency.space" | tee -a "$LOG_FILE"
  fi
else
  red "❌ Nginx configuration file not found" | tee -a "$LOG_FILE"
fi

echo $DIVIDER | tee -a "$LOG_FILE"

# Test connectivity directly
blue "Testing direct connectivity to status container..." | tee -a "$LOG_FILE"

# Test through port mapping
echo "Testing via port 3000 (mapped to container port 80):" | tee -a "$LOG_FILE"
curl -I -s -m 5 http://localhost:3000 | head -1 | tee -a "$LOG_FILE" || echo "Failed to connect" | tee -a "$LOG_FILE"

# Test through Nginx
echo -e "\nTesting via Nginx (status.latency.space):" | tee -a "$LOG_FILE"
curl -I -s -m 5 http://status.latency.space | head -1 | tee -a "$LOG_FILE" || echo "Failed to connect" | tee -a "$LOG_FILE"

# Test with direct IP and Host header
echo -e "\nTesting with direct IP and Host header:" | tee -a "$LOG_FILE"
curl -I -s -m 5 -H "Host: status.latency.space" http://$SERVER_IP | head -1 | tee -a "$LOG_FILE" || echo "Failed to connect" | tee -a "$LOG_FILE"

echo $DIVIDER | tee -a "$LOG_FILE"

# Check Nginx error logs
blue "Checking Nginx error logs..." | tee -a "$LOG_FILE"
if [ -f "/var/log/nginx/error.log" ]; then
  echo "Last 20 lines of Nginx error log:" | tee -a "$LOG_FILE"
  tail -n 20 /var/log/nginx/error.log | tee -a "$LOG_FILE"
else
  echo "No Nginx error log found at /var/log/nginx/error.log" | tee -a "$LOG_FILE"
fi

echo -e "\nChecking Nginx access logs for status.latency.space requests:" | tee -a "$LOG_FILE"
if [ -f "/var/log/nginx/access.log" ]; then
  grep "status.latency.space" /var/log/nginx/access.log | tail -n 10 | tee -a "$LOG_FILE" || echo "No status.latency.space requests found in access log" | tee -a "$LOG_FILE"
else
  echo "No Nginx access log found at /var/log/nginx/access.log" | tee -a "$LOG_FILE"
fi

echo $DIVIDER | tee -a "$LOG_FILE"

# Check Docker networks and DNS resolution
blue "Checking Docker networks and DNS resolution..." | tee -a "$LOG_FILE"

# Get status container network details
if [ -n "$STATUS_CONTAINER" ]; then
  STATUS_NETWORK=$(docker inspect --format '{{range $key, $value := .NetworkSettings.Networks}}{{$key}}{{end}}' $STATUS_CONTAINER)
  STATUS_IP=$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $STATUS_CONTAINER)
  
  echo "Status container network: $STATUS_NETWORK" | tee -a "$LOG_FILE"
  echo "Status container IP: $STATUS_IP" | tee -a "$LOG_FILE"
  
  # Check DNS resolution from proxy container
  PROXY_CONTAINER=$(docker ps -q -f name=proxy)
  if [ -n "$PROXY_CONTAINER" ]; then
    echo -e "\nChecking if proxy container can resolve status container..." | tee -a "$LOG_FILE"
    if docker exec $PROXY_CONTAINER getent hosts status &>/dev/null; then
      PROXY_RESOLVE=$(docker exec $PROXY_CONTAINER getent hosts status | awk '{print $1}')
      echo "Proxy resolves status to: $PROXY_RESOLVE" | tee -a "$LOG_FILE"
      
      if [ "$PROXY_RESOLVE" = "$STATUS_IP" ]; then
        green "✅ Proxy correctly resolves status container" | tee -a "$LOG_FILE"
      else
        red "❌ Proxy resolves status to wrong IP ($PROXY_RESOLVE vs $STATUS_IP)" | tee -a "$LOG_FILE"
        
        # Try to fix DNS resolution
        echo "Adding manual DNS entry to proxy container..." | tee -a "$LOG_FILE"
        docker exec $PROXY_CONTAINER sh -c "echo '$STATUS_IP status' >> /etc/hosts"
        if docker exec $PROXY_CONTAINER getent hosts status | grep -q "$STATUS_IP"; then
          green "✅ Successfully added manual DNS entry" | tee -a "$LOG_FILE"
        else
          red "❌ Failed to add manual DNS entry" | tee -a "$LOG_FILE"
        fi
      fi
    else
      red "❌ Proxy cannot resolve status container" | tee -a "$LOG_FILE"
      
      # Try to fix DNS resolution
      echo "Adding manual DNS entry to proxy container..." | tee -a "$LOG_FILE"
      docker exec $PROXY_CONTAINER sh -c "echo '$STATUS_IP status' >> /etc/hosts"
      if docker exec $PROXY_CONTAINER getent hosts status | grep -q "$STATUS_IP"; then
        green "✅ Successfully added manual DNS entry" | tee -a "$LOG_FILE"
      else
        red "❌ Failed to add manual DNS entry" | tee -a "$LOG_FILE"
      fi
    fi
  else
    yellow "⚠️ Proxy container is not running, can't check DNS resolution" | tee -a "$LOG_FILE"
  fi
else
  yellow "⚠️ Status container is not running, can't check DNS resolution" | tee -a "$LOG_FILE"
fi

echo $DIVIDER | tee -a "$LOG_FILE"

# Check potential issues with Docker networking
blue "Checking Docker networking..." | tee -a "$LOG_FILE"

# Check if bridge network exists and is properly configured
echo "Docker networks:" | tee -a "$LOG_FILE"
docker network ls | tee -a "$LOG_FILE"

echo -e "\nDetails of space-net network:" | tee -a "$LOG_FILE"
if docker network ls | grep -q space-net; then
  docker network inspect space-net | grep -A 50 "Containers" | tee -a "$LOG_FILE"
else
  red "❌ space-net network doesn't exist" | tee -a "$LOG_FILE"
  
  # Try to create the network
  echo "Creating space-net network..." | tee -a "$LOG_FILE"
  docker network create space-net
  if docker network ls | grep -q space-net; then
    green "✅ Successfully created space-net network" | tee -a "$LOG_FILE"
  else
    red "❌ Failed to create space-net network" | tee -a "$LOG_FILE"
  fi
fi

echo $DIVIDER | tee -a "$LOG_FILE"

# Try to diagnose and fix issues
blue "Attempting to diagnose and fix issues..." | tee -a "$LOG_FILE"

# Create fix script
cat > "$OUTPUT_DIR/fix-status-dashboard.sh" << 'EOF'
#!/bin/bash
# Script to fix status dashboard issues

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }

# Check if script is run as root
if [ "$EUID" -ne 0 ]; then
  red "Please run this script as root"
  echo "Try: sudo $0"
  exit 1
fi

# Ensure we're in the right directory
cd /opt/latency-space || { red "Could not change to /opt/latency-space directory"; exit 1; }

# Stop and remove existing status container
blue "Stopping and removing existing status container..."
docker stop $(docker ps -a -q --filter name=status) 2>/dev/null
docker rm $(docker ps -a -q --filter name=status) 2>/dev/null

# Ensure the Docker network exists
blue "Ensuring Docker network exists..."
if ! docker network ls | grep -q space-net; then
  docker network create space-net
fi

# Rebuild the status container from scratch
blue "Rebuilding status container..."
docker-compose build --no-cache status || docker compose build --no-cache status

# Start the status container
blue "Starting status container..."
docker-compose up -d status || docker compose up -d status

# Verify the container is running
if docker ps | grep -q status; then
  green "✅ Status container is running"
else
  red "❌ Status container failed to start"
  exit 1
fi

# Get container IP
STATUS_IP=$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=status))
echo "Status container IP: $STATUS_IP"

# Update proxy container's hosts file
PROXY_CONTAINER=$(docker ps -q -f name=proxy)
if [ -n "$PROXY_CONTAINER" ]; then
  blue "Updating proxy container hosts file..."
  docker exec $PROXY_CONTAINER sh -c "grep -q status /etc/hosts || echo '$STATUS_IP status' >> /etc/hosts"
fi

# Run the Nginx configuration install script
blue "Installing Nginx configuration..."
if [ -f "deploy/install-nginx-config.sh" ]; then
  bash deploy/install-nginx-config.sh
else
  red "❌ Nginx installation script not found"
fi

# Reload Nginx to apply changes
blue "Reloading Nginx..."
systemctl reload nginx

# Final check
echo "Testing status dashboard:"
curl -I -s http://status.latency.space | head -1

green "✅ Status dashboard fixes complete!"
EOF

chmod +x "$OUTPUT_DIR/fix-status-dashboard.sh"
echo "Created fix script at $OUTPUT_DIR/fix-status-dashboard.sh" | tee -a "$LOG_FILE"

echo $DIVIDER | tee -a "$LOG_FILE"

# Final summary and recommendations
blue "Summary and Recommendations:" | tee -a "$LOG_FILE"

# Determine key issues
if [ -z "$STATUS_CONTAINER" ]; then
  red "❌ Status container is not running - this is the primary issue" | tee -a "$LOG_FILE"
  echo "Run the fix script to rebuild and restart the status container" | tee -a "$LOG_FILE"
elif [ "$STATUS_IP" != "$SERVER_IP" ] && [ -n "$STATUS_IP" ] && [ -n "$SERVER_IP" ]; then
  red "❌ DNS issue - status.latency.space is not pointing to this server" | tee -a "$LOG_FILE"
  echo "Run ./deploy/fix-all-dns.sh to update DNS records" | tee -a "$LOG_FILE"
elif ! grep -q "proxy_pass.*status:80" /etc/nginx/sites-enabled/latency.space; then
  red "❌ Nginx configuration issue - incorrect proxy_pass" | tee -a "$LOG_FILE"
  echo "Run ./deploy/install-nginx-config.sh to fix Nginx configuration" | tee -a "$LOG_FILE"
elif ! docker exec $PROXY_CONTAINER getent hosts status | grep -q "$STATUS_IP"; then
  red "❌ Container DNS resolution issue - proxy can't resolve status" | tee -a "$LOG_FILE"
  echo "Run the fix script to add a manual DNS entry" | tee -a "$LOG_FILE"
else
  yellow "⚠️ No obvious issues found, try running the fix script to rebuild everything" | tee -a "$LOG_FILE"
fi

echo -e "\nTo fix the issues, run:" | tee -a "$LOG_FILE"
echo "sudo $OUTPUT_DIR/fix-status-dashboard.sh" | tee -a "$LOG_FILE"

echo $DIVIDER | tee -a "$LOG_FILE"
echo "Debug script completed at $(date)" | tee -a "$LOG_FILE"
echo "Log file: $LOG_FILE" | tee -a "$LOG_FILE"
echo "Fix script: $OUTPUT_DIR/fix-status-dashboard.sh" | tee -a "$LOG_FILE"