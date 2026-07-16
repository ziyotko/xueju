#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
mkdir -p "$ROOT/.runtime/gocache"

export GOCACHE=${GOCACHE:-"$ROOT/.runtime/gocache"}
export APP_PORT=${APP_PORT:-8080}

cd "$ROOT/backend"
exec go run ./cmd/api
