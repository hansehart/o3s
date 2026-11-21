#!/bin/bash
# Spike P on Linux: how much faster a prepared image makes the cage VM ready than the stock image set up by cloud-init
set -uo pipefail
. "$(dirname "$0")/lib.sh"

SPIKE="$(cd "$(dirname "$0")" && pwd)"
WORK="${RUNNER_TEMP:-/tmp}/prebuilt"
CID=3
IMAGE_URL="$(awk '/location:.*amd64/ {gsub(/"/, "", $3); print $3}' internal/vm/o3s.yaml)"
IMAGE_SUM="$(awk '/amd64/ {f=1} f && /digest:/ {gsub(/"|sha256:/, "", $2); print $2; exit}' internal/vm/o3s.yaml)"
NODE=kindest/node:v1.37.0@sha256:a1ed56cfb0e7b93589bdf97c8cd566405a265939e3620fc4f5de89adff580ae5
KIND_URL=https://github.com/kubernetes-sigs/kind/releases/download/v0.33.0/kind-linux-amd64
KIND_SUM=aee6151561422756b764a4ae28e7f44cda5af5a9eead3cc9985112b1de8d8e0d
mkdir -p "${WORK}"

# Ssh into the cage VM over vsock alone
cage() {
  ssh -i "${WORK}/key" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -o ConnectTimeout=5 -o ServerAliveInterval=5 -o ServerAliveCountMax=3 \
    -o ProxyCommand="python3 ${SPIKE}/vsock-relay.py stdio ${CID} 2222" ubuntu@cage "$@"
}

# Print one measurement row
bench() {
  echo "BENCH | $1 | $2 | $3 |"
}

# Boot DISK with no network device, with the seed when one is named, and print the seconds until ssh answers over vsock
boot() {
  local disk=$1 seed=${2:-} start=${SECONDS}
  local drives=(-drive "file=${disk},if=virtio")
  [ -n "${seed}" ] && drives+=(-drive "file=${seed},if=virtio,format=raw")
  qemu-system-x86_64 -machine q35,accel=kvm -cpu host -smp 2 -m 6G -nic none "${drives[@]}" \
    -device vhost-vsock-pci,guest-cid=${CID} -display none -serial file:"${WORK}/console.log" \
    -pidfile "${WORK}/qemu.pid" -daemonize
  # Try every second for up to 15 minutes
  until timeout 20 ssh -i "${WORK}/key" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -o ConnectTimeout=5 \
    -o ProxyCommand="python3 ${SPIKE}/vsock-relay.py stdio ${CID} 2222" ubuntu@cage true 2>/dev/null; do
    [ $((SECONDS - start)) -lt 900 ] || return 1
    sleep 1
  done
  echo $((SECONDS - start))
}

# Power the VM off and wait for QEMU to end
halt() {
  local pid
  pid="$(cat "${WORK}/qemu.pid")"
  timeout 30 ssh -i "${WORK}/key" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR \
    -o ProxyCommand="python3 ${SPIKE}/vsock-relay.py stdio ${CID} 2222" ubuntu@cage sudo systemctl poweroff >/dev/null 2>&1
  # Wait up to a minute, then end QEMU
  for _ in $(seq 60); do kill -0 "${pid}" 2>/dev/null || return 0; sleep 1; done
  kill "${pid}"
}

# Seed a fresh instance, so cloud-init runs its first-boot work again
seed() {
  echo "instance-id: cage-$1" > "${WORK}/meta-data"
  cloud-localds "${WORK}/seed-$1.img" "${WORK}/user-data" "${WORK}/meta-data"
  echo "${WORK}/seed-$1.img"
}

# Seconds a command in the cage takes
timed() {
  local start=${SECONDS}
  cage "$@" >/dev/null 2>&1 || return 1
  echo $((SECONDS - start))
}

SETUP="sudo apt-get update -q && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -q docker.io && sudo usermod -aG docker ubuntu \
  && sudo curl -fsSLo /usr/local/bin/kind ${KIND_URL} && echo '${KIND_SUM}  /usr/local/bin/kind' | sha256sum -c && sudo chmod +x /usr/local/bin/kind"
KIND="env -u http_proxy -u https_proxy -u HTTP_PROXY -u HTTPS_PROXY kind create cluster --wait 5m && kind delete cluster"

# The pinned image and the allow-list proxy on the host, as in spike V
check "the image downloads and matches its digest" bash -c "curl -fsSLo '${WORK}/base.img' '${IMAGE_URL}' && echo '${IMAGE_SUM}  ${WORK}/base.img' | sha256sum -c"
sed "s#@WORK@#${WORK}#g; s#@ALLOW@#${SPIKE}/vsock-allow.txt#" > "${WORK}/tinyproxy.conf" <<'CONF'
Port 3128
Listen 127.0.0.1
Timeout 600
LogLevel Notice
LogFile "@WORK@/tinyproxy.log"
PidFile "@WORK@/tinyproxy.pid"
MaxClients 200
Filter "@ALLOW@"
FilterType ere
FilterDefaultDeny Yes
ConnectPort 443
CONF
tinyproxy -c "${WORK}/tinyproxy.conf"
python3 "${SPIKE}/vsock-relay.py" host 3128 3128 >/dev/null 2>&1 &
bash "${SPIKE}/vsock-seed.sh" "${WORK}"

# A: the stock image, measured already in run 44925617797 unless asked again
if [ -z "${SKIP_STOCK:-}" ]; then
# A: the stock image, set up by cloud-init on first boot, then Docker, kind and the node image installed through the proxy
for run in 1 2 3; do
  qemu-img create -q -f qcow2 -b "${WORK}/base.img" -F qcow2 "${WORK}/a.qcow2" 30G
  t0=${SECONDS}
  b=$(boot "${WORK}/a.qcow2" "$(seed "a${run}")") || { bench "stock: boot to ssh" FAIL s; continue; }
  bench "stock: boot to ssh" "${b}" s
  bench "stock: install Docker and kind" "$(timed "${SETUP}")" s
  bench "stock: pull the node image" "$(timed docker pull -q "${NODE}")" s
  bench "stock: kind create and delete" "$(timed "${KIND}")" s
  bench "stock: start to first cluster" $((SECONDS - t0)) s
  timeout 20 cage systemd-analyze 2>/dev/null | head -1
  halt
done

fi

# Build the prepared image once: the stock image with Docker, kind and the node image, reset for a new instance
qemu-img create -q -f qcow2 -b "${WORK}/base.img" -F qcow2 "${WORK}/build.qcow2" 30G
t0=${SECONDS}
gate "the prepared image's build VM boots" boot "${WORK}/build.qcow2" "$(seed build)"
check "the build installs Docker and kind" cage "${SETUP}"
check "the build pulls the node image" cage docker pull -q "${NODE}"
cage 'sudo apt-get clean && sudo cloud-init clean --logs' >/dev/null
halt
qemu-img convert -O qcow2 "${WORK}/build.qcow2" "${WORK}/prepared.qcow2"
bench "build the prepared image" $((SECONDS - t0)) s
bench "prepared image size" "$(du -m "${WORK}/prepared.qcow2" | cut -f1)" MiB

# B: the prepared image, with cloud-init still taking the seed of a new instance
for run in 1 2 3; do
  qemu-img create -q -f qcow2 -b "${WORK}/prepared.qcow2" -F qcow2 "${WORK}/b.qcow2" 30G
  t0=${SECONDS}
  b=$(boot "${WORK}/b.qcow2" "$(seed "b${run}")") || { bench "prepared: boot to ssh" FAIL s; continue; }
  bench "prepared: boot to ssh" "${b}" s
  bench "prepared: kind create and delete" "$(timed "${KIND}")" s
  bench "prepared: start to first cluster" $((SECONDS - t0)) s
  timeout 20 cage systemd-analyze 2>/dev/null | head -1
  halt
done

# Make the image that needs no seed: cloud-init off, the key and services already in place
qemu-img create -q -f qcow2 -b "${WORK}/prepared.qcow2" -F qcow2 "${WORK}/c-build.qcow2" 30G
boot "${WORK}/c-build.qcow2" "$(seed cbuild)" >/dev/null
cage 'sudo touch /etc/cloud/cloud-init.disabled'
halt
qemu-img convert -O qcow2 "${WORK}/c-build.qcow2" "${WORK}/baked.qcow2"

# C: the prepared image with cloud-init off, booted with no seed at all
for run in 1 2 3; do
  qemu-img create -q -f qcow2 -b "${WORK}/baked.qcow2" -F qcow2 "${WORK}/c.qcow2" 30G
  t0=${SECONDS}
  b=$(boot "${WORK}/c.qcow2") || { bench "baked: boot to ssh" FAIL s; continue; }
  bench "baked: boot to ssh" "${b}" s
  bench "baked: kind create and delete" "$(timed "${KIND}")" s
  bench "baked: start to first cluster" $((SECONDS - t0)) s
  timeout 20 cage systemd-analyze 2>/dev/null | head -1
  timeout 20 cage 'systemd-analyze blame | head -8'
  halt
done
