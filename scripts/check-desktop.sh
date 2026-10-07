#!/bin/sh
# Run bounded Ebitengine scenes and retain captures outside version control.
set -eu
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"
./tools/exclude-local-assets.sh
if [ "$(uname -s)" = Darwin ] && /usr/sbin/ioreg -n Root -d1 | grep -q '"IOConsoleLocked" = Yes'; then
    printf '%s\n' 'Unlock the macOS session before running desktop scene checks.' >&2
    exit 1
fi
capture_directory=.local/captures/desktop-check
mkdir -p "$capture_directory" .local/bin
XENON2_RUNTIME_TEST_DIR="$project_root/assets/runtime" \
XENON2_RENDER_CAPTURE_DIR="$project_root/.local/captures/integrated-render" \
GOWORK=off go test ./internal/app
GOWORK=off go build -o .local/bin/xenon2 ./cmd/xenon2
for view in menu attract shop; do
    .local/bin/xenon2 -view "$view" -mute -frames 360 -screenshot "$capture_directory/$view.png"
done
for level in 1 2 3 4 5; do
    .local/bin/xenon2 -view level -level "$level" -mute -frames 240 -screenshot "$capture_directory/level-$level.png"
done
printf 'Desktop scenes captured in %s\n' "$capture_directory"
