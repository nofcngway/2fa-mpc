#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVICE_DIR="$(dirname "$SCRIPT_DIR")"
PROTO_DIR="$SERVICE_DIR/api"
OUT_DIR="$SERVICE_DIR/internal/pb"

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

protoc \
    --proto_path="$PROTO_DIR" \
    --go_out="$OUT_DIR" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$OUT_DIR" \
    --go-grpc_opt=paths=source_relative \
    $(find "$PROTO_DIR" -name "*.proto")

echo "Proto generation complete for twofa"
