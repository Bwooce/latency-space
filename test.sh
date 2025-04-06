#!/bin/bash
# test.sh - Test the proxy in various modes

set -e

echo "🚀 Testing latency.space proxy..."

# Build the test client if needed
echo "Building test client..."
cd proxy/src
go build -o test_socks test_socks.go

# Check if server is running
if ! curl -s http://localhost:80 > /dev/null; then
  echo "Starting the proxy..."
  cd ../../
  docker-compose up -d
  sleep 5  # Wait for services to start
else
  cd ../../
fi

# Test HTTP proxy through different celestial bodies
echo -e "\n🌍 Testing Earth HTTP proxy..."
time curl -s -o /dev/null http://earth.latency.space

echo -e "\n🔴 Testing Mars HTTP proxy..."
time curl -s -o /dev/null http://mars.latency.space

echo -e "\n🌌 Testing Jupiter HTTP proxy..."
time curl -s -o /dev/null http://jupiter.latency.space

# Test domain format with HTTP
echo -e "\n🌐 Testing domain.body.latency.space format..."
time curl -s -o /dev/null http://www.example.com.mars.latency.space

# Test SOCKS proxy (requires test_socks binary)
echo -e "\n🧦 Testing SOCKS5 proxy through Earth..."
./proxy/src/test_socks localhost:1080 http://example.com

echo -e "\n🧦 Testing SOCKS5 proxy with DNS format..."
# This test uses www.example.com.mars.latency.space SOCKS proxy format
echo "This test requires manual verification with a browser or client that supports SOCKS5"
echo "Configure your browser to use localhost:1080 as SOCKS5 proxy"
echo "Then visit: http://www.example.com.mars.latency.space"

echo -e "\n✅ All tests completed!"