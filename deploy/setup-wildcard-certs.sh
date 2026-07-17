#!/usr/bin/env bash
#
# setup-wildcard-certs.sh
#
# Reissue the latency.space Let's Encrypt certificate so it also covers the
# second-level body subdomains (moon.planet.latency.space). Those hosts are
# routed by nginx and served by the proxy, but the current certificate is a
# single-level "*.latency.space" wildcard, which by definition does NOT match a
# two-label name like phobos.mars.latency.space - so HTTPS fails there.
#
# The fix is one certificate with per-parent-body wildcard SANs. A wildcard SAN
# can only be issued via the DNS-01 challenge, done here through Cloudflare.
#
# The certificate keeps --cert-name "latency.space", so it lands at the same
# path nginx already references (/etc/letsencrypt/live/latency.space/), and the
# renewal deploy-hook installed by vps-maintenance-setup.sh reloads nginx on
# every future renewal. No nginx change is required.
#
# Prerequisites on the host:
#   - certbot with the dns-cloudflare plugin (this script installs it if missing)
#   - a Cloudflare API token with Zone:DNS:Edit on the latency.space zone,
#     provided via the CLOUDFLARE_API_TOKEN environment variable
#
# Usage:
#   CLOUDFLARE_API_TOKEN=xxxx SSL_EMAIL=you@example.com ./deploy/setup-wildcard-certs.sh
#
#   DRY_RUN=1 CLOUDFLARE_API_TOKEN=xxxx ./deploy/setup-wildcard-certs.sh
#     Validate the whole DNS-01 flow against the Let's Encrypt staging endpoint
#     WITHOUT issuing a real certificate or touching production. Run this first.
#
set -euo pipefail

CERT_NAME="latency.space"
SSL_EMAIL="${SSL_EMAIL:-bruce@fitzsimons.org}"
DRY_RUN="${DRY_RUN:-0}"

# Parent bodies that currently have moons. Each needs a wildcard SAN so its
# moons validate at moon.<parent>.latency.space. Regenerate with:
#   curl -s https://latency.space/api/status-data \
#     | python3 -c 'import sys,json;o=[x for v in json.load(sys.stdin)["objects"].values() for x in v];print(*sorted({m["parentName"].lower().replace(" ","-") for m in o if m.get("type")=="moon" and m.get("parentName")}))'
PARENTS=(earth jupiter mars neptune pluto saturn uranus)

if [ -z "${CLOUDFLARE_API_TOKEN:-}" ]; then
  echo "ERROR: CLOUDFLARE_API_TOKEN is not set (needs Zone:DNS:Edit on latency.space)." >&2
  exit 1
fi

# Build the domain argument list: apex + single-level wildcard + one wildcard
# per parent body.
DOMAIN_ARGS=(-d "latency.space" -d "*.latency.space")
for p in "${PARENTS[@]}"; do
  DOMAIN_ARGS+=(-d "*.${p}.latency.space")
done

echo "Certificate '${CERT_NAME}' will cover:"
printf '  %s\n' "latency.space" "*.latency.space"
for p in "${PARENTS[@]}"; do printf '  %s\n' "*.${p}.latency.space"; done

# Ensure certbot + the Cloudflare DNS plugin are present.
if ! command -v certbot >/dev/null 2>&1 || ! certbot plugins 2>/dev/null | grep -q dns-cloudflare; then
  echo "Installing certbot + python3-certbot-dns-cloudflare..."
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -qq
    apt-get install -y certbot python3-certbot-dns-cloudflare
  elif command -v dnf >/dev/null 2>&1; then
    dnf install -y certbot python3-certbot-dns-cloudflare
  else
    echo "ERROR: no supported package manager found; install certbot + dns-cloudflare manually." >&2
    exit 1
  fi
fi

# Write the Cloudflare credentials to a locked-down temp file and remove it on exit.
CF_CREDS="$(mktemp)"
chmod 600 "$CF_CREDS"
trap 'rm -f "$CF_CREDS"' EXIT
printf 'dns_cloudflare_api_token = %s\n' "$CLOUDFLARE_API_TOKEN" > "$CF_CREDS"

CERTBOT_ARGS=(
  certonly
  --dns-cloudflare
  --dns-cloudflare-credentials "$CF_CREDS"
  --dns-cloudflare-propagation-seconds 30
  --cert-name "$CERT_NAME"
  --non-interactive
  --agree-tos
  --email "$SSL_EMAIL"
  --expand
  --keep-until-expiring
  "${DOMAIN_ARGS[@]}"
)

if [ "$DRY_RUN" = "1" ]; then
  echo ">>> DRY RUN: validating DNS-01 against staging, no certificate issued."
  certbot "${CERTBOT_ARGS[@]}" --dry-run
  echo "Dry run succeeded. Re-run without DRY_RUN=1 to issue the real certificate."
  exit 0
fi

certbot "${CERTBOT_ARGS[@]}"

echo "Reloading nginx to serve the new certificate..."
if nginx -t; then
  systemctl reload nginx
  echo "Done. nginx is now serving the multi-wildcard certificate."
else
  echo "ERROR: nginx config test failed; NOT reloading. Investigate before proceeding." >&2
  exit 1
fi
