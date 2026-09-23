#!/usr/bin/env bash
# ==============================================================================
# Go Module Renaming Utility
# Renames the Go module path, package imports, proto options, and environment
# variables throughout the entire repository when starting a new project.
#
# Usage:
#   ./scripts/rename-module.sh <new-module-name>
#   make rename module=<new-module-name>
# ==============================================================================

set -euo pipefail

# Resolve project root
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

# ─── 1. Determine Old and New Module Names ───────────────────────────────────
if [ ! -f "go.mod" ]; then
    echo "❌ Error: go.mod not found in $ROOT_DIR"
    exit 1
fi

OLD_MODULE=$(grep -E '^module[[:space:]]+' go.mod | head -n 1 | awk '{print $2}')
if [ -z "$OLD_MODULE" ]; then
    echo "❌ Error: Could not determine current module name from go.mod"
    exit 1
fi

NEW_MODULE="${1:-}"
if [ -z "$NEW_MODULE" ]; then
    echo "Current module: \033[36m$OLD_MODULE\033[0m"
    printf "Enter new module name (e.g. github.com/username/my-service): "
    read -r NEW_MODULE
fi

# Trim whitespace
NEW_MODULE=$(echo "$NEW_MODULE" | xargs)

if [ -z "$NEW_MODULE" ]; then
    echo "❌ Error: New module name cannot be empty."
    exit 1
fi

if [ "$OLD_MODULE" = "$NEW_MODULE" ]; then
    echo "ℹ️  Module is already named '$NEW_MODULE'. Nothing to do."
    exit 0
fi

# Derive application/service name (last element of module path)
SERVICE_NAME="$(basename "$NEW_MODULE")"

echo ""
echo "================================================================================"
echo "  Renaming Go Module:"
echo "    Old: \033[33m$OLD_MODULE\033[0m"
echo "    New: \033[32m$NEW_MODULE\033[0m (Service: $SERVICE_NAME)"
echo "================================================================================"
echo ""

# ─── 2. Replacement Helper Function ──────────────────────────────────────────
replace_in_file() {
    local target="$1"
    local from="$2"
    local to="$3"

    if [ ! -f "$target" ]; then
        return
    fi

    if command -v perl >/dev/null 2>&1; then
        perl -pi -e "s|\Q$from\E|$to|g" "$target"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        local esc_from esc_to
        esc_from=$(printf '%s\n' "$from" | sed 's/[[\.*^$/]/\\&/g')
        esc_to=$(printf '%s\n' "$to" | sed 's/[\/&]/\\&/g')
        sed -i '' "s|${esc_from}|${esc_to}|g" "$target"
    else
        local esc_from esc_to
        esc_from=$(printf '%s\n' "$from" | sed 's/[[\.*^$/]/\\&/g')
        esc_to=$(printf '%s\n' "$to" | sed 's/[\/&]/\\&/g')
        sed -i "s|${esc_from}|${esc_to}|g" "$target"
    fi
}

# ─── 3. Update go.mod ────────────────────────────────────────────────────────
echo "==> Updating go.mod..."
go mod edit -module "$NEW_MODULE"

# ─── 4. Update Go Source Files (*.go) ─────────────────────────────────────────
echo "==> Updating Go import paths in *.go files..."
while IFS= read -r -d '' file; do
    replace_in_file "$file" "$OLD_MODULE" "$NEW_MODULE"
done < <(find . -type f -name '*.go' ! -path '*/.git/*' ! -path '*/bin/*' -print0)

# ─── 5. Update Protobuf Files (*.proto) ───────────────────────────────────────
echo "==> Updating protobuf files in api/..."
while IFS= read -r -d '' file; do
    replace_in_file "$file" "$OLD_MODULE" "$NEW_MODULE"
done < <(find api -type f -name '*.proto' -print0 2>/dev/null || true)

# ─── 6. Update Build, Scripts & Config Files ──────────────────────────────────
echo "==> Updating Makefile, scripts, Docker, and environment templates..."

# Makefile
if [ -f "Makefile" ]; then
    replace_in_file "Makefile" "MODULE      := $OLD_MODULE" "MODULE      := $NEW_MODULE"
    replace_in_file "Makefile" "MODULE ?= $OLD_MODULE" "MODULE ?= $NEW_MODULE"
fi

# Environment files
if [ -f ".env.example" ]; then
    replace_in_file ".env.example" "APP_NAME=$OLD_MODULE" "APP_NAME=$SERVICE_NAME"
    replace_in_file ".env.example" "OTEL_SERVICE_NAME=$OLD_MODULE" "OTEL_SERVICE_NAME=$SERVICE_NAME"
fi

if [ -f ".env" ]; then
    replace_in_file ".env" "APP_NAME=$OLD_MODULE" "APP_NAME=$SERVICE_NAME"
    replace_in_file ".env" "OTEL_SERVICE_NAME=$OLD_MODULE" "OTEL_SERVICE_NAME=$SERVICE_NAME"
fi

# Docker configs
if [ -f "deployments/docker-compose.yml" ]; then
    replace_in_file "deployments/docker-compose.yml" "$OLD_MODULE" "$SERVICE_NAME"
fi

# README.md title & clone examples
if [ -f "README.md" ]; then
    replace_in_file "README.md" "$OLD_MODULE" "$NEW_MODULE"
fi

# ─── 7. Regenerate Code & Verify Dependencies ────────────────────────────────
echo ""
echo "==> Regenerating protobuf stubs (make protogen)..."
if [ -f "./scripts/protogen.sh" ]; then
    ./scripts/protogen.sh
fi

echo "==> Regenerating Wire Dependency Injection (make wire)..."
if command -v wire >/dev/null 2>&1; then
    wire ./bootstrap/...
else
    echo "    Wire CLI not installed in PATH; skipping wire step. Run 'make wire' when ready."
fi

echo "==> Tidying Go module dependencies (go mod tidy)..."
go mod tidy

echo ""
echo "================================================================================"
echo "  ✅ Successfully renamed module to: $NEW_MODULE"
echo "     Service Name: $SERVICE_NAME"
echo "================================================================================"
echo ""
echo "Next steps:"
echo "  1. Verify changes: git diff"
echo "  2. Run tests:      make test"
echo "  3. Build binary:   make build"
echo ""
