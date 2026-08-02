#!/usr/bin/env bash
# Port Mortem — One-Command Differential Fuzz Survivor Script (Linux/macOS)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "===> Executing Differential Fuzz Survivor Engine (30s)..."
cd "${ROOT_DIR}/port"
go run ./fuzz_survivor -duration=30s
echo "===> Survivor test run COMPLETE!"
