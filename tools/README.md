# DNS and SSL Management Tool

This tool automatically configures DNS records for latency.space and its subdomains using the Cloudflare API. It can also manage SSL certificates using certbot.

## Prerequisites

Before running the tool, you need:

1. A Cloudflare API token with Zone:Edit permissions for the latency.space domain
2. The server's public IP address
3. Proper setup of required Go files

## Setup

Ensure that the necessary files are linked properly:

```bash
# Create symlinks to required files from proxy/src
ln -fs ../proxy/src/calculations.go calculations.go
ln -fs ../proxy/src/models.go models.go
ln -fs ../proxy/src/objects_data.go objects_data.go  # Only if file exists

# Build the tool
go build
```

## Usage

```bash
# Configure DNS records only
./setup_dns -token YOUR_CLOUDFLARE_API_TOKEN -ip YOUR_SERVER_IP

# Configure DNS records and automatically manage SSL certificates
./setup_dns -token YOUR_CLOUDFLARE_API_TOKEN -ip YOUR_SERVER_IP -ssl
```

## Options

- `-token`: Cloudflare API token (required)
- `-ip`: Server IP address for DNS records (required)
- `-zone`: Domain zone name (default: "latency.space")
- `-ssl`: Enable automatic SSL certificate management with certbot

## SSL Certificate Management

When the `-ssl` flag is provided, the tool will:

1. Check if certbot is installed (install it if missing)
2. Check for existing SSL certificates and their expiration
3. Request or renew certificates as needed
4. Configure Nginx to use the certificates (when possible)

## Troubleshooting

If the tool doesn't find all expected subdomains:

1. Ensure `calculations.go` is properly linked 
2. Check that `Go Get` functions are accessible
3. Rebuild the tool using `go build`

For DNS or certificate issues, check the Cloudflare API token permissions and certbot logs.