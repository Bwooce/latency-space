#!/bin/bash
# Comprehensive diagnostic script for latency-space deployment
# This script collects detailed system and service information to diagnose deployment issues
# Output is accessible via http://latency.space/diagnostic.html

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }
yellow() { echo -e "\033[0;33m$1\033[0m"; }

# Check if script is run as root
if [ "$EUID" -ne 0 ]; then
  yellow "⚠️  Running without root privileges. Some tests may fail."
  echo "For complete diagnostics, run with: sudo $0"
fi

# Set up output files
OUTPUT_DIR="/tmp/latency-space"
HTML_DIR="$OUTPUT_DIR/html"
LOG_DIR="$OUTPUT_DIR/logs"
OUTPUT_FILE="$HTML_DIR/diagnostic.html"
LOG_FILE="$LOG_DIR/diagnostic.log"

# Create directory structure
mkdir -p "$HTML_DIR"
mkdir -p "$LOG_DIR"

# Log start of diagnostics
echo "Starting diagnostics at $(date)" | tee -a "$LOG_FILE"

# Create output file with header
cat > "$OUTPUT_FILE" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Latency Space Diagnostic Report</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen-Sans, Ubuntu, Cantarell, "Helvetica Neue", sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        pre {
            background-color: #f8f8f8;
            border: 1px solid #ddd;
            border-radius: 4px;
            padding: 15px;
            overflow-x: auto;
            font-size: 14px;
            line-height: 1.4;
        }
        h1 { color: #2c3e50; margin-top: 20px; }
        h2 { 
            color: #3498db; 
            border-bottom: 2px solid #3498db;
            padding-bottom: 5px;
            margin-top: 30px;
        }
        .timestamp { 
            color: #7f8c8d;
            font-style: italic;
            margin-bottom: 30px;
        }
        .section {
            background-color: white;
            border-radius: 6px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 30px;
            padding: 20px;
        }
        .error { color: #e74c3c; font-weight: bold; }
        .warning { color: #f39c12; font-weight: bold; }
        .success { color: #2ecc71; font-weight: bold; }
        .info { color: #3498db; }
        
        /* Collapsible sections */
        .collapsible {
            background-color: #f8f8f8;
            cursor: pointer;
            padding: 10px;
            width: 100%;
            border: none;
            text-align: left;
            outline: none;
            font-size: 16px;
            border-radius: 4px;
            margin-bottom: 5px;
        }
        .active, .collapsible:hover {
            background-color: #e8e8e8;
        }
        .collapsible:after {
            content: '+';
            font-weight: bold;
            float: right;
            margin-left: 5px;
        }
        .active:after {
            content: '-';
        }
        .content {
            padding: 0 18px;
            max-height: 0;
            overflow: hidden;
            transition: max-height 0.2s ease-out;
            background-color: #f8f8f8;
            border-radius: 0 0 4px 4px;
        }
        
        /* Navigation */
        .toc {
            background-color: white;
            padding: 15px;
            border-radius: 6px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .toc ul {
            list-style-type: none;
            padding-left: 20px;
        }
        .toc a {
            color: #3498db;
            text-decoration: none;
        }
        .toc a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <h1>Latency Space Diagnostic Report</h1>
    <p class="timestamp">Generated on: $(date)</p>
    
    <div class="toc">
        <h2>Table of Contents</h2>
        <ul>
            <li><a href="#summary">Summary</a></li>
            <li><a href="#system">System Information</a></li>
            <li><a href="#network">Network Information</a></li>
            <li><a href="#docker">Docker Status</a></li>
            <li><a href="#containers">Container Status</a></li>
            <li><a href="#nginx">Nginx Configuration</a></li>
            <li><a href="#dns">DNS Resolution</a></li>
            <li><a href="#connectivity">Connectivity Tests</a></li>
            <li><a href="#logs">Recent Logs</a></li>
        </ul>
    </div>
    
    <div class="section" id="summary">
        <h2>Quick Summary</h2>
        <pre>
EOF

# Function to append section headers to output
section() {
  echo -e "<div class=\"section\" id=\"$3\">\n<h2>$1</h2>\n<button class=\"collapsible\">$1 Details (Click to expand)</button>\n<div class=\"content\">\n<pre>" >> "$OUTPUT_FILE"
  
  # Execute command and process output to replace literal \n with actual newlines
  output=$(eval "$2" 2>&1) || output="<span class=\"error\">Command failed!</span>"
  # Replace any literal \n with actual newlines
  output=$(echo "$output" | sed 's/\\n/\n/g')
  echo "$output" >> "$OUTPUT_FILE"
  
  echo "</pre>\n</div>\n</div>" >> "$OUTPUT_FILE"
  
  # Also log to the log file
  echo "=== $1 ===" >> "$LOG_FILE"
  eval "$2" >> "$LOG_FILE" 2>&1 || echo "Command failed!" >> "$LOG_FILE"
  echo "" >> "$LOG_FILE"
}

# Generate the quick summary
generate_summary() {
  echo "LATENCY.SPACE SYSTEM STATUS"
  echo "------------------------"
  
  # System status
  echo -n "Operating System: "
  if [ -f /etc/os-release ]; then
    grep "PRETTY_NAME" /etc/os-release | cut -d= -f2 | tr -d '"'
  else
    echo "Unknown"
  fi
  
  echo "Kernel: $(uname -r)"
  
  # Service status
  echo -n "Nginx: "
  if systemctl is-active --quiet nginx; then
    echo "Running ✅"
  else
    echo "Not running ❌"
  fi
  
  echo -n "Docker: "
  if systemctl is-active --quiet docker; then
    echo "Running ✅"
  else
    echo "Not running ❌"
  fi
  
  # Container status
  echo "CONTAINERS:"
  echo -n "proxy: "
  if docker ps | grep -q proxy; then
    echo "Running ✅"
  else
    echo "Not running ❌"
  fi
  
  echo -n "status: "
  if docker ps | grep -q status; then
    echo "Running ✅"
  else
    echo "Not running ❌"
  fi
  
  echo -n "prometheus: "
  if docker ps | grep -q prometheus; then
    echo "Running ✅"
  else
    echo "Not running ❌"
  fi
  
  # Network status
  echo "NETWORK STATUS:"
  
  echo -n "External IP: "
  SERVER_IP=$(curl -s -4 ifconfig.me || curl -s -4 icanhazip.com || hostname -I | awk '{print $1}')
  echo "$SERVER_IP"
  
  # DNS status
  echo "DNS STATUS:"
  for domain in latency.space status.latency.space; do
    echo -n "$domain: "
    if host $domain &>/dev/null; then
      DOMAIN_IP=$(host $domain | grep "has address" | head -1 | awk '{print $4}')
      if [ "$domain" = "status.latency.space" ] && [ "$DOMAIN_IP" != "$SERVER_IP" ]; then
        echo "$DOMAIN_IP (Should match server IP) ❌"
      else
        echo "$DOMAIN_IP ✅"
      fi
    else
      echo "Failed to resolve ❌"
    fi
  done
  
  # Website status
  echo "WEBSITE STATUS:"
  
  echo -n "Main site (latency.space): "
  if curl -s -I -m 3 http://latency.space | grep -q "200 OK"; then
    echo "200 OK ✅"
  else
    echo "Not responding ❌"
  fi
  
  echo -n "Status dashboard (status.latency.space): "
  if curl -s -I -m 3 http://status.latency.space | grep -q "200 OK"; then
    echo "200 OK ✅"
  else
    echo "Not responding ❌"
  fi
  
  echo -n "Debug endpoint (latency.space/_debug/metrics): "
  if curl -s -I -m 3 http://latency.space/_debug/metrics | grep -q "200 OK"; then
    echo "200 OK ✅"
  else
    echo "Not responding ❌"
  fi
  
  # Issues found
  echo "DETECTED ISSUES:"
  ISSUES_FOUND=0
  
  # Check if Docker is running
  if ! systemctl is-active --quiet docker; then
    echo "- Docker is not running ❌"
    ISSUES_FOUND=$((ISSUES_FOUND+1))
  fi
  
  # Check if Nginx is running
  if ! systemctl is-active --quiet nginx; then
    echo "- Nginx is not running ❌"
    ISSUES_FOUND=$((ISSUES_FOUND+1))
  fi
  
  # Check if proxy container is running
  if ! docker ps | grep -q proxy; then
    echo "- Proxy container is not running ❌"
    ISSUES_FOUND=$((ISSUES_FOUND+1))
  fi
  
  # Check if status container is running
  if ! docker ps | grep -q status; then
    echo "- Status container is not running ❌"
    ISSUES_FOUND=$((ISSUES_FOUND+1))
  fi
  
  # Check DNS records
  if [ -n "$SERVER_IP" ]; then
    STATUS_IP=$(host status.latency.space 2>/dev/null | grep "has address" | head -1 | awk '{print $4}')
    if [ -n "$STATUS_IP" ] && [ "$STATUS_IP" != "$SERVER_IP" ]; then
      echo "- status.latency.space DNS record ($STATUS_IP) doesn't match server IP ($SERVER_IP) ❌"
      ISSUES_FOUND=$((ISSUES_FOUND+1))
    fi
  fi
  
  # Check for 502 errors
  if curl -s -m 3 http://status.latency.space 2>/dev/null | grep -q "502 Bad Gateway"; then
    echo "- status.latency.space is returning 502 Bad Gateway ❌"
    ISSUES_FOUND=$((ISSUES_FOUND+1))
  fi
  
  # Check for port mapping issues
  if docker compose -f /opt/latency-space/docker-compose.yml config 2>/dev/null | grep -q '"3000:3000"'; then
    echo "- Incorrect port mapping in docker-compose.yml (3000:3000 instead of 3000:80) ❌"
    ISSUES_FOUND=$((ISSUES_FOUND+1))
  fi
  
  # Check Nginx configuration
  if [ -f "/etc/nginx/sites-enabled/latency.space" ]; then
    if grep -q "proxy_pass.*status:3000" /etc/nginx/sites-enabled/latency.space; then
      echo "- Nginx configuration using incorrect port (status:3000 instead of status:80) ❌"
      ISSUES_FOUND=$((ISSUES_FOUND+1))
    fi
  fi
  
  if [ $ISSUES_FOUND -eq 0 ]; then
    echo "No major issues detected! ✅"
  fi
  
  echo "RECOMMENDED ACTIONS:"
  if [ $ISSUES_FOUND -gt 0 ]; then
    echo "1. Update from repository: cd /opt/latency-space && git pull"
    echo "2. Fix status container: sudo ./deploy/fix-status-container.sh"
    echo "3. Update Nginx config: sudo ./deploy/install-nginx-config.sh"
    echo "4. Fix DNS records: sudo ./deploy/fix-all-dns.sh"
    echo "5. Restart all containers: cd /opt/latency-space && docker compose down && docker compose up -d"
    echo "6. Run health check: sudo ./deploy/server-health-check.sh"
  else 
    echo "System appears to be healthy! No immediate actions needed."
  fi
}
# Generate summary and fix any \n characters
summary=$(generate_summary)
summary=$(echo "$summary" | sed 's/\\n/\n/g')
echo "$summary" >> "$OUTPUT_FILE"
echo "</pre></div>" >> "$OUTPUT_FILE"

# System information
section "System Information" "echo 'Kernel: $(uname -a)'; echo 'Distribution: $(cat /etc/os-release | grep PRETTY_NAME | cut -d= -f2)'; echo 'Memory:'; free -h; echo -e '\nDisk usage:'; df -h | grep -v tmpfs | grep -v udev; echo -e '\nUptime:'; uptime; echo -e '\nCPU info:'; grep 'model name' /proc/cpuinfo | head -1; echo -e '\nCPU usage:'; top -bn1 | head -15" "system"

# Network information
section "Network Information" "echo 'Network interfaces:'; ip addr show | grep -E 'inet|^[0-9]'; echo -e '\nDefault route:'; ip route | grep default; echo -e '\nDNS configuration:'; cat /etc/resolv.conf; echo -e '\nDocker DNS configuration:'; if [ -f /etc/docker/daemon.json ]; then cat /etc/docker/daemon.json; else echo 'Docker daemon.json not found'; fi; echo -e '\nNetwork connections:'; ss -tuln | grep -E ':(80|443|3000|8080|9090|9091)'; echo -e '\nIP Tables rules:'; iptables -L -n | head -30" "network"

# Docker status
section "Docker Status" "echo 'Docker version: $(docker --version)'; echo 'Docker compose version:'; docker compose version 2>/dev/null || echo 'Docker compose not found'; echo -e '\nDocker status:'; systemctl status docker | head -20; echo -e '\nDocker info:'; docker info | head -30; echo -e '\nDocker networks:'; docker network ls; echo -e '\nDocker disk usage:'; docker system df; echo -e '\nDocker network details:'; docker network inspect space-net 2>/dev/null || echo 'space-net network not found'" "docker"

# Container status
section "Container Status" "echo 'Running containers:'; docker ps; echo -e '\nAll containers:'; docker ps -a; echo -e '\nDocker compose config:'; cd /opt/latency-space 2>/dev/null && (docker compose config 2>/dev/null || echo 'Failed to read compose config'); echo -e '\nProxy container details:'; docker inspect \$(docker ps -q -f name=proxy) 2>/dev/null || echo 'Proxy container not running'; echo -e '\nStatus container details:'; docker inspect \$(docker ps -q -f name=status) 2>/dev/null || echo 'Status container not running'; echo -e '\nContainer logs:'; for c in proxy status prometheus; do echo -e \"\n--- \$c logs ---\"; docker logs --tail 20 \$(docker ps -a -q -f name=\$c) 2>/dev/null || echo \"\$c container not found\"; done" "containers"

# Nginx configuration
section "Nginx Configuration" "echo 'Nginx version: $(nginx -v 2>&1)'; echo -e '\nNginx status:'; systemctl status nginx | head -20; echo -e '\nNginx enabled sites:'; ls -la /etc/nginx/sites-enabled/; echo -e '\nLatency.space configuration:'; cat /etc/nginx/sites-available/latency.space 2>/dev/null || cat /etc/nginx/sites-enabled/latency.space 2>/dev/null || echo 'Configuration file not found'; echo -e '\nNginx configuration test:'; nginx -t 2>&1" "nginx"

# DNS resolution
section "DNS Resolution" "echo 'Testing external DNS resolution:'; for host in github.com google.com cloudflare.com; do echo -n \"Resolving \$host: \"; host \$host || echo 'Failed'; done; echo -e '\nTesting latency.space subdomains:'; for subdomain in latency.space www.latency.space status.latency.space mars.latency.space jupiter.latency.space; do echo -n \"Resolving \$subdomain: \"; host \$subdomain || echo 'Failed'; done; echo -e '\nTesting reverse DNS:'; SERVER_IP=\$(curl -s -4 ifconfig.me || curl -s -4 icanhazip.com || hostname -I | awk '{print \$1}'); if [ -n \"\$SERVER_IP\" ]; then echo -n \"Reverse DNS for \$SERVER_IP: \"; host \$SERVER_IP || echo 'No reverse DNS record'; fi" "dns"

# Connectivity tests
section "Connectivity Tests" "echo 'Internal connectivity tests:'; for service in proxy status prometheus; do echo \"Testing \$service container:\"; if CONTAINER_ID=\$(docker ps -q -f name=\$service 2>/dev/null); then if docker exec \$CONTAINER_ID ping -c 1 1.1.1.1 &>/dev/null; then echo \" - External internet: Success\"; else echo \" - External internet: Failed\"; fi; if docker exec \$CONTAINER_ID getent hosts status &>/dev/null; then echo \" - Can resolve status: Success (\$(docker exec \$CONTAINER_ID getent hosts status | awk '{print \$1}'))\"; else echo \" - Can resolve status: Failed\"; fi; if docker exec \$CONTAINER_ID getent hosts proxy &>/dev/null; then echo \" - Can resolve proxy: Success (\$(docker exec \$CONTAINER_ID getent hosts proxy | awk '{print \$1}'))\"; else echo \" - Can resolve proxy: Failed\"; fi; else echo \"\$service container not running\"; fi; done; echo -e '\nExternal connectivity tests:'; for domain in latency.space www.latency.space status.latency.space mars.latency.space; do echo -n \"Curl \$domain: \"; curl -I -s -m 5 http://\$domain | head -1 || echo 'Failed'; done; echo -e '\nDebug endpoint test:'; echo -n \"Curl latency.space/_debug/metrics: \"; curl -I -s -m 5 http://latency.space/_debug/metrics | head -1 || echo 'Failed'" "connectivity"

# Logs section
section "Recent Logs" "echo 'Nginx error log:'; tail -n 50 /var/log/nginx/error.log 2>/dev/null || echo 'Cannot read Nginx error log'; echo -e '\nNginx access log:'; tail -n 20 /var/log/nginx/access.log 2>/dev/null || echo 'Cannot read Nginx access log'; echo -e '\nSystem log:'; journalctl -n 30 --no-pager; echo -e '\nDocker log:'; journalctl -u docker -n 20 --no-pager" "logs"

# Close HTML file with JavaScript for collapsible sections - use sed to prevent literal \n
js_code=$(cat << 'EOF'
<script>
// Initialize collapsible sections
var coll = document.getElementsByClassName("collapsible");
var i;

for (i = 0; i < coll.length; i++) {
  coll[i].addEventListener("click", function() {
    this.classList.toggle("active");
    var content = this.nextElementSibling;
    if (content.style.maxHeight) {
      content.style.maxHeight = null;
    } else {
      content.style.maxHeight = content.scrollHeight + "px";
    }
  });
}

// Auto-refresh the page every 5 minutes
setTimeout(function() {
  location.reload();
}, 300000);
</script>
</body>
</html>
EOF
)

# Replace any literal \n with actual newlines and write to file
echo "$js_code" | sed 's/\\n/\n/g' >> "$OUTPUT_FILE"

# Configure Nginx to serve the diagnostic page
if [ -f "/etc/nginx/sites-available/latency.space" ]; then
  if ! grep -q "location = /diagnostic.html" /etc/nginx/sites-available/latency.space; then
    # Create a backup of the Nginx config
    cp /etc/nginx/sites-available/latency.space /etc/nginx/sites-available/latency.space.bak.$(date +%s)
    
    # Try to insert after the server_name line for the latency.space server block
    if grep -q "server_name latency.space" /etc/nginx/sites-available/latency.space; then
      TMP_FILE=$(mktemp)
      awk '
      /server_name latency.space/ { 
        print $0
        print "    # Diagnostic page"
        print "    location = /diagnostic.html {"
        print "        alias /tmp/latency-space/html/diagnostic.html;"
        print "        default_type text/html;"
        print "    }"
        next
      }
      { print $0 }
      ' /etc/nginx/sites-available/latency.space > "$TMP_FILE"
      
      # Only update if the changes were made successfully
      if grep -q "location = /diagnostic.html" "$TMP_FILE"; then
        cat "$TMP_FILE" > /etc/nginx/sites-available/latency.space
        rm "$TMP_FILE"
        
        # Test and reload Nginx
        echo "Added diagnostic page location to Nginx config" | tee -a "$LOG_FILE"
        if nginx -t 2>/dev/null; then
          systemctl reload nginx
          echo "Nginx configuration reloaded successfully" | tee -a "$LOG_FILE"
        else
          echo "Nginx configuration test failed, not reloading" | tee -a "$LOG_FILE"
        fi
      else
        echo "Failed to update Nginx config, using standalone config" | tee -a "$LOG_FILE"
        rm "$TMP_FILE"
        
        # Create a separate config file
        cat > /etc/nginx/sites-available/latency-diagnostics << EOF
# Server for diagnostics
server {
    listen 80;
    server_name latency.space www.latency.space;
    
    # Diagnostic page only
    location = /diagnostic.html {
        alias /tmp/latency-space/html/diagnostic.html;
        default_type text/html;
    }
    
    # For all other paths, return 404
    location / {
        return 404;
    }
}
EOF
        ln -sf /etc/nginx/sites-available/latency-diagnostics /etc/nginx/sites-enabled/
        
        # Test and reload Nginx
        if nginx -t 2>/dev/null; then
          systemctl reload nginx
          echo "Created separate diagnostic config and reloaded Nginx" | tee -a "$LOG_FILE"
        else
          echo "Nginx configuration test failed, not reloading" | tee -a "$LOG_FILE"
        fi
      fi
    fi
  fi
fi

# Print final message
echo "Diagnostic report completed and accessible at http://latency.space/diagnostic.html" | tee -a "$LOG_FILE"
echo "Raw report saved to $OUTPUT_FILE" | tee -a "$LOG_FILE"
echo "Log file: $LOG_FILE" | tee -a "$LOG_FILE"

# Create a cronjob to run this script every hour if not already set up
if ! crontab -l 2>/dev/null | grep -q "/opt/latency-space/deploy/diagnostic.sh"; then
  (crontab -l 2>/dev/null; echo "0 * * * * /opt/latency-space/deploy/diagnostic.sh > /tmp/latency-space/diagnostic-cron.log 2>&1") | crontab -
  echo "Added hourly cronjob to update diagnostic report" | tee -a "$LOG_FILE"
fi