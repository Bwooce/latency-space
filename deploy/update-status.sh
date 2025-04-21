#!/bin/bash
# Script to update the status container with the latest changes

set -e

# Define colors for better output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Updating status container...${NC}"

# Get the current directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$( cd "$SCRIPT_DIR/.." && pwd )"

# Make sure docker is running
if ! docker info > /dev/null 2>&1; then
  echo -e "${RED}Error: Docker is not running or not accessible${NC}"
  echo "Please start Docker and try again"
  exit 1
fi

# Get current container IP addresses to preserve them
echo -e "${YELLOW}Getting current container IPs...${NC}"
PROMETHEUS_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' latency-space_prometheus_1 2>/dev/null || echo "172.18.0.3")
PROXY_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' latency-space_proxy_1 2>/dev/null || echo "172.18.0.2")

echo -e "${GREEN}Found Prometheus IP: ${PROMETHEUS_IP}${NC}"
echo -e "${GREEN}Found Proxy IP: ${PROXY_IP}${NC}"

# Rebuild the status container
echo -e "${YELLOW}Rebuilding status container...${NC}"
cd "$PROJECT_DIR"
docker compose build --no-cache status

# Restart only the status container
echo -e "${YELLOW}Restarting status container...${NC}"
docker compose stop status
docker compose up -d status

# Verify the status container is running
echo -e "${YELLOW}Verifying status container...${NC}"
if docker ps | grep -q "latency-space_status"; then
  echo -e "${GREEN}Status container is running!${NC}"
  
  # Get the container logs
  echo -e "${YELLOW}Status container logs:${NC}"
  docker logs latency-space_status_1 --tail 20
  
  # Get IP address
  STATUS_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' latency-space_status_1)
  echo -e "${GREEN}Status container IP: ${STATUS_IP}${NC}"
  
  echo -e "${GREEN}You can access the status dashboard at http://status.latency.space${NC}"
  echo -e "${GREEN}Test the metrics endpoint at http://status.latency.space/test-metrics.html${NC}"
else
  echo -e "${RED}Error: Status container failed to start${NC}"
  docker compose logs status
  exit 1
fi

echo -e "${GREEN}Status container update complete!${NC}"