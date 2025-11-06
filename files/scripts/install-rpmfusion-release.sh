#!/usr/bin/env bash
set -euo pipefail

RETRY_COUNT="${RPMFUSION_RETRY_COUNT:-5}"
RETRY_DELAY="${RPMFUSION_RETRY_DELAY:-10}"

if rpm -q rpmfusion-free-release >/dev/null 2>&1 \
  && rpm -q rpmfusion-nonfree-release >/dev/null 2>&1; then
  echo "RPM Fusion release packages already installed; skipping download."
  exit 0
fi

OS_VERSION="${OS_VERSION:-}"
if [[ -z "${OS_VERSION}" ]]; then
  if ! OS_VERSION="$(rpm -E %fedora)"; then
    echo "Unable to determine Fedora version via rpm -E %fedora" >&2
    exit 1
  fi
fi

if [[ -z "${OS_VERSION}" ]]; then
  echo "Fedora version is empty; cannot build RPM Fusion URLs" >&2
  exit 1
fi

primary_free="https://download1.rpmfusion.org/free/fedora/rpmfusion-free-release-%OS_VERSION%.noarch.rpm"
primary_nonfree="https://download1.rpmfusion.org/nonfree/fedora/rpmfusion-nonfree-release-%OS_VERSION%.noarch.rpm"
fallback_free="https://mirrors.rpmfusion.org/free/fedora/rpmfusion-free-release-%OS_VERSION%.noarch.rpm"
fallback_nonfree="https://mirrors.rpmfusion.org/nonfree/fedora/rpmfusion-nonfree-release-%OS_VERSION%.noarch.rpm"

declare -A URLs=(
  [free_primary]="${primary_free//%OS_VERSION%/${OS_VERSION}}"
  [free_fallback]="${fallback_free//%OS_VERSION%/${OS_VERSION}}"
  [nonfree_primary]="${primary_nonfree//%OS_VERSION%/${OS_VERSION}}"
  [nonfree_fallback]="${fallback_nonfree//%OS_VERSION%/${OS_VERSION}}"
)

download_with_retry() {
  local destination="$1"
  shift
  local url

  for url in "$@"; do
    if curl --retry "${RETRY_COUNT}" --retry-delay "${RETRY_DELAY}" --connect-timeout 10 --max-time 60 -fL -o "${destination}" "${url}"; then
      return 0
    fi
    echo "Download failed for ${url}, trying next mirror if available..." >&2
  done

  echo "All download attempts failed for ${destination}" >&2
  return 1
}

TMP_WORKDIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_WORKDIR}"
}
trap cleanup EXIT

FREE_RPM="${TMP_WORKDIR}/rpmfusion-free-release.rpm"
NONFREE_RPM="${TMP_WORKDIR}/rpmfusion-nonfree-release.rpm"

free_downloaded=0
nonfree_downloaded=0

if download_with_retry "${FREE_RPM}" "${URLs[free_primary]}" "${URLs[free_fallback]}"; then
  free_downloaded=1
else
  echo "Failed to download RPM Fusion FREE repository release package." >&2
fi

if download_with_retry "${NONFREE_RPM}" "${URLs[nonfree_primary]}" "${URLs[nonfree_fallback]}"; then
  nonfree_downloaded=1
else
  echo "Failed to download RPM Fusion NONFREE repository release package." >&2
fi

if [[ "${free_downloaded}" -eq 1 && "${nonfree_downloaded}" -eq 1 ]]; then
  rpm-ostree install "${FREE_RPM}" "${NONFREE_RPM}"
else
  echo "Both RPM Fusion FREE and NONFREE packages must be downloaded successfully before installation." >&2
  exit 1
fi
