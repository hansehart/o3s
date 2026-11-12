#!/usr/bin/env bash
#
# NAME
#        env.sh — expose the allowlist's secret markers to every shell
#
# DESCRIPTION
#        Reads the secret name each credential-inject host declares and writes a profile
#        script pointing that variable at the marker, which the proxy swaps for the real
#        token on egress.
#
# SEE ALSO
#        o3s.json, start.sh

set -euo pipefail

die() { echo "[o3s] ERROR: $*" >&2; exit 1; }

# Read the allowlist from the home mount and expose the markers where every shell picks them up,
# both paths open to a test that points them elsewhere
CONFIG_FILE="${O3S_CONFIG_FILE:-/home/ubuntu/o3s/o3s.json}"
PROFILE_FILE="${O3S_PROFILE_FILE:-/etc/profile.d/o3s-secrets.sh}"

# Collect the secret name each credential-inject host declares
ROWS="$(yq -p=json -o=tsv '[ .hosts[] | [(.secret // "")] ]' "$CONFIG_FILE")" \
  || die "$CONFIG_FILE is not valid JSON, markers not updated"
SECRETS="$(printf '%s\n' "$ROWS" | grep -v '^$' || true)"

# Point each provider variable at the marker, which the proxy swaps for the real token on egress
{
  echo "# Generated from the home's policy on every container start"
  for name in $SECRETS; do
    echo "export $name=\"\$O3S_MARKER\""
  done
} > "$PROFILE_FILE"
