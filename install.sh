#!/usr/bin/env bash
set -euo pipefail

REPO="coder11125/patchwork"
BINARY="patchwork"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
GITHUB_URL="https://github.com/${REPO}"

info()    { echo "==> $*"; }
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
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

main() {
  local version="${1:-}"
  local os arch dest

  os=$(detect_os)
  arch=$(detect_arch)
  dest="${INSTALL_DIR}/${BINARY}"

  info "Detected ${os}/${arch}"

  if [ -z "$version" ]; then
    version=$(latest_release)
  fi

  info "Installing ${BINARY} ${version}"

  local filename="${BINARY}-${os}-${arch}"
  local download_url="${GITHUB_URL}/releases/download/${version}/${filename}"

  info "Downloading ${download_url}"

  if curl -fsSL -o "$dest" "$download_url" 2>/dev/null; then
    chmod 0755 "$dest"
  elif sudo curl -fsSL -o "$dest" "$download_url" 2>/dev/null; then
    sudo chmod 0755 "$dest"
  else
    err_exit "download failed"
  fi

  info "Installed to ${dest}"
  info "Run '${BINARY} --help' to get started"
}

main "$@"
