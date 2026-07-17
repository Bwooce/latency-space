# monitoring/grafana/Dockerfile
FROM grafana/grafana:13.1.0

# Create Grafana directories
USER root
RUN mkdir -p /etc/grafana/provisioning/dashboards /etc/grafana/provisioning/datasources

# Copy dashboards and config
COPY dashboards /etc/grafana/provisioning/dashboards
COPY provisioning/dashboards/default.yaml /etc/grafana/provisioning/dashboards/

# Copy the Prometheus datasource (without this the dashboards have no data source)
COPY provisioning/datasources /etc/grafana/provisioning/datasources

# Fix permissions
RUN chown -R 472:472 /etc/grafana/provisioning/dashboards /etc/grafana/provisioning/datasources

# Switch back to Grafana user (uid 472)
USER 472

