#!/bin/bash
# Comprehensive diagnostic script for latency-space deployment
# This script collects system and service information to diagnose deployment issues
# Output is accessible via http://latency.space/diagnostic.html

OUTPUT_FILE="/tmp/latency-space/html/diagnostic.html"
LOG_FILE="/tmp/latency-space/diagnostic.log"

# Create directory structure
mkdir -p $(dirname $OUTPUT_FILE)
mkdir -p $(dirname $LOG_FILE)

# Create output file with header
cat > $OUTPUT_FILE << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Latency Space Diagnostic Report</title>
    <style>
        body { font-family: monospace; background-color: #f0f0f0; padding: 20px; }
        pre { background-color: #fff; padding: 20px; border-radius: 5px; overflow-x: auto; }
        h1 { color: #333; }
        .timestamp { color: #666; font-style: italic; }
        .section { margin-bottom: 30px; border-bottom: 1px solid #ddd; padding-bottom: 20px; }
        h2 { color: #333; }
    </style>
</head>
<body>
    <h1>Latency Space Diagnostic Report</h1>
    <p class="timestamp">Generated on: $(date)</p>
EOF

# Function to append section headers to output
section() {
  echo -e "<div class=\"section\">\n<h2>$1</h2>\n<pre>" >> $OUTPUT_FILE
  eval "$2" >> $OUTPUT_FILE 2>&1 || echo "<span style=\"color: red;\">Command failed!</span>" >> $OUTPUT_FILE
  echo "</pre>\n</div>" >> $OUTPUT_FILE
}

# System information
section "SYSTEM INFORMATION" "echo 'Kernel: $(uname -a)'; echo 'Distribution: $(cat /etc/os-release | grep PRETTY_NAME | cut -d= -f2)'; echo 'Memory: $(free -h)'; echo 'Disk usage: $(df -h | grep -v tmpfs | grep -v udev)'; echo 'CPU info: $(grep \"model name\" /proc/cpuinfo | head -1)'"

# Network information
section "NETWORK INFORMATION" "echo 'Network interfaces:'; ip addr show | grep -E 'inet|^[0-9]'; echo -e '\nDefault route:'; ip route | grep default; echo -e '\nDNS configuration:'; cat /etc/resolv.conf; echo -e '\nDocker DNS configuration:'; if [ -f /etc/docker/daemon.json ]; then cat /etc/docker/daemon.json; else echo 'Docker daemon.json not found'; fi"

# DNS tests
section "DNS RESOLUTION TESTS" "echo 'Testing internal DNS resolution...'; for host in proxy status prometheus grafana; do echo -n \"Resolving \$host: \"; getent hosts \$host || echo 'Failed'; done; echo -e '\nTesting external DNS resolution...'; for ext_host in github.com google.com cloudflare.com; do echo -n \"Resolving \$ext_host: \"; getent hosts \$ext_host || echo 'Failed'; done"

# Nginx information
section "NGINX CONFIGURATION" "echo 'Nginx version: $(nginx -v 2>&1)'; echo -e '\nNginx status:'; systemctl status nginx | head -20; echo -e '\nNginx enabled sites:'; ls -la /etc/nginx/sites-enabled/; echo -e '\nLatency.space configuration:'; cat /etc/nginx/sites-available/latency.space 2>&1; echo -e '\nNginx configuration test:'; nginx -t 2>&1; echo -e '\nNginx ports:'; ss -tlnp | grep nginx"

# Docker information
section "DOCKER INFORMATION" "echo 'Docker version: $(docker --version)'; echo 'Docker compose version: $(docker compose version)'; echo -e '\nDocker status:'; systemctl status docker | head -20; echo -e '\nDocker network list:'; docker network ls; echo -e '\nDocker containers:'; docker ps -a; echo -e '\nDocker networks info:'; for net in \$(docker network ls --format \"{{.Name}}\"); do echo -e \"\nNetwork: \$net\"; docker network inspect \$net | grep -A 20 \"Containers\"; done"

# Docker compose status
section "DOCKER COMPOSE STATUS" "echo 'Current directory: $(pwd)'; cd /opt/latency-space || echo 'Failed to cd to /opt/latency-space'; echo -e '\nDocker compose config:'; docker compose config --services 2>&1; echo -e '\nDocker compose ps:'; docker compose ps; echo -e '\nService logs:'; for service in proxy status prometheus grafana; do echo -e \"\n--- Logs for \$service ---\"; docker compose logs --tail=50 \$service 2>&1; done"

# Connectivity tests
section "CONNECTIVITY TESTS" "echo 'Internal connectivity tests:'; for internal in proxy status prometheus; do for port in 80 443 3000 9090; do echo -n \"Testing connection to \$internal:\$port: \"; timeout 2 bash -c \"echo > /dev/tcp/\$internal/\$port\" 2>/dev/null && echo 'Success' || echo 'Failed'; done; done; echo -e '\nExternal connectivity tests:'; for domain in latency.space status.latency.space mars.latency.space; do echo -n \"Curl \$domain: \"; curl -I -s -m 5 http://\$domain | head -1 2>&1 || echo 'Failed'; done"

# File permissions and ownership
section "FILE PERMISSIONS" "echo 'Repository permissions:'; ls -la /opt/latency-space; echo -e '\nConfiguration files permissions:'; ls -la /opt/latency-space/deploy; echo -e '\nNginx sites permissions:'; ls -la /etc/nginx/sites-available; ls -la /etc/nginx/sites-enabled"

# SSL certificates
section "SSL CERTIFICATES" "echo 'Certificate files:'; ls -la /etc/ssl/certs | grep latency; ls -la /etc/ssl/private | grep latency; echo -e '\nLet\\'s Encrypt certificates:'; ls -la /etc/letsencrypt/live/ 2>&1; if [ -d /etc/letsencrypt/live/latency.space ]; then echo -e '\nCertificate info:'; openssl x509 -in /etc/letsencrypt/live/latency.space/cert.pem -text -noout | grep -E 'Subject:|Issuer:|Not Before:|Not After :|DNS:' 2>&1; fi"

# Firewall status
section "FIREWALL STATUS" "echo 'UFW status:'; ufw status 2>&1; echo -e '\nIPTables rules:'; iptables -L -n 2>&1"

# Close HTML file
echo "</body></html>" >> $OUTPUT_FILE

# Configure Nginx to serve the diagnostic page if needed
if [ -f "/etc/nginx/sites-available/latency.space" ]; then
  if ! grep -q "location = /diagnostic.html" /etc/nginx/sites-available/latency.space; then
    # Add location for diagnostic.html
    cp /etc/nginx/sites-available/latency.space /etc/nginx/sites-available/latency.space.bak
    
    # Try to insert after the server_name line
    if grep -q "server_name latency.space" /etc/nginx/sites-available/latency.space; then
      sed -i '/server_name latency.space/a \    # Diagnostic page\n    location = \/diagnostic.html {\n        alias \/tmp\/latency-space\/html\/diagnostic.html;\n        default_type text\/html;\n    }' /etc/nginx/sites-available/latency.space
    else
      # Or just create a new server block
      cat > /etc/nginx/sites-available/latency-diagnostics << EOF
# Server for diagnostics
server {
    listen 80;
    server_name latency.space;
    
    # Diagnostic page
    location = /diagnostic.html {
        alias /tmp/latency-space/html/diagnostic.html;
        default_type text/html;
    }
}
EOF
      ln -sf /etc/nginx/sites-available/latency-diagnostics /etc/nginx/sites-enabled/
    fi
    
    # Test and reload Nginx
    nginx -t && systemctl reload nginx
  fi
fi

echo "Diagnostic report completed and accessible at http://latency.space/diagnostic.html"
echo "Raw report saved to $OUTPUT_FILE"

# Create a cronjob to run this script every hour if not already set up
if ! crontab -l | grep -q "/opt/latency-space/deploy/diagnostic.sh"; then
  (crontab -l 2>/dev/null; echo "0 * * * * /opt/latency-space/deploy/diagnostic.sh > /tmp/latency-space/diagnostic-cron.log 2>&1") | crontab -
  echo "Added hourly cronjob to update diagnostic report"
fi