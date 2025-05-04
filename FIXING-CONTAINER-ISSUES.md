# Fixing Container and Website Issues

This document outlines the steps to fix the issues with the containers and website.

## Current Issues

1. **Template Loading Error in Proxy Container**:
   - The proxy container is failing with: `Failed to parse info page template: open proxy/src/templates/info_page.html: no such file or directory`
   - This causes the proxy container to constantly restart

2. **Missing `/var/www/html` Mount**:
   - The Docker compose file attempts to mount `/var/www/html` which doesn't exist or has permission issues

3. **Bad Gateway Errors**:
   - The planetary subdomains (venus.latency.space, etc.) show 502 Bad Gateway errors
   - The status dashboard also shows a 502 Bad Gateway error

## Fix Steps

### 1. Update the Dockerfile for the Proxy Container

Edit `/opt/latency-space/proxy/Dockerfile` to correctly copy template files:

```diff
# Copy and prepare binary
COPY --from=builder /latency-proxy /usr/local/bin/latency-proxy
RUN chmod +x /usr/local/bin/latency-proxy

+ # Copy templates
+ COPY src/templates/ /app/templates/

# Expose ports
EXPOSE 80 443 1080 9090
```

### 2. Update the Template Loading in Main.go

Edit `/opt/latency-space/proxy/src/main.go` to check for templates in multiple locations:

```diff
// Parse the info page template at startup
var err error
- infoTemplate, err = template.ParseFiles("proxy/src/templates/info_page.html")
- if err != nil {
-   log.Fatalf("Failed to parse info page template: %v", err)
- }
+ // Try different paths for the template (container paths first, then local development paths)
+ templatePaths := []string{
+   "/app/templates/info_page.html",            // Docker container path (new)
+   "templates/info_page.html",                 // Relative path
+   "src/templates/info_page.html",             // Another relative path
+   "proxy/src/templates/info_page.html",       // Original path
+ }
+ 
+ var templateErr error
+ for _, path := range templatePaths {
+   infoTemplate, templateErr = template.ParseFiles(path)
+   if templateErr == nil {
+     log.Printf("Successfully loaded template from: %s", path)
+     break
+   }
+ }
+ 
+ if infoTemplate == nil {
+   log.Fatalf("Failed to parse info page template: %v", templateErr)
+ }
```

### 3. Fix the Missing Mount Issue in Docker Compose

Edit `/opt/latency-space/docker-compose.yml` to handle the missing `/var/www/html` directory:

Option 1: Create the directory (recommended):
```bash
sudo mkdir -p /var/www/html/.well-known/acme-challenge
sudo chmod -R 777 /var/www/html
```

Option 2: Temporarily comment out the problematic mount:
```diff
volumes:
  - proxy_config:/etc/space-proxy
  - proxy_ssl:/etc/letsencrypt
  - proxy_certs:/app/certs # For certificate persistence
- - type: bind # Add the new volume mapping for acme challenge
-   source: /var/www/html
-   target: /var/www/html
+ # Temporarily commenting out the problematic bind mount
+ # - type: bind # Add the new volume mapping for acme challenge
+ #   source: /var/www/html
+ #   target: /var/www/html
```

### 4. Restart Docker Service and Containers

```bash
# Make sure only one Docker service is running (either snap or apt, not both)
# For snap-installed Docker:
sudo snap restart docker
sudo sleep 10

# Rebuild and restart containers
cd /opt/latency-space
docker compose down
docker compose build proxy
docker compose up -d
```

### 5. Verify the Fix

```bash
# Check container status
docker ps

# Check proxy container logs
docker logs latency-space-proxy-1

# Test the websites
curl -I https://latency.space
curl -I https://status.latency.space
curl -I https://venus.latency.space
```

## Long-term Recommendations

1. **Update the deployment scripts** to create necessary directories like `/var/www/html` before starting containers
2. **Add proper error handling** in the Go code for template loading
3. **Add health checks** to docker-compose.yml to detect and restart failing containers properly
4. **Review Docker installation** to ensure only one Docker installation (either snap or apt) is active