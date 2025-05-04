# DNS and SSL Configuration Guide for latency.space

This document outlines the DNS configuration, domain handling, and SSL certificate setup for the latency.space project.

## Domain Naming Standards

For consistent SSL certificate validation and user experience, the system enforces these domain name standards:

1. **All domains must be lowercase**
   - All domain parts are automatically converted to lowercase
   - This ensures consistent SSL certificate validation
   - Examples: `mars.latency.space`, `phobos.mars.latency.space`

2. **Moon domains follow the parent-child format**
   - Format: `moonname.planetname.latency.space`
   - Example: `phobos.mars.latency.space`, `europa.jupiter.latency.space`

3. **Spacecraft and other objects use hyphenated names**
   - Spaces in names are replaced with hyphens
   - Example: `voyager-1.latency.space`, `james-webb.latency.space`
   - No underscores are used in domain names (DNS best practice)

## Running the DNS and SSL Setup

The setup script handles registering all domains with Cloudflare and can automatically manage SSL certificates:

```bash
# First, build the tools
cd /opt/latency-space/tools
go build

# Option 1: Run the setup_dns tool with DNS configuration only
./tools -token YOUR_API_TOKEN -ip YOUR_SERVER_IP

# Option 2: Run with both DNS configuration and automatic SSL certificate management
./tools -token YOUR_API_TOKEN -ip YOUR_SERVER_IP -ssl
```

The script will:
1. Generate domains for all celestial objects
2. Validate all domains are lowercase
3. Group domains by level (root, single-level, multi-level)
4. Create or update DNS records in Cloudflare
5. If the `-ssl` flag is provided:
   - Check for existing certificates and their expiration
   - Install certbot if not already present
   - Obtain or renew SSL certificates for all domains
   - Configure Nginx to use the certificates (when using the nginx plugin)

## SSL Certificate Configuration

The system requires a certificate that covers:
- The root domain (`latency.space`)
- Single-level subdomains (`*.latency.space`)
- Multi-level subdomains (`*.*.latency.space`)

For automatic SSL certificate management, use the `-ssl` flag with the setup tool:

```bash
./tools -token YOUR_API_TOKEN -ip YOUR_SERVER_IP -ssl
```

Alternatively, to manually create the proper SSL certificate:

```bash
# Stop Nginx temporarily to free port 80
sudo systemctl stop nginx

# Obtain the wildcard certificate with Let's Encrypt
sudo certbot certonly --standalone \
  -d latency.space \
  -d *.latency.space \
  -d *.*.latency.space

# Start Nginx again
sudo systemctl start nginx

# Update Nginx configuration to use the new certificate
sudo ./deploy/update-nginx.sh
```

## Troubleshooting Domain Issues

If you experience SSL certificate errors or domain resolution problems:

### SSL Certificate Validation Errors

1. **Incorrect Case in URLs**
   - Check if you're accessing the domain with uppercase letters
   - Example: `Mars.latency.space` should be `mars.latency.space`

2. **Multi-level Domain Certificate Coverage**
   - Verify your certificate covers `*.*.latency.space`
   - Use: `sudo certbot certificates` to check coverage

3. **Certificate Renewal**
   - Ensure renewals include all required domain levels
   - Use the same domain parameters from the original certificate

### Domain Resolution Issues

1. **DNS Propagation**
   - Changes may take up to 24 hours to propagate globally
   - Use `dig` or `nslookup` to check current DNS resolution

2. **Cloudflare Proxy Settings**
   - Root domain and www should be proxied (orange cloud)
   - All celestial subdomains should be DNS only (gray cloud)
   - This is critical for proper latency simulation

## Testing Domain Configuration

Test all levels of domains to ensure proper configuration:

```bash
# Test root domain
curl -I https://latency.space

# Test planet domain
curl -I https://mars.latency.space

# Test moon domain
curl -I https://phobos.mars.latency.space

# Test spacecraft domain
curl -I https://voyager-1.latency.space
```

## Domain Format Reference

Here's a complete reference of domain formats supported by the system:

1. **Root Domain**
   - `latency.space` - Main site
   - `www.latency.space` - Alias to main site

2. **Service Domains**
   - Status dashboard is now integrated with the main site

3. **Celestial Body Domains**
   - Planets: `mars.latency.space`, `jupiter.latency.space`, etc.
   - Moons: `moon.earth.latency.space`, `phobos.mars.latency.space`, etc.
   - Dwarf Planets: `pluto.latency.space`, `ceres.latency.space`, etc.
   - Asteroids: `vesta.latency.space`, etc.
   - Spacecraft: `voyager-1.latency.space`, `perseverance.latency.space`, etc.

4. **Proxy-Through Domains**
   - Format: `target-site.celestial-body.latency.space`
   - Example: `example.com.mars.latency.space`
   - Multi-level: `example.com.phobos.mars.latency.space`