#!/bin/bash
# Comprehensive Health Check Script for latency.space server
# This script performs a full system health check and provides actionable recommendations

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

echo ""
blue "🔍 LATENCY.SPACE SERVER HEALTH CHECK"
echo $DIVIDER

# System information
blue "📊 SYSTEM INFORMATION"
echo "Date: $(date)"
echo "Hostname: $(hostname)"
echo "Server IP: $(curl -s -4 ifconfig.me || curl -s -4 icanhazip.com || hostname -I | awk '{print $1}')"
echo "Uptime: $(uptime)"
echo "Kernel: $(uname -r)"
echo "Memory usage: $(free -h | grep Mem | awk '{print $3 "/" $2}')"
echo "Disk usage: $(df -h / | grep -v Filesystem | awk '{print $5 " (" $3 "/" $2 ")"}')"
echo $DIVIDER

# Check services
blue "🔧 SERVICE STATUS"
echo "Nginx: $(systemctl is-active nginx) ($(systemctl status nginx | head -3 | tail -1))"
echo "Docker: $(systemctl is-active docker) ($(systemctl status docker | head -3 | tail -1))"
echo $DIVIDER

# Check Docker containers
blue "🐳 DOCKER CONTAINER STATUS"
docker ps -a
echo ""
echo "Running containers: $(docker ps -q | wc -l)"
echo "Total containers: $(docker ps -a -q | wc -l)"
echo $DIVIDER

# Check Docker disk usage
blue "💾 DOCKER DISK USAGE"
docker system df
echo $DIVIDER

# Check Docker networks
blue "🌐 DOCKER NETWORKS"
docker network ls
echo ""
echo "Docker network details for space-net:"
docker network inspect space-net 2>/dev/null | grep -A 20 "Containers" | grep -v "\"Containers\": {}," || echo "space-net network not found or empty"
echo $DIVIDER

# Container details
check_container() {
  local name=$1
  blue "📦 $name CONTAINER DETAILS"
  
  CONTAINER_ID=$(docker ps -q -f name=$name 2>/dev/null)
  if [ -z "$CONTAINER_ID" ]; then
    red "❌ $name container not running"
    
    # Check if container exists but is stopped
    STOPPED_ID=$(docker ps -a -q -f name=$name 2>/dev/null)
    if [ -n "$STOPPED_ID" ]; then
      yellow "⚠️ $name container exists but is stopped"
      echo "Container logs (last 10 lines):"
      docker logs --tail 10 $STOPPED_ID 2>/dev/null || echo "No logs available"
      
      echo ""
      echo "Container exit reason:"
      docker inspect $STOPPED_ID | grep -A 5 "ExitCode" | grep -v "{},"
    else
      yellow "⚠️ $name container does not exist"
    fi
  else
    green "✅ $name container is running (ID: $CONTAINER_ID)"
    
    echo "Network details:"
    docker inspect --format 'IP: {{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}} | Network: {{range $key, $value := .NetworkSettings.Networks}}{{$key}}{{end}}' $CONTAINER_ID
    
    echo "Ports:"
    docker port $CONTAINER_ID 2>/dev/null || echo "No ports exposed"
    
    echo "Health status:"
    docker inspect --format 'Status: {{.State.Status}} | Started: {{.State.StartedAt}} | Health: {{if .State.Health}}{{.State.Health.Status}}{{else}}No health check{{end}}' $CONTAINER_ID
    
    echo "Logs (last 5 lines):"
    docker logs --tail 5 $CONTAINER_ID 2>/dev/null || echo "No logs available"
  fi
  echo $DIVIDER
}

# Check main containers
check_container "proxy"
check_container "status"
check_container "prometheus"

# Check Nginx configuration
blue "🔧 NGINX CONFIGURATION"
echo "Nginx version: $(nginx -v 2>&1)"
echo "Enabled sites:"
ls -l /etc/nginx/sites-enabled/ 2>/dev/null || echo "No sites enabled"

echo ""
echo "Testing Nginx configuration:"
nginx -t 2>&1

echo ""
echo "Checking status dashboard configuration:"
if [ -f "/etc/nginx/sites-enabled/latency.space" ]; then
  if grep -q "proxy_pass.*172.18.0.4:80" /etc/nginx/sites-enabled/latency.space; then
    green "✅ Status dashboard configuration found (integrated with main site)"
  else
    yellow "⚠️ Status dashboard configuration not found or using unexpected IP/port"
    echo "   The status dashboard should be integrated with the main site (no separate subdomain)"
    echo "   Expected: proxy_pass http://172.18.0.4:80 in the main server block"
  fi
else
  red "❌ Nginx configuration file not found at /etc/nginx/sites-enabled/latency.space"
fi
echo $DIVIDER

# Check DNS resolution
blue "🌐 DNS RESOLUTION"
for domain in latency.space www.latency.space mars.latency.space; do
  echo -n "Resolving $domain: "
  IP=$(getent hosts $domain 2>/dev/null | awk '{print $1}')
  if [ -n "$IP" ]; then
    SERVER_IP=$(curl -s -4 ifconfig.me || curl -s -4 icanhazip.com || hostname -I | awk '{print $1}')
    if [ "$IP" == "$SERVER_IP" ]; then
      green "✅ $IP (matches server IP)"
    elif [[ "$domain" == *".latency.space" && "$domain" != "www.latency.space" && "$domain" != "latency.space" ]]; then
      # For celestial body subdomains, we want the IP to match the server IP directly (no Cloudflare)
      red "❌ $IP (should match server IP $SERVER_IP, DNS is incorrect)"
    else
      # For main domain, we can have Cloudflare or direct IP
      yellow "⚠️ $IP (different from server IP $SERVER_IP, likely using Cloudflare)"
    fi
  else
    red "❌ Failed to resolve"
  fi
done
echo $DIVIDER

# Check Docker container resolution
blue "🔄 INTER-CONTAINER DNS RESOLUTION"
# Check DNS resolution between containers
# First see if proxy container can resolve the status container
PROXY_CONTAINER=$(docker ps -q -f name=proxy 2>/dev/null)
STATUS_CONTAINER=$(docker ps -q -f name=status 2>/dev/null)

if [ -n "$PROXY_CONTAINER" ] && [ -n "$STATUS_CONTAINER" ]; then
  echo "Checking if proxy container can resolve status container..."
  if docker exec $PROXY_CONTAINER getent hosts status &>/dev/null; then
    STATUS_IP=$(docker exec $PROXY_CONTAINER getent hosts status | awk '{print $1}')
    ACTUAL_STATUS_IP=$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $STATUS_CONTAINER)
    
    if [ "$STATUS_IP" == "$ACTUAL_STATUS_IP" ]; then
      green "✅ Proxy can resolve status to correct IP ($STATUS_IP)"
    else
      yellow "⚠️ Proxy resolves status to $STATUS_IP but actual IP is $ACTUAL_STATUS_IP"
    fi
  else
    red "❌ Proxy cannot resolve status container"
    
    # Try to add a host entry as a fix
    echo "Adding manual host entry to /etc/hosts in proxy container..."
    STATUS_IP=$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $STATUS_CONTAINER)
    docker exec $PROXY_CONTAINER sh -c "echo '$STATUS_IP status' >> /etc/hosts"
    if docker exec $PROXY_CONTAINER getent hosts status &>/dev/null; then
      green "✅ Manual host entry added successfully"
    else
      red "❌ Manual host entry failed"
    fi
  fi
else
  if [ -z "$PROXY_CONTAINER" ]; then
    red "❌ Proxy container not running, cannot test inter-container DNS"
  fi
  if [ -z "$STATUS_CONTAINER" ]; then
    red "❌ Status container not running, cannot test inter-container DNS"
  fi
fi
echo $DIVIDER

# Check connectivity
blue "🌐 CONNECTIVITY TESTS"

# Check HTTP and HTTPS for main website
echo "Checking main website (HTTP)..."
curl -s -I -m 5 http://latency.space | head -1 || echo "Failed to connect over HTTP"

echo "Checking main website (HTTPS)..."
curl -s -I -m 5 --insecure https://latency.space | head -1 || echo "Failed to connect over HTTPS"

# Check HTTP and HTTPS for status dashboard
echo "Checking status dashboard (HTTP)..."
curl -s -I -m 5 http://latency.space/ | head -1 || echo "Failed to connect over HTTP"

echo "Checking status dashboard (HTTPS)..."
curl -s -I -m 5 --insecure https://latency.space/ | head -1 || echo "Failed to connect over HTTPS"

# Check HTTP and HTTPS for Mars subdomain
echo "Checking mars subdomain (HTTP)..."
curl -s -I -m 5 http://mars.latency.space | head -1 || echo "Failed to connect over HTTP"

echo "Checking mars subdomain (HTTPS)..."
curl -s -I -m 5 --insecure https://mars.latency.space | head -1 || echo "Failed to connect over HTTPS"

# Check HTTP and HTTPS for Venus subdomain
echo "Checking venus subdomain (HTTP)..."
curl -s -I -m 5 http://venus.latency.space | head -1 || echo "Failed to connect over HTTP"

echo "Checking venus subdomain (HTTPS)..."
curl -s -I -m 5 --insecure https://venus.latency.space | head -1 || echo "Failed to connect over HTTPS"

# Check debug endpoint over HTTP and HTTPS
echo "Checking _debug endpoint (HTTP)..."
curl -s -I -m 5 http://latency.space/_debug/metrics | head -1 || echo "Failed to connect over HTTP"

echo "Checking _debug endpoint (HTTPS)..."
curl -s -I -m 5 --insecure https://latency.space/_debug/metrics | head -1 || echo "Failed to connect over HTTPS"

# Check direct access to containers
echo "Checking direct access to status container..."
curl -s -I -m 3 http://localhost:3000 | head -1 || echo "Failed to connect"

echo "Checking direct access to proxy container (HTTP)..."
curl -s -I -m 3 http://localhost:8080 | head -1 || echo "Failed to connect over HTTP"

echo "Checking direct access to proxy container (HTTPS)..."
curl -s -I -m 3 --insecure https://localhost:8443 | head -1 || echo "Failed to connect over HTTPS"

# Check SSL certificate status
if [ -f "/etc/letsencrypt/live/latency.space/fullchain.pem" ]; then
  echo "Checking SSL certificate expiry..."
  CERT_EXPIRY=$(openssl x509 -enddate -noout -in /etc/letsencrypt/live/latency.space/fullchain.pem | cut -d= -f2)
  echo "Certificate expires: $CERT_EXPIRY"
  
  # Convert to seconds since epoch
  EXPIRY_SECS=$(date -d "$CERT_EXPIRY" +%s)
  NOW_SECS=$(date +%s)
  DAYS_LEFT=$(( ($EXPIRY_SECS - $NOW_SECS) / 86400 ))
  echo "Days until expiry: $DAYS_LEFT"
  
  if [ $DAYS_LEFT -lt 10 ]; then
    red "⚠️ WARNING: SSL certificate expires in less than 10 days!"
  elif [ $DAYS_LEFT -lt 30 ]; then
    yellow "⚠️ SSL certificate expires in less than 30 days"
  else
    green "✅ SSL certificate valid for $DAYS_LEFT days"
  fi
else
  red "❌ SSL certificate not found"
fi
echo $DIVIDER

# Health summary and recommendations
blue "🩺 HEALTH SUMMARY & RECOMMENDATIONS"
issues_found=0
recommendations=()

# Check if Docker is running
if ! systemctl is-active --quiet docker; then
  red "❌ Docker is not running"
  recommendations+=("Start Docker: systemctl start docker && systemctl enable docker")
  issues_found=$((issues_found+1))
fi

# Check if Nginx is running
if ! systemctl is-active --quiet nginx; then
  red "❌ Nginx is not running"
  recommendations+=("Start Nginx: systemctl start nginx && systemctl enable nginx")
  issues_found=$((issues_found+1))
fi

# Check if proxy container is running
if ! docker ps | grep -q proxy; then
  red "❌ Proxy container is not running"
  recommendations+=("Start proxy container: cd /opt/latency-space && docker compose up -d proxy")
  issues_found=$((issues_found+1))
fi

# Check if status container is running
if ! docker ps | grep -q status; then
  red "❌ Status container is not running"
  recommendations+=("Fix and start status container: sudo ./deploy/fix-status-container.sh")
  issues_found=$((issues_found+1))
fi

# Check DNS records (status subdomain removed - now integrated with main site)
server_ip=$(curl -s -4 ifconfig.me || curl -s -4 icanhazip.com || hostname -I | awk '{print $1}')
if [ ! -z "$server_ip" ]; then
  # Check celestial body DNS records, but status.latency.space is no longer used
  echo "Checking celestial body DNS records..."
fi

# Check if we're getting 502 errors
if curl -s -m 5 http://latency.space 2>/dev/null | grep -q "502 Bad Gateway"; then
  red "❌ Main site (latency.space) is returning 502 Bad Gateway"
  recommendations+=("Fix Nginx configuration: sudo ./deploy/install-nginx-config.sh")
  recommendations+=("Restart containers: cd /opt/latency-space && docker compose restart")
  issues_found=$((issues_found+1))
fi

# Check if docker-compose.yml has correct port mapping for status container
if grep -q '"3000:3000"' /opt/latency-space/docker-compose.yml 2>/dev/null; then
  yellow "⚠️ docker-compose.yml has incorrect port mapping for status container (3000:3000)"
  recommendations+=("Fix status container port mapping: sudo ./deploy/fix-status-container.sh")
  issues_found=$((issues_found+1))
fi

# Check for issues with Nginx configuration
if [ -f "/etc/nginx/sites-enabled/latency.space" ]; then
  if grep -q "proxy_pass.*status:3000" /etc/nginx/sites-enabled/latency.space; then
    yellow "⚠️ Nginx configuration is using incorrect port for status container (status:3000)"
    recommendations+=("Update Nginx configuration: sudo ./deploy/install-nginx-config.sh")
    issues_found=$((issues_found+1))
  fi
fi

# Check for _debug endpoint
if ! curl -s -I -m 3 http://latency.space/_debug/metrics 2>/dev/null | grep -q "200 OK"; then
  yellow "⚠️ _debug endpoints are not working properly"
  recommendations+=("Install fixed Nginx configuration: sudo ./deploy/install-nginx-config.sh")
  issues_found=$((issues_found+1))
fi

# Final health verdict
if [ $issues_found -eq 0 ]; then
  green "✅ No major issues detected! System appears to be healthy."
else
  yellow "⚠️ Found $issues_found issue(s) that need attention."
  echo ""
  echo "Recommended fixes:"
  for i in "${!recommendations[@]}"; do
    echo "$(($i+1)). ${recommendations[$i]}"
  done
fi
echo $DIVIDER

# Quick access to useful commands
blue "🔧 QUICK REFERENCE COMMANDS"
echo "1. View container logs: docker logs \$(docker ps -q -f name=status)"
echo "2. Update from repository: cd /opt/latency-space && git pull"
echo "3. Update Nginx config: sudo ./deploy/install-nginx-config.sh"
echo "4. Fix status container: sudo ./deploy/fix-status-container.sh" 
echo "5. Fix DNS records: sudo ./deploy/fix-all-dns.sh"
echo "6. Restart all containers: cd /opt/latency-space && docker compose down && docker compose up -d"
echo "7. Run this health check: sudo ./deploy/server-health-check.sh"
echo "8. See diagnostic data: http://latency.space/diagnostic.html"
echo $DIVIDER

echo "Health check completed at $(date)"
echo ""