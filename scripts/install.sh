#!/bin/bash
#
# gofile installer for Linux and macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/mars-base/gofile/main/install.sh | bash
#   or: curl -fsSL https://raw.githubusercontent.com/mars-base/gofile/main/install.sh | bash -s -- --version v1.0.0
#   or: curl -fsSL https://raw.githubusercontent.com/mars-base/gofile/main/install.sh | bash -s -- --prefix /usr/local/bin
#
set -euo pipefail

REPO="mars-base/gofile"
INSTALL_PREFIX="/usr/local/bin"
VERSION=""

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# --- Parse arguments ---
while [ $# -gt 0 ]; do
    case "$1" in
        --version)  VERSION="$2"; shift 2 ;;
        --prefix)   INSTALL_PREFIX="$2"; shift 2 ;;
        *)          log_error "Unknown option: $1"; exit 1 ;;
    esac
done

# --- Detect OS ---
detect_os() {
    local os
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        linux)  echo "linux" ;;
        darwin) echo "darwin" ;;
        *)      log_error "Unsupported OS: $os"; exit 1 ;;
    esac
}

# --- Detect architecture ---
detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)   echo "amd64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *)              log_error "Unsupported architecture: $arch"; exit 1 ;;
    esac
}

# --- Get latest version from GitHub ---
get_latest_version() {
    local api_url="https://api.github.com/repos/$REPO/releases/latest"
    local version
    version=$(curl -fsSL "$api_url" 2>/dev/null | grep -m1 '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')
    if [ -z "$version" ]; then
        log_error "Failed to fetch latest version from GitHub"
        exit 1
    fi
    echo "$version"
}

# --- Check dependencies ---
check_deps() {
    local missing=()
    command -v curl >/dev/null 2>&1 || missing+=("curl")
    command -v tar  >/dev/null 2>&1 || missing+=("tar")
    if [ ${#missing[@]} -gt 0 ]; then
        log_error "Missing dependencies: ${missing[*]}"
        log_info  "Install them first, then retry"
        exit 1
    fi
}

# --- Main ---
main() {
    echo -e "${CYAN}gofile installer${NC}"
    echo ""

    check_deps

    local os arch
    os=$(detect_os)
    arch=$(detect_arch)

    # Determine binary name based on OS/arch
    local binary_name
    if [ "$os" = "linux" ]; then
        binary_name="gofile-linux"
    elif [ "$os" = "darwin" ]; then
        if [ "$arch" = "arm64" ]; then
            binary_name="gofile-darwin-arm64"
        else
            binary_name="gofile-darwin-amd64"
        fi
    fi

    # Get version
    if [ -z "$VERSION" ]; then
        log_info "Fetching latest version..."
        VERSION=$(get_latest_version)
    fi
    log_info "Version: $VERSION"
    log_info "Platform: $os/$arch"
    log_info "Binary: $binary_name"

    # Determine download filename
    local archive_name
    if [ "$os" = "linux" ]; then
        archive_name="gofile-linux.tar.gz"
    elif [ "$os" = "darwin" ]; then
        if [ "$arch" = "arm64" ]; then
            archive_name="gofile-darwin-arm64.tar.gz"
        else
            archive_name="gofile-darwin-amd64.tar.gz"
        fi
    fi

    local download_url="https://github.com/$REPO/releases/download/$VERSION/$archive_name"
    log_info "Downloading: $download_url"

    # Download
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT

    if ! curl -fSL --progress-bar -o "$tmp_dir/$archive_name" "$download_url"; then
        log_error "Download failed: $download_url"
        exit 1
    fi

    # Extract
    log_info "Extracting..."
    tar -xzf "$tmp_dir/$archive_name" -C "$tmp_dir"

    # Install
    local install_path="$INSTALL_PREFIX/gofile"
    log_info "Installing to: $install_path"

    mkdir -p "$INSTALL_PREFIX"
    cp "$tmp_dir/$binary_name" "$install_path"
    chmod +x "$install_path"

    # Verify
    if [ -x "$install_path" ]; then
        local installed_version
        installed_version=$("$install_path" -v 2>&1 | head -1)
        echo ""
        log_info "Installation successful!"
        log_info "  Binary: $install_path"
        log_info "  $installed_version"
        echo ""

        # Check if install path is in PATH
        if ! echo "$PATH" | tr ':' '\n' | grep -q "^${INSTALL_PREFIX}$"; then
            log_warn "$INSTALL_PREFIX is not in your PATH"
            log_info "Add it to your shell config:"
            echo "  export PATH=\"$INSTALL_PREFIX:\$PATH\""
        fi
    else
        log_error "Installation failed"
        exit 1
    fi
}

main
