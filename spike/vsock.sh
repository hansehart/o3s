#!/bin/bash
# Spike V on Linux: a cage VM with no network device, whose one link is vsock to an allow-list proxy on the host
set -uo pipefail
. "$(dirname "$0")/lib.sh"

SPIKE="$(cd "$(dirname "$0")" && pwd)"
WORK="${RUNNER_TEMP:-/tmp}/vsock"
CID=3
IMAGE_URL="$(awk '/location:.*amd64/ {gsub(/"/, "", $3); print $3}' internal/vm/o3s.yaml)"
IMAGE_SUM="$(awk '/amd64/ {f=1} f && /digest:/ {gsub(/"|sha256:/, "", $2); print $2; exit}' internal/vm/o3s.yaml)"
PROXY_LOG="${WORK}/tinyproxy.log"
mkdir -p "${WORK}"

# Ssh into the cage VM over vsock alone
cage() {
  ssh -i "${WORK}/key" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -o ConnectTimeout=10 \
    -o ProxyCommand="python3 ${SPIKE}/vsock-relay.py stdio ${CID} 2222" ubuntu@cage "$@"
}

# The pinned image, checked against the digest the home's VM template names
check "the cage image downloads and matches its digest" bash -c "curl -fsSLo '${WORK}/base.img' '${IMAGE_URL}' && echo '${IMAGE_SUM}  ${WORK}/base.img' | sha256sum -c"
qemu-img create -q -f qcow2 -b "${WORK}/base.img" -F qcow2 "${WORK}/disk.qcow2" 30G

# The allow-list proxy on the host: names outside the list are refused, HTTPS only on 443
cat > "${WORK}/tinyproxy.conf" <<CONF
Port 3128
Listen 127.0.0.1
Timeout 600
LogLevel Notice
LogFile "${PROXY_LOG}"
PidFile "${WORK}/tinyproxy.pid"
MaxClients 200
Filter "${SPIKE}/vsock-allow.txt"
FilterType ere
FilterDefaultDeny Yes
ConnectPort 443
CONF
tinyproxy -v
tinyproxy -c "${WORK}/tinyproxy.conf"
# Carry the cage VM's vsock connections on port 3128 to the proxy
python3 "${SPIKE}/vsock-relay.py" host 3128 3128 >/dev/null 2>&1 &

bash "${SPIKE}/vsock-seed.sh" "${WORK}"
cloud-localds "${WORK}/seed.img" "${WORK}/user-data" "${WORK}/meta-data"

# Boot the cage VM with no network device, only vsock
START=${SECONDS}
qemu-system-x86_64 -machine q35,accel=kvm -cpu host -smp 2 -m 6G -nic none \
  -drive file="${WORK}/disk.qcow2",if=virtio -drive file="${WORK}/seed.img",if=virtio,format=raw \
  -device vhost-vsock-pci,guest-cid=${CID} -display none -serial file:"${WORK}/console.log" -daemonize
. "${SPIKE}/vsock-checks.sh"
echo "--- console"
tail -40 "${WORK}/console.log"

# Run an extra spike against the booted cage VM when one is named
[ -n "${EXTRA:-}" ] && . "${SPIKE}/${EXTRA}"
