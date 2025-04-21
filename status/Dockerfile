# Build stage
FROM node:18-alpine as builder

WORKDIR /app

# Install dependencies
COPY package*.json ./
RUN npm install

# Copy source code
COPY . .

# Build the app
RUN npm run build

# Production stage
FROM nginx:alpine

# Copy built assets from builder stage
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy backup index.html and assets for resilience
COPY backup-index.html /usr/share/nginx/html/index.html.backup
COPY test-metrics.html /usr/share/nginx/html/test-metrics.html

# Create assets directory and ensure proper permissions
RUN mkdir -p /usr/share/nginx/html/assets && \
    chown -R nginx:nginx /usr/share/nginx/html

# Copy startup script and make executable
COPY start.sh /start.sh
RUN chmod +x /start.sh

# Keep the nginx.conf as a reference, but it won't be used directly
COPY nginx.conf /etc/nginx/conf.d/default.conf.template

# Add environment variables for service IPs
ENV PROMETHEUS_IP="172.18.0.3"
ENV PROXY_IP="172.18.0.2"

EXPOSE 80

# Use the startup script that will generate the configuration with the correct IPs
CMD ["/start.sh"]

