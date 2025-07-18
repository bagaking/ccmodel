#!/bin/bash
# ccmodel installer script
# Usage: curl -sSL https://raw.githubusercontent.com/bagaking/ccmodel/main/install.sh | bash

set -e

REPO="bagaking/ccmodel"
BINARY_NAME="ccmodel"
INSTALL_DIR="/usr/local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Emojis
ROCKET="🚀"
SPARKLES="✨"
FIRE="🔥"
CHECK="✅"
ERROR="❌"
INFO="ℹ️"

print_banner() {
    echo -e "${PURPLE}"
    echo "   ____  ____     __  __  ____  ____  _____ __    "
    echo "  /  __\/  __\   /  \/  \/  _ \/  _ \/  __// /    "
    echo " |  \/||  \/|   |  \/  || / \|| | \||  \  | |    "
    echo " |  __/|  __/   |  ||  || \_/|| |_/||  /_ | |_/\ "
    echo " \_/   \_/      \_/  \/ \____/\____/\____\\____/ "
    echo -e "${NC}"
    echo -e "${CYAN}        AI Model Configuration Switcher${NC}"
    echo -e "${YELLOW}        github.com/bagaking/ccmodel${NC}"
    echo ""
}

get_latest_release() {
    echo "$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)"
}

detect_platform() {
    local os arch
    
    case "$(uname -s)" in
        Darwin*)    os="darwin" ;;
        Linux*)     os="linux" ;;
        CYGWIN*|MINGW*|MSYS*) os="windows" ;;
        *)          echo -e "${ERROR} Unsupported OS: $(uname -s)" && exit 1 ;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        arm64|aarch64)  arch="arm64" ;;
        *)              echo -e "${ERROR} Unsupported architecture: $(uname -m)" && exit 1 ;;
    esac
    
    echo "${os}-${arch}"
}

main() {
    print_banner
    
    echo -e "${INFO} Detecting platform..."
    PLATFORM=$(detect_platform)
    echo -e "${CHECK} Platform: ${GREEN}$PLATFORM${NC}"
    
    echo -e "${INFO} Getting latest release..."
    VERSION=$(get_latest_release)
    if [ -z "$VERSION" ]; then
        echo -e "${ERROR} Failed to get latest release"
        exit 1
    fi
    echo -e "${CHECK} Latest version: ${GREEN}$VERSION${NC}"
    
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/${BINARY_NAME}-$PLATFORM"
    if [[ "$PLATFORM" == *"windows"* ]]; then
        DOWNLOAD_URL="${DOWNLOAD_URL}.exe"
        BINARY_NAME="${BINARY_NAME}.exe"
    fi
    
    echo -e "${ROCKET} Downloading $BINARY_NAME..."
    echo -e "${BLUE}$DOWNLOAD_URL${NC}"
    
    TEMP_FILE=$(mktemp)
    trap "rm -f $TEMP_FILE" EXIT
    
    if command -v curl >/dev/null 2>&1; then
        curl -L -o "$TEMP_FILE" "$DOWNLOAD_URL"
    elif command -v wget >/dev/null 2>&1; then
        wget -O "$TEMP_FILE" "$DOWNLOAD_URL"
    else
        echo -e "${ERROR} Please install curl or wget"
        exit 1
    fi
    
    echo -e "${SPARKLES} Installing to $INSTALL_DIR..."
    
    if [ ! -w "$INSTALL_DIR" ]; then
        echo -e "${INFO} Installing with sudo..."
        sudo install -m 755 "$TEMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
    else
        install -m 755 "$TEMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
    fi
    
    echo -e "${CHECK} ${GREEN}Installation completed!${NC}"
    echo ""
    echo -e "${FIRE} Quick start:"
    echo -e "  ${CYAN}$BINARY_NAME list${NC}     - List available models"
    echo -e "  ${CYAN}$BINARY_NAME current${NC}  - Show current model"
    echo -e "  ${CYAN}$BINARY_NAME switch${NC} <model>  - Switch to model"
    echo ""
    echo -e "${SPARKLES} Enable shell completion:"
    echo -e "  ${YELLOW}# For zsh${NC}"
    echo -e "  echo 'eval \"\$($BINARY_NAME completion zsh)\"' >> ~/.zshrc"
    echo ""
    echo -e "  ${YELLOW}# For bash${NC}"
    echo -e "  echo 'eval \"\$($BINARY_NAME completion bash)\"' >> ~/.bashrc"
    echo ""
    echo -e "${INFO} Documentation: ${BLUE}https://github.com/$REPO${NC}"
    echo -e "${ROCKET} Happy AI model switching!"
}

main "$@"