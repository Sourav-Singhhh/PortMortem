#!/usr/bin/env bash
# Port Mortem — One-Command GoDoc Examples Script (Linux/macOS)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "===> Executing Port Mortem runnable example tests..."
cd "${ROOT_DIR}/port"
go test -v -run="^Example" .
echo "===> Example tests PASSED!"
