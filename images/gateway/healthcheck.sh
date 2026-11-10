#!/usr/bin/env bash
#
# NAME
#        healthcheck.sh — report that the gateway carries allow-listed traffic
#
# DESCRIPTION
#        Resolves the first allow-listed host through the local resolver and opens a real
#        connection to it, so the cage starts once egress works. On an empty policy it checks
#        that the resolver runs, which starts after the firewall is in place.
#
# SEE ALSO
#        compose.yaml, entrypoint.sh

set -euo pipefail

die() { echo "[healthcheck] ERROR: $*" >&2; exit 1; }

# Read the home's policy where the stack mounts it, and the old policy otherwise.
if [ -f /config/o3s.json ]; then
  FIRST_HOST=(-p=json '[ .hosts[0:1][] | [.name, .ports[0]] ]' /config/o3s.json)
else
  FIRST_HOST=(-p=toml '[ to_entries | .[0] | [.key, (.value.ports | .[0])] ]' /config/config.toml)
fi

# Probe the first allow-listed host by connecting to its address on its first port.
read -r ADDR PORT < <(yq -o=tsv "${FIRST_HOST[@]}") || true
# On an empty policy, report healthy once the resolver runs, which starts after the firewall.
if [ -z "$ADDR" ] || [ -z "$PORT" ]; then
  pgrep -x dnsmasq > /dev/null || die "dnsmasq is not running"
  exit 0
fi

# A wildcard entry probes its bare domain, which is what dnsmasq holds either way.
ADDR="${ADDR#\*.}"

# A domain resolves through dnsmasq, which also seeds the ipset, and an IP is used as-is.
case "$ADDR" in
  *[a-zA-Z]*) IP="$(nslookup "$ADDR" 127.0.0.1 2>/dev/null \
                    | awk '/^Name/{f=1} f && /Address/ && $NF ~ /^[0-9]+(\.[0-9]+){3}$/ {print $NF; exit}')" || true ;;
  *)          IP="$ADDR" ;;
esac
[ -n "$IP" ] || die "dnsmasq did not resolve $ADDR"

nc -w3 "$IP" "$PORT" </dev/null >/dev/null 2>&1 \
  || die "no egress to $ADDR ($IP) on TCP $PORT"
