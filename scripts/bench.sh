#!/usr/bin/env bash
# Port Mortem — One-Command Benchmark Script (Linux/macOS)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "===> Executing Port Mortem performance benchmarks..."
cd "${ROOT_DIR}/port"
go test -run="^$" -bench="." -benchmem -count=1 .
echo "===> Benchmark execution COMPLETE!"
