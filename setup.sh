#!/bin/bash
# setup.sh - Run this first

# Create directory structure
mkdir -p {.github/workflows,proxy,metrics,status,config,deploy}
mkdir -p proxy/{src,config}
mkdir -p metrics/{src,config}
mkdir -p status/{src,public}
mkdir -p config/{nginx,ssl}

# Create basic README
cat > README.md << 'EOF'
# latency.space

Simulate interplanetary network latency across the solar system.

## Quick Start

1. Clone this repository
2. Copy `config/example.env` to `.env` and configure
3. Run `docker compose up -d`

## Available Endpoints

- mars.latency.space - Mars latency
- jupiter.latency.space - Jupiter latency
- [etc...]

See full documentation at [docs.latency.space](https://docs.latency.space)
EOF

# Create gitignore
cat > .gitignore << 'EOF'
.env
*.log
.DS_Store
node_modules/
dist/
*.pem
.idea/
.vscode/
*.swp
EOF

# Create example env file
cat > config/example.env << 'EOF'
DEPLOY_HOST=your-vps-ip
DEPLOY_USER=root
DISCORD_WEBHOOK=your-discord-webhook
SSL_EMAIL=your-email@domain.com
GRAFANA_PASSWORD=your-grafana-password
EOF

echo "Directory structure created successfully!"

