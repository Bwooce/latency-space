#!/bin/bash
set -ex

echo "Testing final Dockerfile.proxy build..."
cd /root/latency-space
docker build -t latency-proxy-final -f Dockerfile.proxy .

echo "Checking if the build was successful..."
if docker image inspect latency-proxy-final &> /dev/null; then
    echo "SUCCESS! Docker image was built successfully."
else
    echo "ERROR: Docker build failed."
    exit 1
fi

echo "Running a test container to make sure it starts properly..."
docker run --rm -d --name latency-proxy-final-test latency-proxy-final

# Wait a moment for container to initialize
sleep 2

# Check if container is running
if docker ps | grep latency-proxy-final-test; then
    echo "Container is running successfully!"
    docker stop latency-proxy-final-test
else
    echo "Container failed to start or stopped immediately."
    docker logs latency-proxy-final-test || true
    exit 1
fi

echo "Test completed successfully!"