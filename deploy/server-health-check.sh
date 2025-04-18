#!/bin/bash
# Health check script for latency.space server
# This script checks the overall system health and connectivity

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }
yellow() { echo -e "\033[0;33m$1\033[0m"; }
DIVIDER="----------------------------------------"

echo ""
blue "🔍 LATENCY.SPACE SERVER HEALTH CHECK"
hr() { echo -e "$DIVIDER"; }
hr

# System information
blue "📊 SYSTEM INFORMATION"
echo "Date: $(date)"
echo "Hostname: $(hostname)"
echo "Server IP: $(curl -s -4 ifconfig.me || curl -s -4 icanhazip.com)"
echo "Uptime: $(uptime)"
hr

# Check services
blue "🔧 SERVICE STATUS"
systemctl status nginx | head -3
systemctl status docker | head -3
hr

# Check Docker containers
blue "🐳 DOCKER CONTAINER STATUS"
docker ps
hr

# Check proxy container details
blue "👾 PROXY CONTAINER DETAILS"
echo "Network details:"
docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=proxy) 2>/dev/null || echo "Proxy container not found"

echo ""
echo "Exposed ports:"
docker port $(docker ps -q -f name=proxy) 2>/dev/null || echo "Proxy container not found"

echo ""
echo "Proxy container logs (last 10 lines):"
docker logs --tail 10 $(docker ps -q -f name=proxy) 2>/dev/null || echo "Proxy container not found or has no logs"
hr

# Check status container details
blue "📊 STATUS CONTAINER DETAILS"
echo "Network details:"
docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker ps -q -f name=status) 2>/dev/null || echo "Status container not found"

echo ""
echo "Exposed ports:"
docker port $(docker ps -q -f name=status) 2>/dev/null || echo "Status container not found"

echo ""
echo "Status container logs (last 10 lines):"
docker logs --tail 10 $(docker ps -q -f name=status) 2>/dev/null || echo "Status container not found or has no logs"
hr

# Check Nginx configuration
blue "🔧 NGINX CONFIGURATION"
echo "Nginx version: $(nginx -v 2>&1)"
echo "Nginx enabled sites:"
ls -l /etc/nginx/sites-enabled/ 2>/dev/null || echo "No sites enabled"
hr

# Check DNS resolution
blue "🌐 DNS RESOLUTION"
for domain in latency.space www.latency.space status.latency.space mars.latency.space; do
    echo -n "Resolving $domain: "
    host $domain 2>/dev/null || echo "Failed"
done
hr

# Check network connectivity - removed interactive flag (-it) which can cause problems
blue "🔌 NETWORK CONNECTIVITY"
echo "Checking if curl is available in the proxy container..."
PROXY_CONTAINER=$(docker ps -q -f name=proxy 2>/dev/null)
STATUS_CONTAINER=$(docker ps -q -f name=status 2>/dev/null)

if [ -z "$PROXY_CONTAINER" ]; then
    red "❌ Proxy container is not running"
elif docker exec $PROXY_CONTAINER which curl &>/dev/null; then
    green "✅ curl is available in proxy container"
    
    echo "Checking connectivity to proxy container..."
    if docker exec $PROXY_CONTAINER curl -s -o /dev/null -w "%{http_code}" http://localhost 2>/dev/null; then
        green "✅ Proxy container is responding to HTTP requests"
    else
        red "❌ Proxy container is not responding to HTTP requests"
    fi
    
    echo "Checking connectivity to status container..."
    if docker exec $PROXY_CONTAINER curl -s -o /dev/null -w "%{http_code}" http://status:3000 2>/dev/null; then
        green "✅ Status container is accessible from proxy container"
    else
        red "❌ Status container is not accessible from proxy container"
    fi
else
    yellow "⚠️ curl is not available in proxy container, using alternative network tests"
    
    echo "Testing proxy container network with ping..."
    if docker exec $PROXY_CONTAINER ping -c 1 localhost &>/dev/null; then
        green "✅ Proxy container can ping itself"
    else
        # Try with simple command if ping is not available
        if docker exec $PROXY_CONTAINER echo "Network test" &>/dev/null; then
            green "✅ Proxy container is responsive"
        else
            red "❌ Proxy container is not responsive to basic commands"
        fi
    fi
    
    echo "Testing connectivity to status container..."
    if docker exec $PROXY_CONTAINER ping -c 1 status &>/dev/null; then
        green "✅ Proxy container can ping status container"
    else
        yellow "⚠️ Cannot ping status container, checking docker network directly"
        
        # Get the container IPs from docker inspect
        PROXY_IP=$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $PROXY_CONTAINER 2>/dev/null)
        STATUS_IP=$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $STATUS_CONTAINER 2>/dev/null)
        
        if [ -n "$PROXY_IP" ] && [ -n "$STATUS_IP" ]; then
            echo "Proxy container IP: $PROXY_IP"
            echo "Status container IP: $STATUS_IP"
            
            # Check if they're on the same network
            PROXY_NETWORK=$(docker inspect --format '{{range $key, $value := .NetworkSettings.Networks}}{{$key}}{{end}}' $PROXY_CONTAINER 2>/dev/null)
            STATUS_NETWORK=$(docker inspect --format '{{range $key, $value := .NetworkSettings.Networks}}{{$key}}{{end}}' $STATUS_CONTAINER 2>/dev/null)
            
            if [ "$PROXY_NETWORK" = "$STATUS_NETWORK" ]; then
                green "✅ Both containers are on the same network: $PROXY_NETWORK"
            else
                red "❌ Containers are on different networks: proxy=$PROXY_NETWORK, status=$STATUS_NETWORK"
            fi
        else
            red "❌ Could not determine container IP addresses"
        fi
    fi
    
    echo ""
    yellow "📝 Recommendation: Install curl in the proxy container for better diagnostics:"
    echo "   docker exec $PROXY_CONTAINER sh -c 'apt-get update && apt-get install -y curl'"
fi
hr

# Check latency.space service
blue "🌐 LATENCY.SPACE SERVICE CHECK"
echo "Checking main website (latency.space)..."
curl -s -I -m 5 http://latency.space | head -1 || echo "Failed to connect"

echo "Checking status subdomain (status.latency.space)..."
curl -s -I -m 5 http://status.latency.space | head -1 || echo "Failed to connect"

echo "Checking mars subdomain (mars.latency.space)..."
curl -s -I -m 5 http://mars.latency.space | head -1 || echo "Failed to connect"
hr

# Health summary and recommendations
blue "🩺 HEALTH SUMMARY & RECOMMENDATIONS"
issues_found=0

# Check if Docker is running
if ! systemctl is-active --quiet docker; then
    red "❌ Docker is not running"
    echo "   Fix: systemctl start docker"
    issues_found=$((issues_found+1))
fi

# Check if Nginx is running
if ! systemctl is-active --quiet nginx; then
    red "❌ Nginx is not running"
    echo "   Fix: systemctl start nginx"
    issues_found=$((issues_found+1))
fi

# Check if proxy container is running
if ! docker ps | grep -q proxy; then
    red "❌ Proxy container is not running"
    echo "   Fix: cd /opt/latency-space && docker-compose up -d proxy"
    issues_found=$((issues_found+1))
fi

# Check if status container is running
if ! docker ps | grep -q status; then
    red "❌ Status container is not running"
    echo "   Fix: cd /opt/latency-space && docker-compose up -d status"
    issues_found=$((issues_found+1))
fi

# Check DNS records
server_ip=$(curl -s -4 ifconfig.me || curl -s -4 icanhazip.com)
if [ ! -z "$server_ip" ]; then
    status_ip=$(host status.latency.space | grep "has address" | awk '{print $4}' 2>/dev/null)
    if [ "$status_ip" != "$server_ip" ]; then
        yellow "⚠️ status.latency.space DNS record ($status_ip) doesn't match server IP ($server_ip)"
        echo "   Fix: Run ./deploy/fix-all-dns.sh to update DNS records"
        issues_found=$((issues_found+1))
    fi
fi

# Check if we're getting 502 errors
if curl -s http://status.latency.space | grep -q "502 Bad Gateway"; then
    red "❌ status.latency.space is returning 502 Bad Gateway"
    echo "   Possible fixes:"
    echo "   1. Verify Nginx configuration: ./deploy/fix-nginx-clean.sh"
    echo "   2. Restart containers: cd /opt/latency-space && docker-compose restart"
    echo "   3. Check container logs: docker logs \$(docker ps -q -f name=status)"
    issues_found=$((issues_found+1))
fi

if [ $issues_found -eq 0 ]; then
    green "✅ No major issues detected! System appears to be healthy."
else
    yellow "⚠️ Found $issues_found issue(s) that need attention."
    echo ""
    echo "Quick fix commands:"
    echo "1. Fix DNS records: ./deploy/fix-all-dns.sh"
    echo "2. Fix Nginx configuration: ./deploy/fix-nginx-clean.sh" 
    echo "3. Restart services: cd /opt/latency-space && docker-compose down && docker-compose up -d"
fi
hr

echo "Health check completed at $(date)"
echo ""