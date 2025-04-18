#!/bin/bash
# Simple server diagnostic script that outputs to a static HTML file
# Run this on your server to identify deployment issues

OUTPUT_FILE="/var/www/html/status.html"
LOG_FILE="/tmp/latency-space-debug.log"

echo "Starting diagnostic..." > $LOG_FILE

# Create directory for output file if it doesn't exist
mkdir -p $(dirname $OUTPUT_FILE) 2>> $LOG_FILE

# Start HTML file
cat > $OUTPUT_FILE << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Latency Space Server Status</title>
    <style>
        body { font-family: sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; }
        pre { background-color: #f5f5f5; padding: 10px; border-radius: 4px; overflow-x: auto; }
        .section { margin-bottom: 30px; border-bottom: 1px solid #ddd; padding-bottom: 20px; }
        h2 { color: #333; }
        .error { color: red; font-weight: bold; }
        .success { color: green; font-weight: bold; }
    </style>
</head>
<body>
    <h1>Latency Space Server Status</h1>
    <p>Generated: $(date)</p>
    
    <div class="section">
        <h2>System Information</h2>
        <pre>
Hostname: $(hostname)
Kernel: $(uname -r)
Uptime: $(uptime)
        </pre>
    </div>
EOF

# Function to append a section
add_section() {
    local title="$1"
    local command="$2"
    echo "Running: $command" >> $LOG_FILE
    
    echo "<div class=\"section\">" >> $OUTPUT_FILE
    echo "<h2>$title</h2>" >> $OUTPUT_FILE
    echo "<pre>" >> $OUTPUT_FILE
    eval "$command" >> $OUTPUT_FILE 2>&1 || echo "<span class=\"error\">Command failed!</span>" >> $OUTPUT_FILE
    echo "</pre>" >> $OUTPUT_FILE
    echo "</div>" >> $OUTPUT_FILE
}

# Add various diagnostic sections
add_section "Disk Space" "df -h | grep -v tmpfs"
add_section "Memory Usage" "free -h"
add_section "Docker Status" "systemctl status docker | head -20"
add_section "Nginx Status" "systemctl status nginx | head -20"
add_section "DNS Resolution" "cat /etc/resolv.conf && echo -e '\nExternal DNS test:' && host github.com"
add_section "Docker Containers" "docker ps -a"
add_section "Docker Networks" "docker network ls"
add_section "Docker Compose Config" "cd /opt/latency-space && docker compose config --services"

# Add container logs if they exist
echo "<div class=\"section\">" >> $OUTPUT_FILE
echo "<h2>Container Logs</h2>" >> $OUTPUT_FILE

for container in proxy status prometheus grafana; do
    container_id=$(docker ps -q -f name="$container")
    if [ -n "$container_id" ]; then
        echo "<h3>$container Logs</h3>" >> $OUTPUT_FILE
        echo "<pre>" >> $OUTPUT_FILE
        docker logs --tail 30 "$container_id" >> $OUTPUT_FILE 2>&1
        echo "</pre>" >> $OUTPUT_FILE
    else
        echo "<h3>$container</h3>" >> $OUTPUT_FILE
        echo "<pre class=\"error\">Container not running</pre>" >> $OUTPUT_FILE
    fi
done

echo "</div>" >> $OUTPUT_FILE

# Check networking between containers
echo "<div class=\"section\">" >> $OUTPUT_FILE
echo "<h2>Network Connectivity Tests</h2>" >> $OUTPUT_FILE
echo "<pre>" >> $OUTPUT_FILE

echo "External URL tests:" >> $OUTPUT_FILE
for url in http://latency.space http://status.latency.space http://mars.latency.space; do
    echo -n "Testing $url: " >> $OUTPUT_FILE
    curl -I -s -m 3 "$url" | head -1 >> $OUTPUT_FILE || echo "Failed to connect" >> $OUTPUT_FILE
done

echo -e "\nInternal container resolution:" >> $OUTPUT_FILE
for host in proxy status prometheus; do
    echo -n "Resolving $host: " >> $OUTPUT_FILE
    getent hosts "$host" >> $OUTPUT_FILE 2>&1 || echo "Failed" >> $OUTPUT_FILE
done

# Show Nginx configuration
echo -e "\nNginx site configuration:" >> $OUTPUT_FILE
cat /etc/nginx/sites-enabled/latency.space >> $OUTPUT_FILE 2>&1 || echo "Config file not found" >> $OUTPUT_FILE

echo "</pre>" >> $OUTPUT_FILE
echo "</div>" >> $OUTPUT_FILE

# Check permissions
echo "<div class=\"section\">" >> $OUTPUT_FILE
echo "<h2>File System Permissions</h2>" >> $OUTPUT_FILE
echo "<pre>" >> $OUTPUT_FILE

echo "Repository permissions:" >> $OUTPUT_FILE
ls -la /opt/latency-space | head -20 >> $OUTPUT_FILE

echo -e "\nMount points:" >> $OUTPUT_FILE
mount | grep -E 'latency|opt' >> $OUTPUT_FILE

echo -e "\nDocker volumes:" >> $OUTPUT_FILE
docker volume ls | grep latency >> $OUTPUT_FILE

echo "</pre>" >> $OUTPUT_FILE
echo "</div>" >> $OUTPUT_FILE

# Add fixing instructions
echo "<div class=\"section\">" >> $OUTPUT_FILE
echo "<h2>Common Fixes</h2>" >> $OUTPUT_FILE
echo "<pre>" >> $OUTPUT_FILE
cat << 'FIXES' >> $OUTPUT_FILE
# Restart all containers
cd /opt/latency-space && docker compose down && docker compose up -d

# Fix DNS issues
sudo ./deploy/fix-dns.sh

# Update Nginx configuration
sudo ./deploy/update-nginx.sh

# Pull latest changes
cd /opt/latency-space && git pull

# Check logs for specific container
docker logs latency-space-proxy-1
FIXES
echo "</pre>" >> $OUTPUT_FILE
echo "</div>" >> $OUTPUT_FILE

# Close HTML file
echo "</body></html>" >> $OUTPUT_FILE

# Set permissions
chmod 644 $OUTPUT_FILE

echo "Diagnostic complete. Results at: http://latency.space/status.html"
echo "If Nginx is working correctly, you should be able to access this page."