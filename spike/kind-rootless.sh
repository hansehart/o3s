#!/bin/bash
# Spike K, sourced by vsock.sh once the cage VM runs Docker and kind: does kind run on a rootless engine inside the confined cage?
# A runs the docker-in-docker feature as homes do today; B runs systemd in the cage, with the rootless engine as a delegated user service

NODE=kindest/node:v1.37.0
STACK="$(cd "${SPIKE}/.." && pwd)/internal/home/stack"
CAGE_OPTS="--network host --cap-drop MKNOD --cap-drop NET_RAW --cap-drop AUDIT_WRITE --device /dev/net/tun \
  --security-opt no-new-privileges=false --security-opt apparmor=o3s-cage --security-opt seccomp=/tmp/kr/seccomp.json \
  --security-opt systempaths=unconfined"
PROXY="-e http_proxy=http://127.0.0.1:3128 -e https_proxy=http://127.0.0.1:3128 -e no_proxy=localhost,127.0.0.1"

# The cage's own profile and syscall filter, loaded in the cage VM as o3s init loads them on a host
cage 'mkdir -p /tmp/kr/f'
cage 'cat > /tmp/kr/apparmor.conf' < "${STACK}/apparmor.conf"
cage 'cat > /tmp/kr/seccomp.json' < "${STACK}/seccomp.json"
check "the cage VM loads the o3s-cage profile" cage 'sudo apparmor_parser -r /tmp/kr/apparmor.conf'
cage 'sudo modprobe -a ip_tables ip6_tables iptable_nat ip6table_nat' || true

# The cage image with the docker-in-docker feature, systemd and kind, built through the proxy
cage 'curl -fsSL https://codeload.github.com/agentic-software-engineers/devcontainer-features/tar.gz/refs/tags/docker-in-docker-5.0.0 | tar -xz -C /tmp/kr/f --strip-components=3 --wildcards "*/src/docker-in-docker/*"'
cage 'cat > /tmp/kr/Dockerfile' <<'DOCKERFILE'
FROM ghcr.io/hansehart/o3s-cage:latest
ARG http_proxy
ARG https_proxy
USER root
RUN apt-get update -q && DEBIAN_FRONTEND=noninteractive apt-get install -y -q systemd systemd-sysv dbus-user-session >/dev/null
COPY f /tmp/feature
RUN cd /tmp/feature && _REMOTE_USER=ubuntu _REMOTE_USER_HOME=/home/ubuntu bash install.sh >/dev/null
RUN curl -fsSLo /usr/local/bin/kind https://github.com/kubernetes-sigs/kind/releases/download/v0.33.0/kind-linux-amd64 \
 && echo "aee6151561422756b764a4ae28e7f44cda5af5a9eead3cc9985112b1de8d8e0d  /usr/local/bin/kind" | sha256sum -c && chmod +x /usr/local/bin/kind \
 && mkdir -p /var/lib/systemd/linger && touch /var/lib/systemd/linger/ubuntu
DOCKERFILE
check "the cage image builds with the feature, systemd and kind" cage 'docker build -q --network host --build-arg http_proxy=http://127.0.0.1:3128 --build-arg https_proxy=http://127.0.0.1:3128 -t cage-kr /tmp/kr'
# Tag the node image the VM's engine holds, to hand it to the rootless engines without a pull
cage "docker tag \$(docker images -q kindest/node | head -1) ${NODE}"

# Load the node image into a cage's rootless engine and raise a cluster there, as the dev user
kind_in() {
  local name=$1 sock=$2
  cage "docker save ${NODE} | docker exec -i -u ubuntu -e DOCKER_HOST=${sock} ${name} docker load -q" || return 1
  cage "docker exec -u ubuntu -e DOCKER_HOST=${sock} ${name} docker info --format 'driver={{.CgroupDriver}} memory={{.MemoryLimit}} pids={{.PidsLimit}} cpu={{.CPUShares}}'"
  cage "docker exec -u ubuntu -e DOCKER_HOST=${sock} ${name} kind create cluster --image ${NODE} --wait 5m"
}
# Wait up to two minutes for a command in the cage VM to succeed
waits() {
  local i
  for i in $(seq 60); do cage "$@" >/dev/null 2>&1 && return 0; sleep 2; done
  return 1
}

# A: the feature's own entrypoint, the engine started without systemd
cage "docker run -d --name cage-a -v dind-a:/var/lib/docker-rootless ${CAGE_OPTS} ${PROXY} cage-kr /usr/local/libexec/docker-in-docker/docker-init.sh sleep infinity" >/dev/null
check "A: the cage's rootless engine answers" waits 'docker exec -u ubuntu cage-a docker -H unix:///run/docker-rootless.sock info'
check "A: kind raises a cluster on the rootless engine" kind_in cage-a unix:///run/docker-rootless.sock
echo "--- A logs"
cage 'docker ps -a --filter name=cage-a --format "{{.Status}}"; docker logs --tail 30 cage-a 2>&1; docker exec cage-a tail -40 /tmp/dockerd.log 2>&1; docker rm -f cage-a >/dev/null'

# B: systemd as the cage's init, the engine a user service with delegated cgroups
cage "docker run -d -t --name cage-b -v dind-b:/home/ubuntu/.local/share/docker ${CAGE_OPTS} ${PROXY} --cgroupns=private --tmpfs /run --tmpfs /run/lock --stop-signal SIGRTMIN+3 cage-kr /sbin/init" >/dev/null
check "B: systemd runs the dev user's session" waits 'docker exec cage-b systemctl is-active user@1000.service'
echo "--- B logs"
cage 'docker ps -a --filter name=cage-b --format "{{.Status}}"; docker logs --tail 60 cage-b 2>&1 | tr -d "\r"'
cage 'sudo journalctl -k --no-pager | grep DENIED | grep -v profile=.who | tail -20'
cage 'docker exec cage-b systemctl --no-pager --failed; docker exec cage-b cat /sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/cgroup.controllers'
check "B: the rootless engine installs as a user service" cage 'docker exec -u ubuntu -e XDG_RUNTIME_DIR=/run/user/1000 -e DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus cage-b dockerd-rootless-setuptool.sh install'
check "B: kind raises a cluster on the rootless engine" kind_in cage-b unix:///run/user/1000/docker.sock
check "B: the cluster's node is Ready" cage 'docker exec -u ubuntu -e DOCKER_HOST=unix:///run/user/1000/docker.sock cage-b docker exec kind-control-plane kubectl wait --for=condition=Ready node --all --timeout=60s'
cage 'docker exec cage-b journalctl --no-pager -n 30 --user-unit docker 2>/dev/null || true'
