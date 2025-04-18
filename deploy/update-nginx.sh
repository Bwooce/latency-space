#!/bin/bash
# Quick script to update Nginx configuration

set -e

echo "🔄 Updating Nginx configuration..."

# Check if the nginx-proxy.conf file exists
if [ ! -f "deploy/nginx-proxy.conf" ]; then
  echo "❌ Configuration file not found"
  exit 1
fi

# Copy the configuration to Nginx
cp deploy/nginx-proxy.conf /etc/nginx/sites-available/latency.space

# Create symlink if it doesn't exist
if [ ! -f "/etc/nginx/sites-enabled/latency.space" ]; then
  ln -sf /etc/nginx/sites-available/latency.space /etc/nginx/sites-enabled/
fi

# Remove default if it exists
if [ -f "/etc/nginx/sites-enabled/default" ]; then
  rm -f /etc/nginx/sites-enabled/default
fi

# Create directory for main site and ensure permissions
mkdir -p /var/www/html/latency-space
chmod 755 /var/www/html/latency-space

# Create a comprehensive landing page
echo "Creating landing page at /var/www/html/latency-space/index.html"

# Copy the comprehensive static index.html if available
if [ -f "deploy/static/index.html" ]; then
  cp deploy/static/index.html /var/www/html/latency-space/index.html
  echo "✅ Copied comprehensive index.html to web root"
else
  # Fall back to creating a simple landing page
  cat > /var/www/html/latency-space/index.html << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Latency.Space - Interplanetary Latency Simulation</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, "Open Sans", "Helvetica Neue", sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f9f9f9;
        }
        h1 {
            color: #2c3e50;
            border-bottom: 2px solid #3498db;
            padding-bottom: 10px;
        }
        h2 {
            color: #2980b9;
        }
        code, pre {
            background-color: #f1f1f1;
            padding: 2px 5px;
            border-radius: 3px;
            font-family: 'Courier New', monospace;
        }
        .planet {
            display: inline-block;
            width: 20px;
            height: 20px;
            border-radius: 50%;
            margin-right: 5px;
            vertical-align: middle;
        }
        .mercury { background-color: #ada8a5; }
        .venus { background-color: #e8cda2; }
        .earth { background-color: #6b93d6; }
        .mars { background-color: #c1440e; }
        .jupiter { background-color: #e0ae6f; }
        .saturn { background-color: #d9bf77; }
        .uranus { background-color: #82b3d1; }
        .neptune { background-color: #77c2e0; }
        .latency-table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        .latency-table th, .latency-table td {
            border: 1px solid #ddd;
            padding: 8px;
            text-align: left;
        }
        .latency-table th {
            background-color: #f2f2f2;
        }
        .latency-table tr:nth-child(even) {
            background-color: #f9f9f9;
        }
        .warning {
            background-color: #ffe0e0;
            padding: 10px;
            border-left: 4px solid #ff5555;
            margin: 20px 0;
        }
    </style>
</head>
<body>
    <h1>Latency.Space - Interplanetary Network Simulation</h1>
    
    <p>Welcome to Latency.Space, an educational tool that simulates the network latency of communicating across the solar system.</p>
    
    <div class="warning">
        <strong>Note:</strong> This service is for educational purposes only. To prevent abuse, we've implemented rate limiting and only allow connections to major websites.
    </div>
    
    <h2>How to Use</h2>
    
    <p>You can simulate latency by using one of our celestial body proxies:</p>
    
    <h3>HTTP Proxy</h3>
    <p>Visit any celestial body subdomain to experience that body's latency:</p>
    <ul>
        <li><span class="planet earth"></span> <a href="http://earth.latency.space">earth.latency.space</a> (minimal latency)</li>
        <li><span class="planet mars"></span> <a href="http://mars.latency.space">mars.latency.space</a> (~3-22 minutes)</li>
        <li><span class="planet jupiter"></span> <a href="http://jupiter.latency.space">jupiter.latency.space</a> (~35-52 minutes)</li>
    </ul>
    
    <p>Or use our special DNS-style routing to access specific sites:</p>
    <ul>
        <li><a href="http://www.example.com.mars.latency.space">www.example.com.mars.latency.space</a> - Access example.com with Mars latency</li>
        <li><a href="http://www.google.com.jupiter.latency.space">www.google.com.jupiter.latency.space</a> - Access Google with Jupiter latency</li>
    </ul>
    
    <h3>SOCKS5 Proxy</h3>
    <p>Configure your applications to use our SOCKS5 proxy:</p>
    <pre>Host: mars.latency.space
Port: 1080</pre>
    
    <h2>Latency Table</h2>
    
    <table class="latency-table">
        <tr>
            <th>Celestial Body</th>
            <th>One-way Latency</th>
            <th>Round-trip Latency</th>
        </tr>
        <tr>
            <td><span class="planet earth"></span> Earth</td>
            <td>0 seconds</td>
            <td>0 seconds</td>
        </tr>
        <tr>
            <td><span class="planet mercury"></span> Mercury</td>
            <td>~3-7 minutes</td>
            <td>~6-14 minutes</td>
        </tr>
        <tr>
            <td><span class="planet venus"></span> Venus</td>
            <td>~2-7 minutes</td>
            <td>~4-14 minutes</td>
        </tr>
        <tr>
            <td><span class="planet mars"></span> Mars</td>
            <td>~3-22 minutes</td>
            <td>~6-44 minutes</td>
        </tr>
        <tr>
            <td><span class="planet jupiter"></span> Jupiter</td>
            <td>~35-52 minutes</td>
            <td>~70-104 minutes</td>
        </tr>
        <tr>
            <td><span class="planet saturn"></span> Saturn</td>
            <td>~1.0-1.5 hours</td>
            <td>~2.0-3.0 hours</td>
        </tr>
        <tr>
            <td><span class="planet uranus"></span> Uranus</td>
            <td>~2.5 hours</td>
            <td>~5 hours</td>
        </tr>
        <tr>
            <td><span class="planet neptune"></span> Neptune</td>
            <td>~4.1 hours</td>
            <td>~8.2 hours</td>
        </tr>
    </table>
    
    <h2>Educational Purpose</h2>
    
    <p>This service helps to demonstrate the challenges of interplanetary communication. The latencies experienced here are based on the actual distance between planets and the speed of light, which is the physical limit for communication across space.</p>
    
    <p>Consider how these latencies would affect different types of applications:</p>
    <ul>
        <li>Web browsing becomes impractical beyond Mars</li>
        <li>Real-time video calls are impossible even to the Moon</li>
        <li>Command and control of distant spacecraft requires autonomous systems</li>
    </ul>
    
    <h2>Technical Details</h2>
    
    <p>This service simulates latency by delaying network traffic based on the distance between Earth and the selected celestial body. The latency is calculated using the speed of light (299,792,458 meters per second) and the current distance to each body.</p>
    
    <footer style="margin-top: 40px; border-top: 1px solid #ddd; padding-top: 20px; font-size: 0.8em; color: #666;">
        <p>Latency.Space is an educational project. This service is rate-limited and only allows connections to major websites to prevent abuse.</p>
    </footer>
</body>
</html>
EOF

# Set proper permissions on the HTML file
chmod 644 /var/www/html/latency-space/index.html
echo "Landing page created successfully"

# Create directory for Let's Encrypt validation
mkdir -p /var/www/html/.well-known/acme-challenge

# Test Nginx configuration
echo "🔍 Testing Nginx configuration..."
if nginx -t; then
  echo "✅ Configuration is valid"
  echo "🔄 Reloading Nginx..."
  systemctl reload nginx
  echo "✅ Nginx reloaded successfully"
else
  echo "❌ Configuration is invalid"
  exit 1
fi

echo "🌐 Nginx is now configured to:"
echo "  - Serve the main website at latency.space and www.latency.space"
echo "  - Proxy all celestial-body subdomains to http://proxy:80"
echo "  - Proxy status.latency.space to http://status:3000"
echo ""
echo "You should now be able to access the main website and all subdomain proxies."