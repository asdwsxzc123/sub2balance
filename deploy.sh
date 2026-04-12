#!/bin/bash
set -e

VERSION=${1:-"v1.0.0"}
REPO_URL="https://github.com/YOUR_USERNAME/sub2balance"

echo "🚀 Deploying Sub2Balance ${VERSION}"
echo ""

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "❌ Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

BINARY_NAME="sub2balance-${OS}-${ARCH}"
ARCHIVE_NAME="${BINARY_NAME}.tar.gz"

if [ "$OS" = "windows" ]; then
    ARCHIVE_NAME="${BINARY_NAME}.zip"
fi

echo "📦 Detected platform: ${OS}-${ARCH}"
echo "📥 Downloading ${ARCHIVE_NAME}..."

# Download release
DOWNLOAD_URL="${REPO_URL}/releases/download/${VERSION}/${ARCHIVE_NAME}"
curl -L -o "${ARCHIVE_NAME}" "${DOWNLOAD_URL}"

# Extract
echo "📂 Extracting..."
if [ "$OS" = "windows" ]; then
    unzip -q "${ARCHIVE_NAME}"
else
    tar xzf "${ARCHIVE_NAME}"
fi

# Setup
echo "⚙️  Setting up..."
chmod +x "${BINARY_NAME}"

if [ ! -f .env ]; then
    cp .env.example .env
    echo "📝 Created .env file - please configure it before running"
    echo ""
    echo "Required environment variables:"
    echo "  - JWT_SECRET"
    echo "  - SUB2API_URL"
    echo "  - SUB2API_API_KEY"
    echo ""
fi

# Create systemd service (optional)
if [ "$OS" = "linux" ] && [ -d /etc/systemd/system ]; then
    read -p "📋 Create systemd service? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        sudo tee /etc/systemd/system/sub2balance.service > /dev/null <<EOF
[Unit]
Description=Sub2Balance Service
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$(pwd)
EnvironmentFile=$(pwd)/.env
ExecStart=$(pwd)/${BINARY_NAME}
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
        sudo systemctl daemon-reload
        sudo systemctl enable sub2balance
        echo "✅ Systemd service created"
        echo "   Start with: sudo systemctl start sub2balance"
        echo "   View logs: sudo journalctl -u sub2balance -f"
    fi
fi

# Cleanup
rm -f "${ARCHIVE_NAME}"

echo ""
echo "✅ Deployment complete!"
echo ""
echo "Next steps:"
echo "  1. Edit .env and configure required variables"
echo "  2. Run: ./${BINARY_NAME}"
echo "  3. Access at http://localhost:8080"
echo ""
echo "Default admin credentials:"
echo "  Email: admin@sub2balance.local"
echo "  Password: admin123"
