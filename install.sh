#!/bin/sh
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Convert architecture to our naming scheme
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)
        echo "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# Convert OS to our naming scheme
case "$OS" in
    linux)   OS="linux" ;;
    darwin)  OS="darwin" ;;
    *)
        echo "${RED}Unsupported operating system: $OS${NC}"
        exit 1
        ;;
esac

# Check if Docker is installed
if ! command -v docker >/dev/null 2>&1; then
    echo "${RED}Error: Docker is not installed${NC}"
    echo "${YELLOW}Please install Docker first:${NC}"
    echo "  - For Linux: https://docs.docker.com/engine/install/"
    echo "  - For macOS: https://docs.docker.com/desktop/install/mac/"
    exit 1
fi

# Check if Docker daemon is running
if ! docker info >/dev/null 2>&1; then
    echo "${RED}Error: Docker daemon is not running${NC}"
    echo "${YELLOW}Please start Docker and try again${NC}"
    exit 1
fi

echo "${GREEN}Downloading latest release...${NC}"

# Get the latest release URL
LATEST_RELEASE_URL=$(curl -s https://api.github.com/repos/Automata-Labs-team/code-sandbox-mcp/releases/latest | grep "browser_download_url.*code-sandbox-mcp-$OS-$ARCH" | cut -d '"' -f 4)

if [ -z "$LATEST_RELEASE_URL" ]; then
    echo "${RED}Error: Could not find release for $OS-$ARCH${NC}"
    exit 1
fi

# Create installation directory
INSTALL_DIR="$HOME/.local/share/code-sandbox-mcp"
mkdir -p "$INSTALL_DIR"

# Download and install the binary
echo "${GREEN}Installing to $INSTALL_DIR/code-sandbox-mcp...${NC}"
curl -L "$LATEST_RELEASE_URL" -o "$INSTALL_DIR/code-sandbox-mcp"
chmod +x "$INSTALL_DIR/code-sandbox-mcp"

# Add to Claude Desktop config
echo "${GREEN}Adding to Claude Desktop configuration...${NC}"
"$INSTALL_DIR/code-sandbox-mcp" --install

echo "${GREEN}Installation complete!${NC}"
echo "You can now use code-sandbox-mcp with Claude Desktop or other AI applications." 