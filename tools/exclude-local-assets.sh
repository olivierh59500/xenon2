#!/bin/sh
# Install local exclusions before importing original or generated resources.
set -eu
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"
exclude=$(git rev-parse --git-path info/exclude)
for pattern in '/previous/' '/captures/' '/new/' '/.local/' '/assets/original/' '/assets/runtime/*' '!/assets/runtime/GENERATED.txt' '/assets/imported/' '/bin/' '/dist/' '/.gitignore' '*.adf' '*.dms' '*.rom'; do
    if ! grep -Fqx "$pattern" "$exclude"; then
        printf '%s\n' "$pattern" >> "$exclude"
    fi
done
printf '%s\n' 'Original references, recovered programs and generated assets are excluded locally.'
