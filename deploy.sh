#!/bin/bash
set -e

VERSION=${1:-"v1.0.0"}
REPO_URL="${SUB2BALANCE_REPO_URL:-https://github.com/asdwsxzc123/sub2balance}"

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
if ! curl -fL -o "${ARCHIVE_NAME}" "${DOWNLOAD_URL}"; then
    echo "❌ Failed to download ${DOWNLOAD_URL}"
    echo "   Check that release ${VERSION} exists at ${REPO_URL}/releases"
    exit 1
fi

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
    echo "  - ADMIN_EMAIL"
    echo "  - ADMIN_PASSWORD"
    echo ""
fi

if [ ! -f config.yaml ] && [ -f config.yaml.example ]; then
    cp config.yaml.example config.yaml
    echo "📝 Created config.yaml from example"
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
echo "  1. Copy config.yaml.example to config.yaml if you haven't already"
echo "  2. Edit .env — set JWT_SECRET, ADMIN_EMAIL, ADMIN_PASSWORD"
echo "  3. Run: ./${BINARY_NAME}"
echo "  4. Access at http://localhost:8080 and log in as the admin you defined"
echo "  5. From the admin UI, set the Sub2API upstream at /admin/settings"
