# latency.space

Simulate interplanetary network latency across the solar system.

## Quick Start

1. Clone this repository
2. Copy `config/example.env` to `.env` and configure
3. Run `docker-compose up -d`

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

This works with both HTTP(S) and SOCKS5 proxies.

## Monitoring

- Status page: http://localhost:3000
- Prometheus: http://localhost:9091
- Grafana: http://localhost:3001 (admin/admin by default)

See full documentation at [docs.latency.space](https://docs.latency.space)