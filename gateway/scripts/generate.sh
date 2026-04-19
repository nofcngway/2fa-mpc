#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVICE_DIR="$(dirname "$SCRIPT_DIR")"
PROTO_DIR="$SERVICE_DIR/api"
OUT_DIR="$SERVICE_DIR/internal/pb"
SWAGGER_OUT="$OUT_DIR/swagger"

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"
mkdir -p "$SWAGGER_OUT"

PROTOS=$(find "$PROTO_DIR" -name "*.proto" ! -path "*/google/*")

protoc \
    --proto_path="$PROTO_DIR" \
    --go_out="$OUT_DIR" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$OUT_DIR" \
    --go-grpc_opt=paths=source_relative \
    --grpc-gateway_out="$OUT_DIR" \
    --grpc-gateway_opt=paths=source_relative \
    --grpc-gateway_opt=generate_unbound_methods=false \
    --openapiv2_out="$SWAGGER_OUT" \
    --openapiv2_opt=logtostderr=true \
    $PROTOS

echo "Proto generation complete for gateway"
