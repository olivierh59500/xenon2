#!/bin/sh
# Restore local import inputs from a user-owned original disk image.
set -eu
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"
./tools/exclude-local-assets.sh
import_only=0
analysis_directory=.local/imported
runtime_directory=${XENON2_RUNTIME_OUTPUT:-assets/runtime}
expect_analysis=0
for argument in "$@"; do
    if [ "$expect_analysis" -eq 1 ]; then
        analysis_directory=$argument
        expect_analysis=0
        continue
    fi
    case "$argument" in
        -dry-run|-dry-run=true|-verify|-verify=true) import_only=1 ;;
        -analysis) expect_analysis=1 ;;
        -analysis=*) analysis_directory=${argument#-analysis=} ;;
    esac
done
GOWORK=off go run ./cmd/import-assets "$@"
if [ "$import_only" -eq 0 ]; then
    GOWORK=off go run ./cmd/export-assets -analysis "$analysis_directory" -output "$runtime_directory"
    GOWORK=off go run ./cmd/export-audio -analysis "$analysis_directory" -output "$runtime_directory/audio"
    GOWORK=off go run ./cmd/export-shop -analysis "$analysis_directory" -output "$runtime_directory"
    GOWORK=off go run ./cmd/export-presentation -analysis "$analysis_directory" -output "$runtime_directory"
    GOWORK=off go run ./cmd/export-shop-audio -analysis "$analysis_directory" -output "$runtime_directory/shop-audio"
fi
