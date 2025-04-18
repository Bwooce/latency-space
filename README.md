# latency.space

Simulate interplanetary network latency across the solar system with real-time orbital calculations.

## Quick Start

1. Clone this repository
2. Copy `config/example.env` to `.env` and configure
3. Run `docker compose up -d`

## GitHub Actions Setup

To enable CI/CD with GitHub Actions for deployment, you need to configure the following secrets in your GitHub repository:

1. Go to your repository on GitHub
2. Navigate to Settings > Secrets and variables > Actions
3. Add the following secrets:

- `DEPLOY_HOST`: Your server's IP address or hostname
- `DEPLOY_USER`: SSH username for deployment
- `SSH_PRIVATE_KEY`: Private SSH key for authentication

Optional:
- `CLOUDFLARE_API_TOKEN`: If using Cloudflare for DNS, add your API token here

## Available Endpoints

### HTTP Proxy Endpoints

Access websites with planetary latency:

- mars.latency.space - Mars latency
- jupiter.latency.space - Jupiter latency
- [other celestial bodies]

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

**Important SSL Certificate Note:**
- First-level subdomains (mars.latency.space) support HTTPS with valid certificates
- Multi-level subdomains (www.google.com.mars.latency.space) work over HTTP only
  - This is because wildcard SSL certificates only cover one level of subdomains

## Monitoring

- Status page: http://localhost:3000
- Prometheus: http://localhost:9092
- Grafana: http://localhost:3002 (admin/admin by default)

See full documentation at [docs.latency.space](https://docs.latency.space)