#!/usr/bin/env bash
#
# NAME
#        check-egress.sh — probe that the egress policy holds
#
# DESCRIPTION
#        Reports on every container start whether a non-allow-listed address stays refused.
#
# SEE ALSO
#        devcontainer.json, o3s init

set -euo pipefail

DENY_HOST=1.1.1.1
# A non-allow-listed public IP must stay blocked
if curl -k -s --connect-timeout 3 -o /dev/null "https://$DENY_HOST"; then
  echo "[o3s] WARNING: network lockdown is not active, the cage can reach hosts outside your allowlist" >&2
else
  echo "[o3s] INFO: network lockdown active, only allowlisted hosts are reachable"
fi
