# monitoring/grafana/Dockerfile
FROM grafana/grafana:latest

# Create Grafana directories
USER root
RUN mkdir -p /etc/grafana/provisioning/dashboards

# Copy dashboards and config
COPY dashboards /etc/grafana/provisioning/dashboards
COPY provisioning/dashboards/default.yaml /etc/grafana/provisioning/dashboards/

# Fix permissions
RUN chown -R 472:472 /etc/grafana/provisioning/dashboards

# Switch back to Grafana user (uid 472)
USER 472

