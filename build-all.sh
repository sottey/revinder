#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_DIR="$ROOT_DIR/build"

mkdir -p "$BUILD_DIR"

echo "Building revinder_bridge..."
(
  cd "$ROOT_DIR/revinder_bridge"
  go build -o "$BUILD_DIR/revinder_bridge" .
)

echo "Building revinder_task_consumer..."
(
  cd "$ROOT_DIR/consumers/revinder_task_consumer"
  go build -o "$BUILD_DIR/revinder_task_consumer" ./cmd/revinder_task_consumer
)

echo "Building revinder_memory_consumer..."
(
  cd "$ROOT_DIR/consumers/revinder_memory_consumer"
  go build -o "$BUILD_DIR/revinder_memory_consumer" ./cmd/revinder_memory_consumer
)

echo "Packaging revinder_alexa_skill Lambda..."
(
  cd "$ROOT_DIR/revinder_alexa_skill/lambda"
  zip -qr "$BUILD_DIR/lambda.zip" index.js package.json
)

echo "Build complete."
