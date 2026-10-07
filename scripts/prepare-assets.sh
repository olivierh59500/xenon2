#!/bin/sh
# Restore local import inputs from a user-owned original disk image.
set -eu
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"
GOWORK=off go run ./cmd/import-assets "$@"
