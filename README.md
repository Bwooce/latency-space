# latency.space

Simulate interplanetary network latency across the solar system with real-time orbital calculations.

## Quick Start

**Prerequisites:**
- Docker: [https://docs.docker.com/get-docker/](https://docs.docker.com/get-docker/)
- Docker Compose: [https://docs.docker.com/compose/install/](https://docs.docker.com/compose/install/) (Included with Docker Desktop, but may require a separate install on Linux)

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

## Available Endpoints

### HTTP Proxy Endpoints

Access websites with planetary latency:

- `mars.latency.space` - Mars latency
- `jupiter.latency.space` - Jupiter latency
- `earth.latency.space` - Earth latency (minimal, useful for baseline)
- etc. (any celestial body defined in the configuration)

### SOCKS5 Proxy

Connect to latency.space as a SOCKS5 proxy on port 1080:

```
# Example with curl
curl --socks5 mars.latency.space:1080 https://example.com

# Configure browser to use the SOCKS5 proxy
Host: mars.latency.space
Port: 1080
```

### Special DNS-style Routing

You can also proxy any domain through specific celestial bodies:

- www.google.com.mars.latency.space - Access Google through Mars
- api.github.com.jupiter.latency.space - Access GitHub API through Jupiter
- example.com.moon.earth.latency.space - Access example.com through Earth's Moon

This works with both HTTP and SOCKS5 proxies.

 **Important Note on SSL Certificates:**
 - First-level subdomains (mars.latency.space) support HTTPS with valid certificates.
- Multi-level subdomains (e.g., `www.google.com.mars.latency.space`) work over **HTTP only**.
   - This is a limitation of standard wildcard SSL certificates (`*.latency.space`), which do not cover multiple subdomain levels. HTTPS connections to these multi-level domains will fail certificate validation.

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

 Full documentation is available at [docs.latency.space](https://docs.latency.space) *(Note: This documentation link may be outdated or inactive.)*
