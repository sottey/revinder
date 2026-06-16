#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Testing revinder_bridge..."
(
  cd "$ROOT_DIR/revinder_bridge"
  go test -v ./...
)

echo "Testing revinder_reminders_consumer..."
(
  cd "$ROOT_DIR/consumers/revinder_reminders_consumer"
  go test -v ./...
)

echo "Testing revinder_memory_consumer..."
(
  cd "$ROOT_DIR/consumers/revinder_memory_consumer"
  go test -v ./...
)

echo "Tests complete."
