#!/bin/sh
# Install local exclusions before importing original or generated resources.
set -eu
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"
exclude=$(git rev-parse --git-path info/exclude)
for pattern in '/previous/' '/captures/' '/recordings/' '/new/' '/.local/' '/assets/original/' '/assets/runtime/*' '!/assets/runtime/GENERATED.txt' '!/assets/runtime/embed.go' '/assets/imported/' '/bin/' '/dist/' '/android/.gradle/' '/android/build/' '/android/app/build/' '/android/app/libs/' '/android/local.properties' '/.gitignore' '/*.test' '/*.pprof' '.DS_Store' '*.adf' '*.dms' '*.rom'; do
    if ! grep -Fqx "$pattern" "$exclude"; then
        printf '%s\n' "$pattern" >> "$exclude"
    fi
done
printf '%s\n' 'Original references, recovered programs and generated assets are excluded locally.'
