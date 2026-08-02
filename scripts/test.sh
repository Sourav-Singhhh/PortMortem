#!/usr/bin/env bash
# Port Mortem — One-Command Test Script (Linux/macOS)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "===> Executing Port Mortem full test suite..."
cd "${ROOT_DIR}/port"
go test -count=1 ./...
echo "===> All tests PASSED!"
