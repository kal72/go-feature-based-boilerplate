#!/usr/bin/env bash
set -euo pipefail

# ─── Configuration ────────────────────────────────────────────────────────────
PROTO_DIR="api/proto"
EXTERNAL_PROTO_DIR="api/external"
GEN_GO_DIR="gen/pb"
GEN_EXTERNAL_GO_DIR="gen/externalpb"
GEN_SWAGGER_DIR="gen/openapi"

# Resolve project root (scripts/ lives one level below root)
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

# ─── Dependency check & auto-install ──────────────────────────────────────────
echo "==> Checking protoc plugins..."

command -v protoc >/dev/null 2>&1 || {
  echo "ERROR: protoc not found. Install via https://grpc.io/docs/protoc-installation/"
  exit 1
}

install_if_missing() {
  local bin="$1" pkg="$2"
  if ! command -v "$bin" >/dev/null 2>&1; then
    echo "    Installing $bin..."
    go install "$pkg"
  fi
}

install_if_missing protoc-gen-go          "google.golang.org/protobuf/cmd/protoc-gen-go@latest"
install_if_missing protoc-gen-go-grpc     "google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
install_if_missing protoc-gen-grpc-gateway "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest"
install_if_missing protoc-gen-openapiv2   "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest"

# ─── Prepare output directories ──────────────────────────────────────────────
echo "==> Cleaning previous generated files..."
rm -rf "$GEN_GO_DIR" "$GEN_EXTERNAL_GO_DIR" "$GEN_SWAGGER_DIR"
mkdir -p "$GEN_GO_DIR" "$GEN_EXTERNAL_GO_DIR" "$GEN_SWAGGER_DIR"

# ─── 1. Inbound Service Protos (gRPC + Gateway + Swagger) ─────────────────────
PROTO_FILES=$(find "$PROTO_DIR" -name '*.proto')

if [ -n "$PROTO_FILES" ]; then
  echo "==> Generating Inbound Go code + gRPC + Gateway + Swagger..."

  GATEWAY_PROTO="$(go env GOPATH)/pkg/mod/github.com/grpc-ecosystem/grpc-gateway/v2@v2.29.0"
  PLUGINS_DIR="$PROTO_DIR/plugins"

  protoc \
    --proto_path="$PLUGINS_DIR" \
    --proto_path="$PROTO_DIR" \
    --proto_path="$GATEWAY_PROTO" \
    --go_out="$GEN_GO_DIR" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$GEN_GO_DIR" \
    --go-grpc_opt=paths=source_relative \
    --grpc-gateway_out="$GEN_GO_DIR" \
    --grpc-gateway_opt=paths=source_relative \
    --openapiv2_out="$GEN_SWAGGER_DIR" \
    --openapiv2_opt=logtostderr=true \
    --openapiv2_opt=allow_merge=true \
    --openapiv2_opt=merge_file_name=api \
    $PROTO_FILES
else
  echo "WARNING: No inbound .proto files found in $PROTO_DIR"
fi

# ─── 2. External Client Protos (gRPC Client ONLY: No Gateway, No Swagger) ──────
if [ -d "$EXTERNAL_PROTO_DIR" ]; then
  EXTERNAL_PROTO_FILES=$(find "$EXTERNAL_PROTO_DIR" -name '*.proto')
  if [ -n "$EXTERNAL_PROTO_FILES" ]; then
    echo "==> Generating External Client Go code (Client stubs only)..."
    mkdir -p "$GEN_EXTERNAL_GO_DIR"
    protoc \
      --proto_path="$EXTERNAL_PROTO_DIR" \
      --go_out="$GEN_EXTERNAL_GO_DIR" \
      --go_opt=paths=source_relative \
      --go-grpc_out="$GEN_EXTERNAL_GO_DIR" \
      --go-grpc_opt=paths=source_relative \
      $EXTERNAL_PROTO_FILES
  fi
fi

# ─── Generate openapi.go wrapper ─────────────────────────────────────────────
cat << 'EOF' > "$GEN_SWAGGER_DIR/openapi.go"
package openapi

import _ "embed"

// Spec contains the raw JSON bytes of the generated OpenAPI / Swagger spec.
//
//go:embed api.swagger.json
var Spec []byte
EOF

# ─── Summary ──────────────────────────────────────────────────────────────────
echo ""
echo "==> Done!"
echo "    Inbound Go code  → $GEN_GO_DIR/"
echo "    External Go code → $GEN_EXTERNAL_GO_DIR/"
echo "    Inbound Swagger  → $GEN_SWAGGER_DIR/api.swagger.json"
echo ""
echo "    Generated files:"
find "$GEN_GO_DIR" -type f -name '*.go' | sed 's/^/      /'
find "$GEN_EXTERNAL_GO_DIR" -type f -name '*.go' 2>/dev/null | sed 's/^/      /' || true
find "$GEN_SWAGGER_DIR" -type f -name '*.json' | sed 's/^/      /'
find "$GEN_SWAGGER_DIR" -type f -name 'openapi.go' | sed 's/^/      /'
