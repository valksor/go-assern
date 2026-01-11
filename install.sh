#!/bin/bash
set -e

# Valksor Assern Install Script
# Usage: curl -fsSL https://raw.githubusercontent.com/valksor/go-assern/main/install.sh | bash

REPO="valksor/go-assern"
BINARY_NAME="assern"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Detect OS and architecture
detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "$OS" in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        *)
            error "Unsupported OS: $OS"
            ;;
    esac

    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            error "Unsupported architecture: $ARCH"
            ;;
    esac

    PLATFORM="${OS}-${ARCH}"
    info "Detected platform: $PLATFORM"
}

# Get latest version from GitHub
get_latest_version() {
    VERSION="${VERSION:-}"
    if [ -z "$VERSION" ]; then
        info "Fetching latest version..."
        VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
        if [ -z "$VERSION" ]; then
            error "Failed to fetch latest version"
        fi
    fi
    info "Version: $VERSION"
}

# Download and install binary
install_binary() {
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}-${PLATFORM}"
    CHECKSUM_URL="${DOWNLOAD_URL}.sha256"

    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    info "Downloading $BINARY_NAME..."
    curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$BINARY_NAME"

    # Verify checksum if available
    if curl -fsSL "$CHECKSUM_URL" -o "$TMP_DIR/checksum.sha256" 2>/dev/null; then
        info "Verifying checksum..."
        cd "$TMP_DIR"
        if command -v sha256sum &> /dev/null; then
            echo "$(cat checksum.sha256)  $BINARY_NAME" | sha256sum -c - || error "Checksum verification failed"
        elif command -v shasum &> /dev/null; then
            echo "$(cat checksum.sha256)  $BINARY_NAME" | shasum -a 256 -c - || error "Checksum verification failed"
        else
            warn "No checksum tool available, skipping verification"
        fi
        cd - > /dev/null
    else
        warn "Checksum file not available, skipping verification"
    fi

    # Create install directory if needed
    if [ ! -d "$INSTALL_DIR" ]; then
        info "Creating install directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR"
    fi

    # Install binary
    info "Installing to $INSTALL_DIR/$BINARY_NAME..."
    chmod +x "$TMP_DIR/$BINARY_NAME"
    mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"

    info "Installation complete!"
}

# Check if install dir is in PATH
check_path() {
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        warn "$INSTALL_DIR is not in your PATH"
        echo ""
        echo "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo ""
        echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
        echo ""
    fi
}

# Verify installation
verify_install() {
    if command -v "$BINARY_NAME" &> /dev/null; then
        info "Verification: $($BINARY_NAME version | head -1)"
    else
        info "Run '$INSTALL_DIR/$BINARY_NAME version' to verify installation"
    fi
}

main() {
    echo ""
    echo "Valksor Assern Installer"
    echo "========================"
    echo ""

    detect_platform
    get_latest_version
    install_binary
    check_path
    verify_install

    echo ""
    info "To get started, run: $BINARY_NAME config init"
    echo ""
}

main "$@"
