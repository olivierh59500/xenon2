#!/bin/sh
# Run frontend logic on one explicitly selected ARM64 Android device.
# This executable does not start an Android view, draw or read GPU pixels.
set -eu

usage() {
    echo "Usage: $0 [--build-only] [--run TEST_REGEXP]" >&2
    echo "Device execution requires ANDROID_SERIAL; the default selects the real title-idle admission test." >&2
    echo "XENON2_TEST_WATCHDOG_MS optionally selects a runner deadline from 1 to 470000 milliseconds (default 470000)." >&2
}

build_only=false
test_pattern='^TestTitleIdleStartsExpertDemoAtSixtySeconds$'
while [ "$#" -gt 0 ]; do
    case "$1" in
        --build-only) build_only=true; shift ;;
        --run)
            if [ "$#" -lt 2 ] || [ -z "$2" ]; then
                usage
                exit 2
            fi
            test_pattern=$2
            shift 2
            ;;
        --help|-h) usage; exit 0 ;;
        *) usage; exit 2 ;;
    esac
done

watchdog_ms=${XENON2_TEST_WATCHDOG_MS:-470000}
case "$watchdog_ms" in
    ''|*[!0-9]*) echo "XENON2_TEST_WATCHDOG_MS must be an integer from 1 to 470000." >&2; exit 2 ;;
esac
if [ "${#watchdog_ms}" -gt 6 ] || [ "$watchdog_ms" -lt 1 ] || [ "$watchdog_ms" -gt 470000 ]; then
    echo "XENON2_TEST_WATCHDOG_MS must be an integer from 1 to 470000." >&2
    exit 2
fi

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
android_sdk=${ANDROID_HOME:-${ANDROID_SDK_ROOT:-}}
if [ -z "$android_sdk" ]; then
    for candidate in /opt/homebrew/share/android-commandlinetools "$HOME/Library/Android/sdk"; do
        if [ -d "$candidate/ndk/28.2.13676358" ]; then
            android_sdk=$candidate
            break
        fi
    done
fi
android_ndk=${ANDROID_NDK_HOME:-$android_sdk/ndk/28.2.13676358}
case "$(uname -s)" in
    Darwin) ndk_host=darwin-x86_64 ;;
    Linux) ndk_host=linux-x86_64 ;;
    *) echo "This runner requires an existing macOS or Linux Android NDK." >&2; exit 1 ;;
esac
ndk_bin="$android_ndk/toolchains/llvm/prebuilt/$ndk_host/bin"
if [ ! -x "$ndk_bin/aarch64-linux-android23-clang" ] || [ ! -x "$ndk_bin/aarch64-linux-android23-clang++" ]; then
    echo "Existing ARM64/API23 NDK compilers were not found. Set ANDROID_HOME or ANDROID_NDK_HOME." >&2
    exit 1
fi
if ! "$build_only" && [ -z "${ANDROID_SERIAL:-}" ]; then
    echo "Set ANDROID_SERIAL to the exact authorized device; this script never selects a device automatically." >&2
    exit 1
fi

export GOWORK=off
export GOTOOLCHAIN=local
export GOPROXY=off
export GOSUMDB=off
export GOCACHE="${GOCACHE:-$project_root/.local/go-cache}"
test_root="$project_root/.local/android-frontend-logic"
mkdir -p "$test_root" "$GOCACHE"
test_binary="$test_root/app-logic.test"
cd "$project_root"
echo "Building Android ARM64 frontend logic tests (API23, PIE)..."
CGO_ENABLED=1 GOOS=android GOARCH=arm64 \
    CC="$ndk_bin/aarch64-linux-android23-clang" \
    CXX="$ndk_bin/aarch64-linux-android23-clang++" \
    go test -c -buildmode=pie -tags=xenon2_logic_only -o "$test_binary" ./internal/app
echo "Test executable: $test_binary"
if "$build_only"; then
    exit 0
fi

adb_path=${ADB:-$android_sdk/platform-tools/adb}
if [ ! -x "$adb_path" ] && [ -z "${ADB:-}" ]; then
    adb_path="$HOME/Library/Android/sdk/platform-tools/adb"
fi
if [ ! -x "$adb_path" ]; then
    echo "ADB was not found. Set ADB to an existing platform-tools/adb executable." >&2
    exit 1
fi
if [ "$("$adb_path" -s "$ANDROID_SERIAL" get-state)" != device ]; then
    echo "The explicitly selected Android device is not connected and authorized." >&2
    exit 1
fi
if [ "$("$adb_path" -s "$ANDROID_SERIAL" shell getprop ro.product.cpu.abi | tr -d '\r')" != arm64-v8a ]; then
    echo "The selected Android device must use arm64-v8a." >&2
    exit 1
fi

runtime_source=${XENON2_RUNTIME_TEST_DIR:-$project_root/assets/runtime}
if [ ! -d "$runtime_source" ]; then
    echo "Original runtime exports were not found: $runtime_source" >&2
    exit 1
fi
stage_root=$(mktemp -d "$test_root/stage.XXXXXX")
trap 'rm -rf "$stage_root"' 0
mkdir "$stage_root/runtime"
cp -R "$runtime_source/." "$stage_root/runtime/"
rm -f "$stage_root/runtime/embed.go" "$stage_root/runtime/GENERATED.txt"
asset_count=$(find "$stage_root/runtime" -type f | wc -l | tr -d '[:space:]')
if [ "$asset_count" -ne 169 ]; then
    echo "Expected 169 original runtime resource files; found $asset_count. Check the local exports." >&2
    exit 1
fi

# The remote shell receives only a base64 alphabet, never the regex's pipes,
# parentheses, quotes or spaces. Android TestMain decodes and sets test.run.
test_pattern_base64=$(printf '%s' "$test_pattern" | base64 | tr -d '\r\n')
remote_root=/data/local/tmp/xenon2-frontend-logic
test_log="$test_root/test.log"
test_status="$test_root/status.txt"
rm -f "$test_log" "$test_status"
"$adb_path" -s "$ANDROID_SERIAL" shell \
    "mkdir -p '$remote_root' && rm -rf '$remote_root/runtime' && rm -f '$remote_root/test.log' '$remote_root/status.txt'"
"$adb_path" -s "$ANDROID_SERIAL" push "$test_binary" "$remote_root/app-logic.test" >/dev/null
"$adb_path" -s "$ANDROID_SERIAL" push "$stage_root/runtime" "$remote_root/" >/dev/null
"$adb_path" -s "$ANDROID_SERIAL" shell "chmod 700 '$remote_root/app-logic.test'"
echo "Running selected logic tests on $ANDROID_SERIAL with a ${watchdog_ms}ms runner watchdog and an eight-minute Go timeout..."
remote_exit=0
"$adb_path" -s "$ANDROID_SERIAL" shell \
    "exec env XENON2_RUNTIME_TEST_DIR='$remote_root/runtime' XENON2_TEST_OUTPUT='$remote_root/test.log' XENON2_TEST_STATUS='$remote_root/status.txt' XENON2_TEST_PATTERN_BASE64='$test_pattern_base64' XENON2_TEST_WATCHDOG_MS='$watchdog_ms' XENON2_HUMAN_PRESENTATION_CHECK=1 XENON2_DEMO_PROGRESS_CHECK=1 '$remote_root/app-logic.test' -test.v -test.timeout=8m" \
    || remote_exit=$?
"$adb_path" -s "$ANDROID_SERIAL" pull "$remote_root/test.log" "$test_log" >/dev/null || { if [ "$remote_exit" -eq 0 ]; then remote_exit=1; fi; }
"$adb_path" -s "$ANDROID_SERIAL" pull "$remote_root/status.txt" "$test_status" >/dev/null || { if [ "$remote_exit" -eq 0 ]; then remote_exit=1; fi; }
if [ -f "$test_log" ]; then
    cat "$test_log"
fi
echo "Local logic-test output: $test_log"
if [ ! -f "$test_status" ]; then
    echo "Android logic tests produced no completion status (remote exit $remote_exit). The runner did not complete; inspect $test_log for a timeout or process termination." >&2
    if [ "$remote_exit" -eq 0 ]; then remote_exit=1; fi
    exit "$remote_exit"
fi
reported_status=$(cat "$test_status")
if [ "$reported_status" = exit=124 ]; then
    echo "Android logic test runner timed out; this is not a gameplay assertion failure. Goroutine stacks are in $test_log." >&2
    exit 124
fi
if [ "$remote_exit" -ne 0 ]; then
    echo "Android logic test runner exited with status $remote_exit (completion status: $reported_status)." >&2
    exit "$remote_exit"
fi
if [ "$reported_status" != exit=0 ]; then
    echo "The Android test runner did not report successful completion." >&2
    exit 1
fi
if grep -Fq 'warning: no tests to run' "$test_log"; then
    echo "The supplied pattern selected no tests." >&2
    exit 1
fi
