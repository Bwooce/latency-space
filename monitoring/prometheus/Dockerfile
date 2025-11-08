FROM prom/prometheus:latest

# Copy prometheus config
COPY prometheus.yml /etc/prometheus/prometheus.yml

# Copy alert rules if you have them
COPY rules/ /etc/prometheus/rules/

# Set permissions but stay as root
USER root
RUN chown -R nobody:nobody /etc/prometheus /prometheus
# Stay as root to avoid permission issues
# USER nobody

# Default command
CMD [ "--config.file=/etc/prometheus/prometheus.yml", \
      "--storage.tsdb.path=/prometheus", \
      "--web.console.libraries=/usr/share/prometheus/console_libraries", \
      "--web.console.templates=/usr/share/prometheus/consoles" ]

