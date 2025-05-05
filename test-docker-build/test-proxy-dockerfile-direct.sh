#!/bin/bash
set -ex

echo "Building directly from proxy/Dockerfile..."

# Move to the proxy directory so COPY statements using relative paths work
cd /root/latency-space/proxy

# Build using the updated Dockerfile
docker build -t latency-proxy-direct .

echo "Checking if the build was successful..."
if docker image inspect latency-proxy-direct &> /dev/null; then
    echo "SUCCESS! Docker image latency-proxy-direct was built successfully."
else
    echo "ERROR: Docker build failed."
    exit 1
fi

# Run the container to verify it works
echo "Running container to verify it starts correctly..."
docker run --rm -d --name latency-proxy-direct-test latency-proxy-direct

# Give it a moment to initialize
sleep 2

# Check if container is running
if docker ps | grep latency-proxy-direct-test; then
    echo "Container is running successfully!"
    docker stop latency-proxy-direct-test
else
    echo "Container failed to start or exited immediately."
    docker logs latency-proxy-direct-test || true
    exit 1
fi

echo "Test completed successfully!"