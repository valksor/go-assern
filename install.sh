#!/bin/bash
#
# Assern Install Script
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/valksor/go-assern/master/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/valksor/go-assern/master/install.sh | bash -s -- --nightly
#   curl -fsSL https://raw.githubusercontent.com/valksor/go-assern/master/install.sh | bash -s -- -v v1.2.3
#

set -euo pipefail

REPO="valksor/go-assern"
BINARY_NAME="assern"
VERSION=""
NIGHTLY=false

# Minisign public key for binary verification
MINISIGN_PUBLIC_KEY="RWTFiZ4b+sgoFLiIMuMrTZr1mmropNlDsnwKl5RfoUtyUWUk4zyVpPw2"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[OK]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--version) VERSION="$2"; shift 2 ;;
        -n|--nightly) NIGHTLY=true; shift ;;
        -h|--help)
            echo "Usage: install.sh [OPTIONS]"
            echo "  -v, --version VERSION  Install specific version"
            echo "  -n, --nightly          Install nightly build"
            echo "  -h, --help             Show help"
            exit 0
            ;;
        *) error "Unknown option: $1" ;;
    esac
done

check_dependencies() {
    command -v curl &> /dev/null || error "curl required"
}

kill_running_processes() {
    if pgrep -x "$BINARY_NAME" >/dev/null 2>&1; then
        info "Stopping running $BINARY_NAME processes..."
        pkill -x "$BINARY_NAME" 2>/dev/null || true
        sleep 0.5
    fi
}

is_wsl() {
    [[ -n "${WSL_DISTRO_NAME:-}" ]] && return 0
    [[ -f /proc/version ]] && grep -qiE "(microsoft|wsl)" /proc/version 2>/dev/null && return 0
    return 1
}

detect_os() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$os" in
        linux) is_wsl && info "WSL environment detected"; echo "linux" ;;
        darwin) echo "darwin" ;;
        mingw*|msys*|cygwin*) error "Windows not directly supported. Use WSL or install.ps1" ;;
        *) error "Unsupported OS: $os" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *) error "Unsupported architecture: $(uname -m)" ;;
    esac
}

get_latest_version() {
    local version=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    [[ -z "$version" ]] && error "Failed to fetch latest version"
    echo "$version"
}

find_install_dir() {
    for dir in "$HOME/.local/bin" "$HOME/bin" "/usr/local/bin"; do
        [[ -d "$dir" && -w "$dir" ]] && echo "$dir" && return 0
        [[ "$dir" == "$HOME/.local/bin" || "$dir" == "$HOME/bin" ]] && mkdir -p "$dir" 2>/dev/null && echo "$dir" && return 0
    done
    echo "/usr/local/bin"
}

check_path() {
    local dir="$1"
    [[ ":$PATH:" == *":$dir:"* ]] && return
    warn "$dir is not in your PATH"
    echo "Add it: echo 'export PATH=\"$dir:\$PATH\"' >> ~/.$(basename $SHELL)rc"
}

verify_checksum() {
    local file="$1" expected="$2" actual
    if command -v sha256sum &> /dev/null; then
        actual=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum &> /dev/null; then
        actual=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        warn "No checksum tool available"; return 0
    fi
    [[ "$actual" != "$expected" ]] && error "Checksum mismatch"
    success "Checksum verified"
}

verify_minisign() {
    local base_url="$1" tmpdir="$2"
    [[ -z "$MINISIGN_PUBLIC_KEY" ]] && info "Minisign key not configured" && return 0
    command -v minisign &> /dev/null || { info "minisign not found"; return 0; }

    info "Verifying Minisign signature..."
    curl -fsSL "${base_url}/checksums.txt.minisig" -o "${tmpdir}/checksums.txt.minisig" 2>/dev/null || { warn "Signature not available"; return 0; }
    curl -fsSL "${base_url}/checksums.txt" -o "${tmpdir}/checksums.txt" 2>/dev/null || { warn "Checksums not available"; return 0; }

    if minisign -Vm "${tmpdir}/checksums.txt" -P "$MINISIGN_PUBLIC_KEY" -x "${tmpdir}/checksums.txt.minisig" &>/dev/null; then
        success "Minisign signature verified"
    else
        error "Signature verification failed!"
    fi
}

main() {
    echo ""
    echo "     _                          "
    echo "    / \\   ___ ___  ___ _ __ _ __  "
    echo "   / _ \\ / __/ __|/ _ \\ '__| '_ \\ "
    echo "  / ___ \\\\__ \\__ \\  __/ |  | | | |"
    echo " /_/   \\_\\___/___/\\___|_|  |_| |_|"
    echo ""
    echo "  MCP Aggregator with Project Configuration"
    echo ""

    check_dependencies
    local os=$(detect_os) arch=$(detect_arch)
    info "Platform: ${os}/${arch}"

    [[ "$NIGHTLY" == true ]] && VERSION="nightly" && info "Installing nightly"
    [[ -z "$VERSION" ]] && info "Fetching latest..." && VERSION=$(get_latest_version)
    info "Version: ${VERSION}"

    local binary_name="${BINARY_NAME}-${os}-${arch}"
    local base_url="https://github.com/${REPO}/releases/download/${VERSION}"
    local tmpdir=$(mktemp -d)
    trap "rm -rf '$tmpdir'" EXIT

    info "Downloading ${binary_name}..."
    curl -fsSL "${base_url}/${binary_name}" -o "${tmpdir}/${BINARY_NAME}" || error "Download failed"

    if curl -fsSL "${base_url}/${binary_name}.sha256" -o "${tmpdir}/checksum" 2>/dev/null; then
        verify_checksum "${tmpdir}/${BINARY_NAME}" "$(cat ${tmpdir}/checksum | awk '{print $1}')"
    fi

    verify_minisign "$base_url" "$tmpdir"

    chmod +x "${tmpdir}/${BINARY_NAME}"
    local install_dir=$(find_install_dir)
    kill_running_processes

    info "Installing to ${install_dir}/${BINARY_NAME}..."
    if [[ -w "$install_dir" ]]; then
        mv "${tmpdir}/${BINARY_NAME}" "${install_dir}/${BINARY_NAME}"
    else
        sudo mv "${tmpdir}/${BINARY_NAME}" "${install_dir}/${BINARY_NAME}"
    fi

    success "Installed ${BINARY_NAME} ${VERSION}"
    check_path "$install_dir"

    command -v "$BINARY_NAME" &> /dev/null && "$BINARY_NAME" version
    success "Done! Run '${BINARY_NAME} --help' to get started."
}

main "$@"
