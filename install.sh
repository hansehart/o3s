#!/bin/sh
# NAME
#        install.sh — installs the o3s binary and the devcontainer CLI it rebuilds the cage with
#
# SYNOPSIS
#        curl -fsSL https://github.com/hansehart/o3s/releases/latest/download/install.sh | sh
#        sh install.sh [--version <X.Y.Z>] [--no-devcontainer]
#
# DESCRIPTION
#        Downloads the o3s binary of a release for this host, checks it against the release's
#        checksums, verifies its build attestation where gh is logged in, and installs it to
#        ~/.local/bin. Then installs the devcontainer CLI, with its own Node, into ~/.devcontainers.
#
# ENVIRONMENT
#        O3S_INSTALL_DIR             where the binary goes (default: ~/.local/bin)
#        O3S_RELEASE_URL             where the release's files come from (default: GitHub releases)
#        DEVCONTAINERS_INSTALL_DIR   where the devcontainer CLI goes (default: ~/.devcontainers)
set -eu

repo="hansehart/o3s"
signer="${repo}/.github/workflows/publish.yaml"

# The devcontainer CLI's installer at a pinned tag, and the checksum it has there.
devcontainers_version="0.90.0"
devcontainers_installer_sha256="137e273aea49666a6ed9b5b0796153efc5873d8778f69ce552beb6ff3fab2d69"

# say prints a line for the user.
say() {
	printf '%s\n' "$*"
}

# die names why the install stops, and stops it.
die() {
	printf 'install.sh: %s\n' "$*" >&2
	exit 1
}

# fetch downloads a URL into a file, over HTTPS alone for the default addresses.
fetch() {
	# The protocol flags split into words on purpose
	# shellcheck disable=SC2086
	curl -fsSL ${proto} -o "$2" "$1"
}

# check_sums checks every file a checksum list names in the folder that holds it.
check_sums() {
	(cd "$1" && sha256sum -c --quiet "$2" >/dev/null 2>&1)
}

version=""
devcontainer="yes"
# Read every option
while [ $# -gt 0 ]; do
	# Take each option the script knows, and name any other
	case "$1" in
	--version)
		[ $# -ge 2 ] || die "--version needs a version, as in --version 0.7.0"
		version="${2#v}"
		shift 2
		;;
	--no-devcontainer)
		devcontainer="no"
		shift
		;;
	-h | --help)
		say "Usage: sh install.sh [--version <X.Y.Z>] [--no-devcontainer]"
		exit 0
		;;
	*)
		die "unknown option $1, see --help"
		;;
	esac
done

kernel=$(uname -s)
arch=$(uname -m)
# Accept the one kernel o3s ships for
case "${kernel}" in
Linux) ;;
*) die "o3s runs on Linux, and this host runs ${kernel}" ;;
esac
# Name each architecture by the name the release gives it
case "${arch}" in
x86_64 | amd64) arch="amd64" ;;
aarch64 | arm64) arch="arm64" ;;
*) die "o3s ships for amd64 and arm64, and this host is ${arch}" ;;
esac
asset="o3s-linux-${arch}"

prefix="${DEVCONTAINERS_INSTALL_DIR:-${HOME}/.devcontainers}"
# Name the tools the devcontainer CLI's installer unpacks Node with, before anything installs
if [ "${devcontainer}" = "yes" ] && [ ! -x "${prefix}/bin/devcontainer" ]; then
	# Check each tool in turn
	for tool in tar xz; do
		command -v "${tool}" >/dev/null 2>&1 ||
			die "the devcontainer CLI's installer needs tar and xz to unpack Node, so install them, or run install.sh with --no-devcontainer"
	done
fi

proto="--proto =https --tlsv1.2"
base="https://github.com/${repo}/releases/latest/download"
# Take the release the user named
if [ -n "${version}" ]; then
	base="https://github.com/${repo}/releases/download/v${version}"
fi
# Take the files from where the environment points, a mirror or a test, over any protocol
if [ -n "${O3S_RELEASE_URL:-}" ]; then
	base="${O3S_RELEASE_URL}"
	proto=""
fi

tmp=$(mktemp -d)
trap 'rm -rf "${tmp}"' EXIT

fetch "${base}/${asset}" "${tmp}/${asset}" || die "failed to download ${asset} from ${base}"
fetch "${base}/checksums.txt" "${tmp}/checksums.txt" || die "failed to download checksums.txt from ${base}"
grep "  ${asset}\$" "${tmp}/checksums.txt" >"${tmp}/${asset}.sha256" || die "checksums.txt names no ${asset}"
check_sums "${tmp}" "${asset}.sha256" || die "${asset} differs from its line in checksums.txt, so the download stays uninstalled"

verified="no"
# Verify the build attestation where gh is installed and logged in
if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
	(cd "${tmp}" && gh attestation verify "${asset}" --repo "${repo}" --signer-workflow "${signer}" >/dev/null 2>&1) || die "the build attestation of ${asset} fails to verify, so the download stays uninstalled"
	verified="yes"
fi

dir="${O3S_INSTALL_DIR:-${HOME}/.local/bin}"
mkdir -p "${dir}"
install -m 0755 "${tmp}/${asset}" "${dir}/o3s"
say "Installed $("${dir}/o3s" --version) to ${dir}/o3s"
# Confirm the attestation, or show how to verify it by hand
if [ "${verified}" = "yes" ]; then
	say "Verified its build attestation"
else
	say "Verify its build attestation with: gh attestation verify ${dir}/o3s --repo ${repo} --signer-workflow ${signer}"
fi
# Show how to reach the binary from a folder outside the PATH
case ":${PATH}:" in
*":${dir}:"*) ;;
*) say "Add ${dir} to your PATH: export PATH=\"${dir}:\$PATH\"" ;;
esac

# Install the devcontainer CLI, which o3s rebuilds the cage with
if [ "${devcontainer}" = "yes" ]; then
	# Keep a devcontainer CLI the host holds already
	if [ -x "${prefix}/bin/devcontainer" ]; then
		say "Found the devcontainer CLI in ${prefix}"
	else
		proto="--proto =https --tlsv1.2"
		fetch "https://raw.githubusercontent.com/devcontainers/cli/v${devcontainers_version}/scripts/install.sh" "${tmp}/devcontainers.sh" ||
			die "failed to download the devcontainer CLI's installer"
		printf '%s  devcontainers.sh\n' "${devcontainers_installer_sha256}" >"${tmp}/devcontainers.sha256"
		check_sums "${tmp}" "devcontainers.sha256" || die "the devcontainer CLI's installer differs from its pinned checksum"
		DEVCONTAINERS_INSTALL_DIR="${prefix}" sh "${tmp}/devcontainers.sh" --version "${devcontainers_version}"
	fi
fi

say ""
say "Next: mkdir ~/o3s && cd ~/o3s && o3s init"
