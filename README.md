# latency.space

Simulate interplanetary network latency across the solar system with real-time orbital calculations.

## Quick Start

**Prerequisites:**
- Docker: [https://docs.docker.com/get-docker/](https://docs.docker.com/get-docker/)
- Docker Compose: [https://docs.docker.com/compose/install/](https://docs.docker.com/compose/install/) (Included with Docker Desktop, but may require a separate install on Linux)

**Important Note for Snap-installed Docker:**
If you're using Docker installed via Snap on Ubuntu/Debian, you'll need to set privileged mode:
```bash
sudo snap set docker privileged=true
sudo snap restart docker
```
This addresses AppArmor permission denied errors that can occur with volume mounts.

**Steps:**

1.  **Clone this repository:**
    ```bash
    git clone https://github.com/username/repo-name.git # Replace with the appropriate repository URL
    cd repo-name
    ```
2.  **Configure Environment:**
    - Copy the example configuration file:
      ```bash
      cp config/example.env .env
      ```
     - **Edit the `.env` file** with your specific settings. You need to provide:
      - `SSL_EMAIL`: Your email address for Let's Encrypt SSL certificate generation.
      - `GRAFANA_PASSWORD`: A secure password for the Grafana admin user.
      - (Optional) `DEPLOY_HOST`, `DEPLOY_USER`, `DISCORD_WEBHOOK` if needed for deployment or notifications.
3.  **Start the Services:**
    Run the following command from the project's root directory:
    ```bash
    docker compose up -d
    ```
    *(Note: If you have an older version, you might need to use `docker-compose up -d`)*

     This will build the necessary Docker images (if they haven't been built already) and start all the services defined in `docker-compose.yml` in detached mode.

4.  **Access the Services:**
    Once the containers are running, you can access the various endpoints and monitoring tools as described in the sections below.

## GitHub Actions Setup

To enable CI/CD with GitHub Actions for deployment, you need to configure the following secrets in your GitHub repository:

1. Go to your repository on GitHub
 2. Navigate to Settings > Secrets and Variables > Actions
3. Add the following secrets:

- `DEPLOY_HOST`: Your server's IP address or hostname
- `DEPLOY_USER`: SSH username for deployment
- `SSH_PRIVATE_KEY`: Private SSH key for authentication

Optional:
 - `CLOUDFLARE_API_TOKEN`: If using Cloudflare for DNS updates in the main deployment workflow (`.github/workflows/main.yml`), add your API token here.

*(Note: The above secrets have been verified against the workflow files in `.github/workflows/` as of the last update.)*

## Deployment

This project includes scripts and configurations to facilitate deployment to a server, primarily managed through Docker Compose. Deployment can be triggered automatically via GitHub Actions (on pushes to the `main` branch see previous section) or performed manually.

The `/deploy` directory contains various shell scripts (`.sh`) used for setting up the server environment, managing the Docker services, troubleshooting common issues (such as DNS resolution or container restarts), and other deployment-related tasks.

For detailed step-by-step instructions on manual deployment, server setup prerequisites, and troubleshooting guidance, please refer to the dedicated README within the deployment directory:

➡️ **[Deployment Guide](./deploy/README.md)**

### Common Container Issues and Solutions

#### Template Loading Issues
If the proxy container fails with template loading errors, ensure the Dockerfile copies templates correctly:
```dockerfile
# Copy templates
COPY src/templates/ /app/templates/
```

#### Volume Mount Problems
If containers fail to start due to volume mount issues:

1. Check if Docker is installed via snap and needs privileged mode:
```bash
sudo snap set docker privileged=true
sudo snap restart docker
```

2. If getting errors about missing directories, create them:
```bash
sudo mkdir -p /var/www/html/.well-known/acme-challenge
sudo chmod -R 777 /var/www/html
```

3. For persistent issues, consider commenting out problematic mounts in docker-compose.yml:
```yaml
# Temporarily commenting out problematic mount
# - type: bind
#   source: /var/www/html
#   target: /var/www/html
```

## Available Endpoints

### HTTP Proxy Endpoints

Access websites with planetary latency:

- `mars.latency.space` - Mars latency
- `jupiter.latency.space` - Jupiter latency
- `earth.latency.space` - Earth latency (minimal, useful for baseline)
- etc. (any celestial body defined in the configuration)

### SOCKS5 Proxy

Connect to latency.space as a SOCKS5 proxy on port 1080 for TCP connections:

```bash
# Example with curl for TCP (HTTPS)
curl --socks5 mars.latency.space:1080 https://example.com

# Configure browser or application to use the SOCKS5 proxy
Host: mars.latency.space
Port: 1080
```

The proxy also supports UDP forwarding via the SOCKS5 `UDP ASSOCIATE` command. When you initiate a UDP association, the proxy sets up a UDP relay. Latency for relayed UDP packets (both outgoing and incoming) is applied based on the celestial body hostname used in the initial SOCKS5 connection (e.g., `mars.latency.space`).

```bash
# Example: Send a UDP packet (e.g., DNS query) to 1.1.1.1:53 via Mars
# Requires a netcat version that supports SOCKS5 UDP (-X 5 and -x)
echo "dns-query-data" | nc -u -X 5 -x mars.latency.space:1080 1.1.1.1 53
```

### Special DNS-style Routing

You can also proxy any domain through specific celestial bodies:

- www.google.com.mars.latency.space - Access Google through Mars
- api.github.com.jupiter.latency.space - Access GitHub API through Jupiter
- example.com.moon.earth.latency.space - Access example.com through Earth's Moon

This works with both HTTP and SOCKS5 proxies (TCP only for SOCKS5 with this method).

 **Important Note on SSL Certificates:**
 - First-level subdomains (mars.latency.space) support HTTPS with valid certificates.
 - Second-level subdomains (e.g., `phobos.mars.latency.space`) require special wildcard certificate configuration.
 - Multi-level proxy-through domains (e.g., `www.google.com.mars.latency.space`) work over **HTTP only**.
   - This is a limitation of standard wildcard SSL certificates.

For proper SSL certificate configuration covering all domain levels:
```bash
sudo certbot certonly --standalone -d latency.space -d *.latency.space -d *.*.latency.space
```

See the [DNS and SSL Configuration Guide](./DNS-AND-SSL-CONFIGURATION.md) for detailed instructions.

### API Endpoint: `/api/status-data`

 Provides real-time data for celestial bodies in JSON format, including distance from Earth, calculated one-way light-travel latency, and occlusion status.

**Example Request:**

```bash
curl http://latency.space/api/status-data
# Or access via a specific body (latency is not added to the API request itself)
curl http://mars.latency.space/api/status-data
```

**Example JSON Response Snippet:**

```json
{
  "timestamp": "2023-10-27T10:00:00Z",
  "objects": {
    "planets": [
      {
        "name": "Mars",
        "type": "planet",
        "distance_km": 225000000,
        "latency_seconds": 750.5,
        "occluded": false
      },
      // ... other planets
    ],
    "moons": [
      {
        "name": "Moon",
        "type": "moon",
        "parentName": "Earth",
        "distance_km": 384400,
        "latency_seconds": 1.28,
        "occluded": false
      },
      // ... other moons
    ]
    // ... other object types (dwarf_planets, etc.)
  }
}
```
 *(Note: The `latency.space` domain used in the `curl` example assumes the service is deployed and publicly accessible at that domain. Replace `latency.space` with your actual domain if running locally or elsewhere.)*

## Monitoring

- Status page: http://localhost:3000
- Prometheus: http://localhost:9092
- Grafana: http://localhost:3002 (Default login: admin / `admin`, or the password set in your `.env` file)

