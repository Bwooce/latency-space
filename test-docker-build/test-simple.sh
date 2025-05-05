#!/bin/bash
set -ex

echo "Building proxy service with the simplified Dockerfile..."
cd /root/latency-space

# Build using the simplified Dockerfile
docker build -t latency-proxy-simple -f Dockerfile.simple .

echo "Checking if the build was successful..."
if docker image inspect latency-proxy-simple &> /dev/null; then
    echo "SUCCESS! Docker image latency-proxy-simple was built successfully."
else
    echo "ERROR: Docker build failed."
    exit 1
fi

# Copy the simplified Dockerfile to Dockerfile.proxy
cp Dockerfile.simple Dockerfile.proxy

echo "Test completed successfully!"