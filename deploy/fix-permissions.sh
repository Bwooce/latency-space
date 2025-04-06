#!/bin/bash
# Fix permissions for Docker volumes and directories

set -e

echo "🔧 Fixing permissions for Docker directories..."

# Change to the project directory
cd /opt/latency-space || { echo "❌ Failed to change directory"; exit 1; }

# Create necessary directories if they don't exist
mkdir -p monitoring/prometheus/rules
mkdir -p config
mkdir -p certs

# Fix permissions for all directories
echo "👉 Setting ownership for directories..."
chown -R root:root .

# Make sure prometheus.yml exists
if [ ! -f monitoring/prometheus/prometheus.yml ]; then
  echo "📝 Creating prometheus.yml..."
  cat > monitoring/prometheus/prometheus.yml << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'latency-proxy'
    static_configs:
      - targets: ['proxy:9090']

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']

rule_files:
  - rules/*.yml
EOF
fi

# Make sure directories are accessible
chmod -R 755 monitoring
chmod -R 755 config
chmod -R 755 certs

echo "✅ Permissions fixed successfully!"
echo "👉 Now run: docker compose up -d --build"