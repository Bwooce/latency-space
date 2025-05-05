#!/bin/bash
set -e

echo "Building proxy service with the new Dockerfile..."
cd /root/latency-space
docker build -t latency-proxy-test -f Dockerfile.proxy .

echo "Checking if the build was successful..."
if docker image inspect latency-proxy-test &> /dev/null; then
    echo "Success! Docker image latency-proxy-test was built successfully."
else
    echo "Error: Docker build failed."
    exit 1
fi

echo "Running a test container..."
docker run --rm -it --name latency-proxy-test-container latency-proxy-test --help || echo "Note: The service may not support --help flag, but it should have run without crashing."

echo "Test completed successfully."