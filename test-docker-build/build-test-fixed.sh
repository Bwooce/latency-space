#!/bin/bash
set -ex

echo "Building proxy service with the updated Dockerfile..."
cd /root/latency-space
docker build -t latency-proxy-test-fixed -f Dockerfile.proxy .

echo "Checking if the build was successful..."
if docker image inspect latency-proxy-test-fixed &> /dev/null; then
    echo "SUCCESS! Docker image latency-proxy-test-fixed was built successfully."
else
    echo "ERROR: Docker build failed."
    exit 1
fi

# Clean up any old container if it exists
docker rm -f latency-proxy-test-container 2>/dev/null || true

echo "Running a test container to verify it starts correctly..."
docker run -d --name latency-proxy-test-container latency-proxy-test-fixed || {
    echo "Failed to start container"
    exit 1
}

# Give it a moment to initialize
sleep 2

# Check if container is running
if docker ps --filter "name=latency-proxy-test-container" --filter "status=running" | grep latency-proxy-test-container; then
    echo "Container is running successfully."
else
    echo "Container failed to start or stopped immediately."
    docker logs latency-proxy-test-container
    exit 1
fi

# Clean up
docker stop latency-proxy-test-container
docker rm latency-proxy-test-container

echo "Test completed successfully!"