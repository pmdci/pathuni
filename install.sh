#!/bin/bash
set -e

# pathuni installer
# Usage: curl -sSL https://raw.githubusercontent.com/pmdci/pathuni/main/install.sh | bash

REPO="pmdci/pathuni"
INSTALL_DIR="$HOME/.local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[pathuni]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[pathuni]${NC} $1"
}

error() {
    echo -e "${RED}[pathuni]${NC} $1" >&2
}

# Detect platform and architecture
detect_platform() {
    local os arch
    
    case "$(uname -s)" in
        Darwin*)    os="Darwin" ;;
        Linux*)     os="Linux" ;;
        *)          error "Unsupported operating system: $(uname -s)"; exit 1 ;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64)   arch="x86_64" ;;
        arm64|aarch64)  arch="arm64" ;;
        armv7l)         arch="armv7" ;;
        armv6l)         arch="armv6" ;;
        i386|i686)      arch="i386" ;;
        *)              error "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac
    
    echo "${os}_${arch}"
}

# Get latest release version
get_latest_version() {
    curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
}

main() {
    log "Installing pathuni..."
    
    # Detect platform
    PLATFORM=$(detect_platform)
    log "Detected platform: $PLATFORM"
    
    # Get latest version
    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
        error "Failed to get latest version"
        exit 1
    fi
    log "Latest version: $VERSION"
    
    # Create install directory
    mkdir -p "$INSTALL_DIR"
    
    # Download binary
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/pathuni_${PLATFORM}.tar.gz"
    TEMP_FILE="/tmp/pathuni_${VERSION}.tar.gz"
    
    log "Downloading from: $DOWNLOAD_URL"
    if ! curl -sSL "$DOWNLOAD_URL" -o "$TEMP_FILE"; then
        error "Failed to download pathuni"
        exit 1
    fi
    
    # Extract and install
    log "Installing to $INSTALL_DIR/pathuni"
    if command -v tar >/dev/null 2>&1; then
        tar -xzf "$TEMP_FILE" -C /tmp
        cp "/tmp/pathuni" "$INSTALL_DIR/pathuni"
    else
        error "tar command not found"
        exit 1
    fi
    
    # Cleanup
    rm -f "$TEMP_FILE" "/tmp/pathuni"
    
    # Make executable
    chmod +x "$INSTALL_DIR/pathuni"
    
    # Verify installation
    if "$INSTALL_DIR/pathuni" --version >/dev/null 2>&1; then
        log "Successfully installed pathuni $VERSION"
        log "Location: $INSTALL_DIR/pathuni"
        
        # Check if install dir is in PATH
        if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
            warn "Add $INSTALL_DIR to your PATH to use pathuni from anywhere:"
            warn "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.bashrc"
            warn "  # (or ~/.zshrc, ~/.config/fish/config.fish, etc.)"
        fi
        
        log "Run 'pathuni --help' to get started!"
    else
        error "Installation failed - binary not working"
        exit 1
    fi
}

main "$@"