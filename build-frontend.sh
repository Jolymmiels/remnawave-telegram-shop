#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRONTEND_DIR="${SCRIPT_DIR}/tg-admin"
DIST_DIR="${FRONTEND_DIR}/dist"

echo "Building frontend..."

cd "${FRONTEND_DIR}"

if [ ! -d "node_modules" ]; then
  echo "Installing dependencies..."
  npm ci
fi

echo "Running build..."
pnpm build

echo ""
echo "Done! Frontend built to: ${DIST_DIR}"
echo "Size: $(du -sh "${DIST_DIR}" | cut -f1)"
