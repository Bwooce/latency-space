# monitoring/grafana/Dockerfile
FROM grafana/grafana:latest

# Copy dashboards
COPY dashboards /etc/grafana/provisioning/dashboards

# Copy provisioning config
COPY provisioning/dashboards/default.yaml /etc/grafana/provisioning/dashboards/default.yaml

# Set permissions
USER root
RUN chown -R grafana:grafana /etc/grafana/provisioning/dashboards
USER grafana

