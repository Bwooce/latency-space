#!/bin/bash
# Comprehensive diagnostic script for latency.space deployment
# This script collects system and service information to diagnose deployment issues

OUTPUT_FILE="/tmp/latency-space-diagnostic.txt"
HTML_OUTPUT="/var/www/html/diagnostic.html"

# Create output file with header
cat > $OUTPUT_FILE << EOF
==================================================
LATENCY SPACE DIAGNOSTIC REPORT
Timestamp: $(date)
Hostname: $(hostname)
==================================================

EOF

# Function to append section headers to output
section() {
  echo -e "\n\n==================================================\n$1\n==================================================\n" >> $OUTPUT_FILE
}

# System information
section "SYSTEM INFORMATION"
echo "Kernel: $(uname -a)" >> $OUTPUT_FILE
echo "Distribution: $(cat /etc/os-release | grep PRETTY_NAME | cut -d= -f2)" >> $OUTPUT_FILE
echo "Memory: $(free -h)" >> $OUTPUT_FILE
echo "Disk usage: $(df -h | grep -v tmpfs | grep -v udev)" >> $OUTPUT_FILE
echo "CPU info: $(grep "model name" /proc/cpuinfo | head -1)" >> $OUTPUT_FILE

# Network information
section "NETWORK INFORMATION"
echo "Network interfaces:" >> $OUTPUT_FILE
ip addr show | grep -E "inet|^[0-9]" >> $OUTPUT_FILE
echo -e "\nDefault route:" >> $OUTPUT_FILE
ip route | grep default >> $OUTPUT_FILE
echo -e "\nDNS configuration:" >> $OUTPUT_FILE
cat /etc/resolv.conf >> $OUTPUT_FILE
echo -e "\nDocker DNS configuration:" >> $OUTPUT_FILE
if [ -f /etc/docker/daemon.json ]; then
  cat /etc/docker/daemon.json >> $OUTPUT_FILE
else
  echo "Docker daemon.json not found" >> $OUTPUT_FILE
fi

# DNS tests
section "DNS RESOLUTION TESTS"
echo "Testing internal DNS resolution..." >> $OUTPUT_FILE
for host in proxy status prometheus grafana; do
  echo -n "Resolving $host: " >> $OUTPUT_FILE
  getent hosts $host >> $OUTPUT_FILE || echo "Failed" >> $OUTPUT_FILE
done

echo -e "\nTesting external DNS resolution..." >> $OUTPUT_FILE
for ext_host in github.com google.com cloudflare.com; do
  echo -n "Resolving $ext_host: " >> $OUTPUT_FILE
  getent hosts $ext_host >> $OUTPUT_FILE || echo "Failed" >> $OUTPUT_FILE
done

# Nginx information
section "NGINX CONFIGURATION"
echo "Nginx version: $(nginx -v 2>&1)" >> $OUTPUT_FILE
echo -e "\nNginx status:" >> $OUTPUT_FILE
systemctl status nginx | head -20 >> $OUTPUT_FILE
echo -e "\nNginx enabled sites:" >> $OUTPUT_FILE
ls -la /etc/nginx/sites-enabled/ >> $OUTPUT_FILE
echo -e "\nLatency.space configuration:" >> $OUTPUT_FILE
cat /etc/nginx/sites-available/latency.space >> $OUTPUT_FILE 2>&1
echo -e "\nNginx configuration test:" >> $OUTPUT_FILE
nginx -t >> $OUTPUT_FILE 2>&1
echo -e "\nNginx ports:" >> $OUTPUT_FILE
ss -tlnp | grep nginx >> $OUTPUT_FILE

# Docker information
section "DOCKER INFORMATION"
echo "Docker version: $(docker --version)" >> $OUTPUT_FILE
echo "Docker compose version: $(docker compose version)" >> $OUTPUT_FILE
echo -e "\nDocker status:" >> $OUTPUT_FILE
systemctl status docker | head -20 >> $OUTPUT_FILE
echo -e "\nDocker network list:" >> $OUTPUT_FILE
docker network ls >> $OUTPUT_FILE
echo -e "\nDocker containers:" >> $OUTPUT_FILE
docker ps -a >> $OUTPUT_FILE
echo -e "\nDocker networks info:" >> $OUTPUT_FILE
for net in $(docker network ls --format "{{.Name}}"); do
  echo -e "\nNetwork: $net" >> $OUTPUT_FILE
  docker network inspect $net | grep -A 20 "Containers" >> $OUTPUT_FILE
done

# Docker compose status
section "DOCKER COMPOSE STATUS"
echo "Current directory: $(pwd)" >> $OUTPUT_FILE
cd /opt/latency-space || echo "Failed to cd to /opt/latency-space" >> $OUTPUT_FILE
echo -e "\nDocker compose config:" >> $OUTPUT_FILE
docker compose config --services >> $OUTPUT_FILE 2>&1
echo -e "\nDocker compose ps:" >> $OUTPUT_FILE
docker compose ps >> $OUTPUT_FILE
echo -e "\nService logs:" >> $OUTPUT_FILE
for service in proxy status prometheus grafana; do
  echo -e "\n--- Logs for $service ---" >> $OUTPUT_FILE
  docker compose logs --tail=50 $service >> $OUTPUT_FILE 2>&1
done

# Connectivity tests
section "CONNECTIVITY TESTS"
echo "Internal connectivity tests:" >> $OUTPUT_FILE
for internal in proxy status prometheus; do
  for port in 80 443 3000 9090; do
    echo -n "Testing connection to $internal:$port: " >> $OUTPUT_FILE
    timeout 2 bash -c "echo > /dev/tcp/$internal/$port" 2>/dev/null && echo "Success" >> $OUTPUT_FILE || echo "Failed" >> $OUTPUT_FILE
  done
done

echo -e "\nExternal connectivity tests:" >> $OUTPUT_FILE
for domain in latency.space status.latency.space mars.latency.space; do
  echo -n "Curl $domain: " >> $OUTPUT_FILE
  curl -I -s -m 5 http://$domain | head -1 >> $OUTPUT_FILE 2>&1 || echo "Failed" >> $OUTPUT_FILE
done

# File permissions and ownership
section "FILE PERMISSIONS"
echo "Repository permissions:" >> $OUTPUT_FILE
ls -la /opt/latency-space >> $OUTPUT_FILE
echo -e "\nConfiguration files permissions:" >> $OUTPUT_FILE
ls -la /opt/latency-space/deploy >> $OUTPUT_FILE
echo -e "\nNginx sites permissions:" >> $OUTPUT_FILE
ls -la /etc/nginx/sites-available >> $OUTPUT_FILE
ls -la /etc/nginx/sites-enabled >> $OUTPUT_FILE

# SSL certificates
section "SSL CERTIFICATES"
echo "Certificate files:" >> $OUTPUT_FILE
ls -la /etc/ssl/certs | grep latency >> $OUTPUT_FILE 2>&1
ls -la /etc/ssl/private | grep latency >> $OUTPUT_FILE 2>&1
echo -e "\nLet's Encrypt certificates:" >> $OUTPUT_FILE
ls -la /etc/letsencrypt/live/ >> $OUTPUT_FILE 2>&1
if [ -d /etc/letsencrypt/live/latency.space ]; then
  echo -e "\nCertificate info:" >> $OUTPUT_FILE
  openssl x509 -in /etc/letsencrypt/live/latency.space/cert.pem -text -noout | grep -E "Subject:|Issuer:|Not Before:|Not After :|DNS:" >> $OUTPUT_FILE 2>&1
fi

# Firewall status
section "FIREWALL STATUS"
echo "UFW status:" >> $OUTPUT_FILE
ufw status >> $OUTPUT_FILE 2>&1
echo -e "\nIPTables rules:" >> $OUTPUT_FILE
iptables -L -n >> $OUTPUT_FILE 2>&1

# Make the output accessible via web
section "DIAGNOSTIC ACCESS"
echo "Creating HTML output file for web access..." >> $OUTPUT_FILE
mkdir -p $(dirname $HTML_OUTPUT)

# Generate HTML file with the diagnostic info
cat > $HTML_OUTPUT << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Latency Space Diagnostic Results</title>
    <style>
        body { font-family: monospace; background-color: #f0f0f0; padding: 20px; }
        pre { background-color: #fff; padding: 20px; border-radius: 5px; overflow-x: auto; }
        h1 { color: #333; }
        .timestamp { color: #666; font-style: italic; }
    </style>
</head>
<body>
    <h1>Latency Space Diagnostic Results</h1>
    <p class="timestamp">Generated on: $(date)</p>
    <pre>$(cat $OUTPUT_FILE)</pre>
</body>
</html>
EOF

chmod 644 $HTML_OUTPUT

echo -e "\nDiagnostic report completed and accessible at http://latency.space/diagnostic.html" >> $OUTPUT_FILE
echo -e "Raw report saved to $OUTPUT_FILE"

# Print completion message to console
echo "Diagnostic report completed and accessible at http://latency.space/diagnostic.html"
echo "Raw report saved to $OUTPUT_FILE"