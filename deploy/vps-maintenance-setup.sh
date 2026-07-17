#!/bin/bash
# One-time setup: disk-cleanup + cert-reload automation for latency.space.
# Root cause of the 2026 outage: 25GB of Docker build cache filled the disk,
# so no containers could start (nginx then returned 502). Plus certbot renewed
# the cert but nginx was never reloaded, so an expired cert was served.
set -uo pipefail

echo "== 1. Weekly Docker cleanup cron =="
cat > /etc/cron.d/docker-cleanup <<'CRON'
# Weekly Docker cleanup - prevents build-cache disk exhaustion.
# Prunes build cache unused >7 days and dangling images. Named volumes untouched.
SHELL=/bin/bash
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
30 4 * * 0 root docker builder prune -f --filter until=168h >> /var/log/docker-cleanup.log 2>&1; docker image prune -f >> /var/log/docker-cleanup.log 2>&1
CRON
chmod 644 /etc/cron.d/docker-cleanup
echo "   -> /etc/cron.d/docker-cleanup"

echo "== 2. BuildKit GC cap in daemon.json (auto-limit; applies on next docker restart) =="
cp -a /etc/docker/daemon.json "/etc/docker/daemon.json.bak.$(date +%s)"
tmp=$(mktemp)
if jq '. + {builder: {gc: {enabled: true, defaultKeepStorage: "10GB"}}}' /etc/docker/daemon.json > "$tmp" && jq empty "$tmp" 2>/dev/null; then
  mv "$tmp" /etc/docker/daemon.json
  echo "   -> merged builder.gc (cap 10GB). NOT restarting docker; effective on next restart."
else
  echo "   -> ERROR merging daemon.json; original kept."; rm -f "$tmp"
fi

echo "== 3. certbot deploy-hook: reload nginx on every renewal =="
mkdir -p /etc/letsencrypt/renewal-hooks/deploy
cat > /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh <<'HOOK'
#!/bin/sh
# Runs after any successful certbot renewal so the new cert is actually served.
systemctl reload nginx
HOOK
chmod +x /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh
echo "   -> /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh"

echo "== 4. Weekly safety nginx reload cron =="
cat > /etc/cron.d/nginx-cert-reload <<'CRON'
# Weekly graceful nginx reload so a renewed cert never drifts unserved.
SHELL=/bin/bash
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
15 3 * * 1 root systemctl reload nginx >> /var/log/nginx-cert-reload.log 2>&1
CRON
chmod 644 /etc/cron.d/nginx-cert-reload
echo "   -> /etc/cron.d/nginx-cert-reload"

echo "== 5. Reload nginx now (serve the already-renewed cert) =="
if nginx -t 2>/tmp/nginxtest; then
  systemctl reload nginx && echo "   -> nginx reloaded OK"
else
  echo "   -> nginx config test FAILED, not reloading:"; cat /tmp/nginxtest
fi
echo "== done =="
