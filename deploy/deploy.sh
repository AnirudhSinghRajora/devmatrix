#!/usr/bin/env bash
set -euo pipefail

# DevMatrix deployment script.
# Usage: ./deploy.sh [host]
#   host: SSH target (default: devmatrix-server)

HOST="${1:-devmatrix-server}"
REMOTE_DIR="/home/Anirudh/devmatrix"
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "=== DevMatrix Deploy ==="
echo "Host:    $HOST"
echo "Source:  $PROJECT_ROOT"

# 1. Build client.
echo ""
echo "--- Building client ---"
cd "$PROJECT_ROOT/client"
npm ci --production=false
npx vite build
echo "Client build complete: dist/"

# 2. Build server.
echo ""
echo "--- Building server ---"
cd "$PROJECT_ROOT/server"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o devmatrix ./cmd/devmatrix
echo "Server binary: devmatrix"

# 3. Upload artifacts.
echo ""
echo "--- Uploading ---"
ssh "$HOST" "mkdir -p $REMOTE_DIR/server $REMOTE_DIR/client"
scp "$PROJECT_ROOT/server/devmatrix" "$HOST:$REMOTE_DIR/server/devmatrix"
rsync -az --delete "$PROJECT_ROOT/client/dist/" "$HOST:$REMOTE_DIR/client/dist/"

# 4. Upload service and config files (only if they don't exist on remote).
scp "$PROJECT_ROOT/deploy/devmatrix.service" "$HOST:/tmp/devmatrix.service"
ssh "$HOST" "
  sudo mv /tmp/devmatrix.service /etc/systemd/system/devmatrix.service
  sudo systemctl daemon-reload
"

# 5. Upload Caddyfile.
scp "$PROJECT_ROOT/deploy/Caddyfile" "$HOST:/tmp/Caddyfile"
ssh "$HOST" "
  sudo mv /tmp/Caddyfile /etc/caddy/Caddyfile
  sudo systemctl reload caddy || true
"

# 6. Restart server.
echo ""
echo "--- Restarting server ---"
ssh "$HOST" "sudo systemctl restart devmatrix"

# 7. Health check.
echo ""
echo "--- Health check ---"
sleep 2
ssh "$HOST" "curl -sf http://localhost:8080/health" && echo " OK" || echo " FAILED"

echo ""
echo "=== Deploy complete ==="
