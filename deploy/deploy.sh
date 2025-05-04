#!/bin/bash
# Simple deployment script for latency.space

# Display usage if parameters are not provided
if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <username> <host> [ssh_key_file]"
  echo "Example: $0 ubuntu 192.168.1.100 ~/.ssh/id_rsa"
  exit 1
fi

USERNAME=$1
HOST=$2
SSH_KEY=${3:-"~/.ssh/id_rsa"}  # Default to ~/.ssh/id_rsa if not provided

echo "Starting deployment to $USERNAME@$HOST..."

# Run deployment commands via SSH
ssh -i $SSH_KEY $USERNAME@$HOST << 'ENDSSH'
set -e  # Exit on any error

echo "Connected to server. Starting deployment..."

# Check if the directory exists, if not create it and clone the repository
if [ ! -d "/opt/latency-space" ]; then
  echo "Creating deployment directory..."
  sudo mkdir -p /opt/latency-space
  sudo chown $(whoami):$(whoami) /opt/latency-space
  git clone https://github.com/Bwooce/latency-space.git /opt/latency-space
fi

# Navigate to the deployment directory
cd /opt/latency-space

# Check DNS resolution
echo "Testing DNS resolution..."
if ! ping -c 1 github.com &> /dev/null; then
  echo "DNS resolution issue - fixing..."
  echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf > /dev/null
  echo "nameserver 1.1.1.1" | sudo tee -a /etc/resolv.conf > /dev/null
fi

# Update repository
echo "Updating code from repository..."
git fetch --all
git reset --hard origin/main

# Ensure ACME challenge directory exists and has correct permissions
echo "Ensuring ACME challenge directory exists and has correct permissions..."
sudo mkdir -p /var/www/html/.well-known/acme-challenge
# Note: Using 777 is simple but potentially insecure.
# Consider adjusting ownership/group if the container runs as non-root.
sudo chmod 777 /var/www/html/.well-known/acme-challenge
echo "ACME challenge directory setup complete."

# Check if docker and docker-compose are installed
if ! command -v docker &> /dev/null; then
  echo "Docker not found. Please install Docker first."
  exit 1
fi

# We're using the standard Docker installation, not the snap version
# The code for snap Docker configuration has been removed as we're 
# using the more stable package-based Docker installation

# Deploy using Docker Compose
echo "Deploying with Docker Compose..."
if command -v docker compose &> /dev/null; then
  # Docker Compose v2
  docker compose down
  docker compose pull
  docker compose up -d --force-recreate
elif command -v docker-compose &> /dev/null; then
  # Legacy Docker Compose
  docker-compose down
  docker-compose pull
  docker-compose up -d --force-recreate
else
  echo "Neither docker compose nor docker-compose found. Please install Docker Compose."
  exit 1
fi

# Update Nginx config with container IPs
echo "Updating Nginx configuration with container IPs..."
if sudo /opt/latency-space/deploy/update-nginx.sh; then
  echo "Nginx configuration updated successfully."
else
  echo "WARNING: Failed to update Nginx configuration."
  # Optionally add further error handling here if needed
fi

# Verify deployment
echo "Verifying services..."
sleep 10  # Wait for services to start

if docker compose ps 2>/dev/null | grep -q "Up" || docker-compose ps 2>/dev/null | grep -q "Up"; then
  echo "Services are running. Deployment successful!"
else
  echo "WARNING: Services might not be running correctly."
  docker compose logs 2>/dev/null || docker-compose logs
fi

ENDSSH

# Check if the SSH command was successful
if [ $? -eq 0 ]; then
  echo "Deployment completed successfully!"
else
  echo "Deployment failed with exit code $?."
  exit 1
fi