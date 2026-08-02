#!/usr/bin/env bash
# Port Mortem — One-Command Unmodified Original Test Suite Script (Linux/macOS)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "===> Building Go adapter for original picomatch test suite..."
cd "${ROOT_DIR}/port"
go build -o ../tests/go_adapter ./cmd/adapter

echo "===> Executing unmodified original picomatch test suite..."
cd "${ROOT_DIR}/tests/original"
NODE_PATH="../../original-picomatch/picomatch-master/node_modules" node "../../original-picomatch/picomatch-master/node_modules/mocha/bin/mocha.js" "*.js"
echo "===> Original test suite execution COMPLETE!"
