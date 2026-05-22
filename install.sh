#!/usr/bin/env bash
set -euo pipefail

REPO="coder11125/patchwork"
BINARY="patchwork"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
GITHUB_URL="https://github.com/${REPO}"

info()    { echo "==> $*"; }
warn()    { echo "!!> $*" >&2; }
err_exit(){ echo "!!> $*" >&2; exit 1; }

detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    *)       err_exit "unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *)             err_exit "unsupported arch: $(uname -m)" ;;
  esac
}

latest_release() {
  local tag
  tag=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  if [ -z "$tag" ]; then
    err_exit "could not determine latest release"
  fi
  echo "$tag"
}

download_binary() {
  local version="$1" os="$2" arch="$3"
  local filename="${BINARY}-${os}-${arch}"
  local download_url="${GITHUB_URL}/releases/download/${version}/${filename}"
  local tmpfile
  tmpfile=$(mktemp)

  info "Downloading ${download_url}"
  if ! curl -fsSL -o "$tmpfile" "$download_url"; then
    rm -f "$tmpfile"
    err_exit "download failed — check that release ${version} exists for ${os}/${arch}"
  fi

  echo "$tmpfile"
}

install_binary() {
  local tmpfile="$1"
  local dest="${INSTALL_DIR}/${BINARY}"

  if [ ! -d "$INSTALL_DIR" ]; then
    err_exit "install directory does not exist: ${INSTALL_DIR}"
  fi

  if ! install -m 0755 "$tmpfile" "$dest" 2>/dev/null; then
    info "Permission denied — retrying with sudo"
    if ! sudo install -m 0755 "$tmpfile" "$dest"; then
      err_exit "failed to install to ${dest}"
    fi
  fi

  rm -f "$tmpfile"
}

main() {
  local version="${1:-}"
  local os arch tmpfile

  os=$(detect_os)
  arch=$(detect_arch)

  info "Detected ${os}/${arch}"

  if [ -z "$version" ]; then
    version=$(latest_release)
  fi

  info "Installing ${BINARY} ${version}"

  tmpfile=$(download_binary "$version" "$os" "$arch")
  trap 'rm -f "$tmpfile"' EXIT

  install_binary "$tmpfile"

  info "Installed to $(command -v "$BINARY" 2>/dev/null || echo "${dest}")"
  info "Run 'patchwork --help' to get started"
}

main "$@"
