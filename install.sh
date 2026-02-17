#!/usr/bin/env bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPO="vansdevcode/worktree-manager"
BINARY_NAME="wtm"
GH_EXTENSION_NAME="gh-wtm"
INSTALL_DIR="$HOME/.local/bin"
TEMP_DIR=$(mktemp -d)

# Helper functions
info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1" >&2
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

cleanup() {
    rm -rf "$TEMP_DIR"
}

trap cleanup EXIT

# Detect OS and architecture
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
    case "$os" in
        linux) OS="linux" ;;
        darwin) OS="darwin" ;;
        *) error "Unsupported OS: $os"; exit 1 ;;
    esac
    
    case "$arch" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $arch"; exit 1 ;;
    esac
}

# Get latest release version
get_latest_version() {
    info "Fetching latest release..."
    VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        error "Failed to fetch latest version"
        exit 1
    fi
    
    success "Latest version: $VERSION"
}

# Calculate SHA256 checksum (cross-platform)
calculate_sha256() {
    local file="$1"
    
    if command -v sha256sum >/dev/null 2>&1; then
        # Linux
        sha256sum "$file" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        # macOS
        shasum -a 256 "$file" | awk '{print $1}'
    else
        error "Neither sha256sum nor shasum found. Cannot verify checksum."
        return 1
    fi
}

# Verify checksum
verify_checksum() {
    local file="$1"
    local expected_checksum="$2"
    
    info "Verifying checksum..."
    
    # Calculate actual checksum
    local actual_checksum=$(calculate_sha256 "$file")
    
    if [ -z "$actual_checksum" ]; then
        error "Failed to calculate checksum"
        return 1
    fi
    
    if [ "$actual_checksum" != "$expected_checksum" ]; then
        error "Checksum verification failed!"
        error "  Expected: $expected_checksum"
        error "  Got:      $actual_checksum"
        error "This could indicate a compromised download or network tampering."
        return 1
    fi
    
    success "Checksum verified successfully"
    return 0
}

# Download and install binary
install_standalone() {
    info "Installing standalone wtm command to $INSTALL_DIR..."
    
    # Create install directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"
    
    # Download checksums file
    local checksums_url="https://github.com/$REPO/releases/download/$VERSION/checksums.txt"
    info "Downloading checksums..."
    
    if ! curl -fsSL "$checksums_url" -o "$TEMP_DIR/checksums.txt"; then
        error "Failed to download checksums file. Building from source instead..."
        build_from_source
        return
    fi
    
    # Extract expected checksum for this binary
    local binary_filename="wtm_${OS}_${ARCH}"
    local expected_checksum=$(grep "^[a-f0-9]+  ${binary_filename}$" "$TEMP_DIR/checksums.txt" | awk '{print $1}')
    
    if [ -z "$expected_checksum" ]; then
        error "Could not find checksum for $binary_filename. Building from source instead..."
        build_from_source
        return
    fi
    
    # Download binary
    local download_url="https://github.com/$REPO/releases/download/$VERSION/$binary_filename"
    info "Downloading from $download_url..."
    
    if ! curl -fsSL "$download_url" -o "$TEMP_DIR/$BINARY_NAME"; then
        error "Failed to download binary. Building from source instead..."
        build_from_source
        return
    fi
    
    # Verify checksum before installing
    if ! verify_checksum "$TEMP_DIR/$BINARY_NAME" "$expected_checksum"; then
        error "Installation aborted due to checksum mismatch."
        error "Building from source instead..."
        build_from_source
        return
    fi
    
    # Install binary
    chmod +x "$TEMP_DIR/$BINARY_NAME"
    mv "$TEMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    
    success "Installed wtm to $INSTALL_DIR/$BINARY_NAME"
    
    # Check if in PATH (ensuring full PATH entry match)
    case ":$PATH:" in
        *":$INSTALL_DIR:"*) ;;
        *)
            warning "$INSTALL_DIR is not in your PATH"
            echo "  Add this to your ~/.bashrc or ~/.zshrc:"
            echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
            ;;
    esac
}

# Build from source as fallback
build_from_source() {
    info "Building from source..."
    
    # Check for Go
    if ! command -v go >/dev/null 2>&1; then
        error "Go is not installed. Please install Go 1.22+ or download a pre-built binary"
        exit 1
    fi
    
    # Clone and build
    cd "$TEMP_DIR"
    git clone "https://github.com/$REPO.git"
    cd worktree-manager
    go build -o "$BINARY_NAME" cmd/wtm/*.go
    
    # Install
    chmod +x "$BINARY_NAME"
    mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    
    success "Built and installed wtm from source"
}

# Install as GitHub CLI extension
install_gh_extension() {
    info "Installing as GitHub CLI extension (gh wtm)..."
    
    # Check if standalone binary exists
    if [ ! -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        error "Standalone binary not found at $INSTALL_DIR/$BINARY_NAME"
        error "Please install standalone version first"
        return 1
    fi
    
    # Create permanent folder for gh extension
    local ext_dir="$HOME/.local/share/gh-wtm"
    mkdir -p "$ext_dir"
    
    # Create symlink to the standalone binary
    ln -sf "$INSTALL_DIR/$BINARY_NAME" "$ext_dir/$GH_EXTENSION_NAME"
    
    # Install from the permanent folder (gh creates symlink to this directory)
    if (cd "$ext_dir" && gh extension install .); then
        success "Installed as gh extension (use: gh wtm)"
    else
        error "Failed to install gh extension"
        return 1
    fi
}

# Prompt for yes/no with fallback for non-interactive mode
prompt_yes_no() {
    local prompt="$1"
    local default="${2:-Y}"  # Default to Y if not specified
    
    # Check if stdin is a terminal
    if [ -t 0 ]; then
        read -p "$(echo -e ${BLUE}?${NC}) $prompt " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
            return 0
        else
            return 1
        fi
    else
        # Non-interactive mode: use default
        info "$prompt [non-interactive: using default: $default]"
        if [[ $default =~ ^[Yy]$ ]]; then
            return 0
        else
            return 1
        fi
    fi
}

# Main installation flow
main() {
    echo ""
    echo "╔═══════════════════════════════════════════════════════════╗"
    echo "║         Worktree Manager Installer                       ║"
    echo "╚═══════════════════════════════════════════════════════════╝"
    echo ""
    
    info "This tool can be used in two ways:"
    echo "  1. Standalone command: wtm"
    echo "  2. GitHub CLI extension: gh wtm"
    echo ""
    
    # Support environment variables for non-interactive installation
    # INSTALL_WTM=yes|no - install standalone binary
    # INSTALL_GH_WTM=yes|no - install gh extension
    
    detect_platform
    get_latest_version
    
    # Ask about standalone installation
    echo ""
    local install_standalone="${INSTALL_WTM:-}"
    if [ -n "$install_standalone" ]; then
        info "Using INSTALL_WTM=$install_standalone (from environment)"
        if [[ $install_standalone =~ ^[Yy]|[Yy][Ee][Ss]$ ]]; then
            install_standalone
        fi
    elif prompt_yes_no "Install standalone 'wtm' command? [Y/n]"; then
        install_standalone
    fi
    
    # Check for gh CLI
    if command -v gh >/dev/null 2>&1; then
        echo ""
        local install_gh="${INSTALL_GH_WTM:-}"
        if [ -n "$install_gh" ]; then
            info "Using INSTALL_GH_WTM=$install_gh (from environment)"
            if [[ $install_gh =~ ^[Yy]|[Yy][Ee][Ss]$ ]]; then
                install_gh_extension
            fi
        elif prompt_yes_no "Install as GitHub CLI extension 'gh wtm'? [Y/n]"; then
            install_gh_extension
        fi
    else
        echo ""
        info "GitHub CLI (gh) not found. Skipping extension installation."
        echo "  Install gh CLI from: https://cli.github.com/"
    fi
    
    # Final message
    echo ""
    echo "╔═══════════════════════════════════════════════════════════╗"
    echo "║         Installation Complete! 🎉                        ║"
    echo "╚═══════════════════════════════════════════════════════════╝"
    echo ""
    
    if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        success "Standalone: Use 'wtm --help' to get started"
    fi
    
    if gh extension list 2>/dev/null | grep -q "$GH_EXTENSION_NAME"; then
        success "Extension: Use 'gh wtm --help' to get started"
    fi
    
    echo ""
    info "Documentation: https://github.com/$REPO"
    echo ""
}

main "$@"
