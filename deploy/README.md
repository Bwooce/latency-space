# Deployment Instructions

This directory contains scripts and configuration for deploying Latency Space.

## Manual Deployment

If automatic GitHub Action deployments are failing, you can deploy manually using one of these methods:

### Method 1: Direct SSH Deployment

```bash
# SSH into your server
ssh your-username@your-server-ip

# Navigate to or create the deployment directory
cd /opt/latency-space || mkdir -p /opt/latency-space

# Clone the repository if it doesn't exist
if [ ! -d .git ]; then
  git clone https://github.com/Bwooce/latency-space.git .
else
  git pull
fi

# Deploy with Docker Compose
docker compose down
docker compose pull
docker compose up -d
```

### Method 2: Using the deploy.sh Script

```bash
# From your local machine
./deploy/deploy.sh your-username your-server-ip
```

### Method 3: GitHub Action Manual Dispatch

1. Go to GitHub repository Actions tab
2. Select the "Manual Deploy" workflow
3. Click "Run workflow"
4. Enter your server details if secrets aren't configured

## Setting up GitHub Secrets for Automatic Deployment

For automatic deployment via GitHub Actions, you need to configure these secrets:

1. Go to your GitHub repository
2. Navigate to Settings > Secrets and variables > Actions
3. Add the following secrets:

- `DEPLOY_HOST`: Your server's IP address or hostname
- `DEPLOY_USER`: SSH username with deployment permissions
- `SSH_PRIVATE_KEY`: The private SSH key for authentication (the full key content)

## Troubleshooting Deployment Issues

### Common Issues

1. **DNS Resolution Problems**:
   - The server might have DNS issues
   - Fix by adding Google DNS to /etc/resolv.conf:
     ```
     echo "nameserver 8.8.8.8" > /etc/resolv.conf
     echo "nameserver 1.1.1.1" >> /etc/resolv.conf
     ```

2. **Docker Compose Not Found**:
   - Try both `docker compose` and `docker-compose` commands
   - Install if missing: `apt-get install docker-compose-plugin`

3. **SSH Connection Issues**:
   - Verify SSH key has correct permissions (600)
   - Ensure the server accepts key-based authentication
   - Test connection with: `ssh -i your_key your-username@your-server-ip`

4. **Permission Errors**:
   - Ensure the deployment user has sudo permissions if needed
   - Check file ownership: `chown -R your-username:your-username /opt/latency-space`