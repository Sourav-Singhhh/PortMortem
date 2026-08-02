#!/usr/bin/env bash
# Port Mortem — One-Command Build Script (Linux/macOS)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "===> Building Port Mortem package module..."
cd "${ROOT_DIR}/port"
go build ./...
echo "===> Build SUCCESSFUL!"
