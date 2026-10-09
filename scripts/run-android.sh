#!/bin/sh
# Build a debug ARM64 APK with the same Ebitengine version as the Go game.
set -eu

usage() {
    echo "Usage: $0 [--build-only]" >&2
    echo "Build Xenon2 for Android, then install and launch on one authorized USB device." >&2
}

build_only=false
for option in "$@"; do
    case "$option" in
        --build-only) build_only=true ;;
        --help|-h) usage; exit 0 ;;
        *) usage; exit 2 ;;
    esac
done

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
android_sdk=${ANDROID_HOME:-${ANDROID_SDK_ROOT:-}}
java_home_path=${JAVA_HOME:-}
adb_path=${ADB:-}

if [ -z "$android_sdk" ]; then
    for candidate in /opt/homebrew/share/android-commandlinetools "$HOME/Library/Android/sdk"; do
        if [ -f "$candidate/platforms/android-36/android.jar" ] && [ -d "$candidate/ndk/28.2.13676358" ]; then
            android_sdk=$candidate
            break
        fi
    done
fi
if [ -z "$java_home_path" ] && [ -x /opt/homebrew/opt/openjdk@17/bin/java ]; then
    java_home_path=/opt/homebrew/opt/openjdk@17
fi
if [ -z "$java_home_path" ] && [ -x "/Applications/Android Studio.app/Contents/jbr/Contents/Home/bin/java" ]; then
    java_home_path="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
fi
if [ -z "$adb_path" ]; then
    if [ -x "$HOME/Library/Android/sdk/platform-tools/adb" ]; then
        adb_path="$HOME/Library/Android/sdk/platform-tools/adb"
    else
        adb_path="$android_sdk/platform-tools/adb"
    fi
fi

if [ -z "$android_sdk" ] || [ ! -f "$android_sdk/platforms/android-36/android.jar" ] || [ ! -d "$android_sdk/ndk/28.2.13676358" ]; then
    echo "Android SDK 36 and NDK 28.2.13676358 were not found. Set ANDROID_HOME to an existing installation." >&2
    exit 1
fi
if [ ! -x "$android_sdk/build-tools/36.0.0/aapt" ]; then
    echo "Android Build Tools 36.0.0 were not found in the selected SDK." >&2
    exit 1
fi
if [ -z "$java_home_path" ] || [ ! -x "$java_home_path/bin/java" ]; then
    echo "Java 17 was not found. Set JAVA_HOME to an existing JDK." >&2
    exit 1
fi
if ! "$java_home_path/bin/java" -version 2>&1 | head -1 | grep -q 'version "17\.'; then
    echo "The Android build requires Java 17; check JAVA_HOME." >&2
    exit 1
fi
if [ ! -x "$project_root/android/gradlew" ]; then
    echo "The Gradle wrapper is missing from android/." >&2
    exit 1
fi
if ! "$build_only" && [ ! -x "$adb_path" ]; then
    echo "ADB was not found. Set ADB to an existing platform-tools/adb executable." >&2
    exit 1
fi

export ANDROID_HOME="$android_sdk"
export ANDROID_SDK_ROOT="$android_sdk"
export JAVA_HOME="$java_home_path"
export PATH="$java_home_path/bin:$android_sdk/platform-tools:$PATH"
export GOWORK=off
export GOCACHE="${GOCACHE:-$project_root/.local/go-cache}"

cd "$project_root"
./tools/exclude-local-assets.sh
mkdir -p android/app/libs "$GOCACHE"
ebiten_version=$(go list -m -f '{{.Version}}' github.com/hajimehoshi/ebiten/v2)
if [ "$ebiten_version" != "v2.9.11" ]; then
    echo "This Android wrapper expects Ebitengine v2.9.11; found $ebiten_version." >&2
    exit 1
fi
echo "Checking the Android module graph..."
GOOS=android GOARCH=arm64 go list -m -tags=android all >/dev/null
echo "Building the Go/Ebitengine ARM64 library..."
run_ebitenmobile() {
    if [ -n "${EBITENMOBILE_BIN:-}" ]; then
        if [ ! -x "$EBITENMOBILE_BIN" ]; then
            echo "EBITENMOBILE_BIN is not executable." >&2
            exit 1
        fi
        tool_version=$(go version -m "$EBITENMOBILE_BIN" | awk '$1 == "mod" && $2 == "github.com/hajimehoshi/ebiten/v2" { print $3 }')
        if [ "$tool_version" != "$ebiten_version" ]; then
            echo "The supplied ebitenmobile must match $ebiten_version; found $tool_version." >&2
            exit 1
        fi
        "$EBITENMOBILE_BIN" "$@"
    else
        go run github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@"$ebiten_version" "$@"
    fi
}
run_ebitenmobile \
    bind -target android/arm64 -androidapi 23 \
    -javapkg com.olivierh.xenon2 \
    -o android/app/libs/libxenon2.aar ./mobile

echo "Building the debug APK..."
"$project_root/android/gradlew" -p "$project_root/android" --console=plain :app:assembleDebug
apk_path="$project_root/android/app/build/outputs/apk/debug/app-debug.apk"
echo "APK: $apk_path"
if "$build_only"; then
    exit 0
fi

if [ -z "${ANDROID_SERIAL:-}" ]; then
    device_count=$("$adb_path" devices | awk 'NR > 1 && $2 == "device" { count++ } END { print count + 0 }')
    if [ "$device_count" -ne 1 ]; then
        echo "Exactly one authorized Android device is required; found $device_count. Set ANDROID_SERIAL to select one." >&2
        "$adb_path" devices -l >&2
        exit 1
    fi
    ANDROID_SERIAL=$("$adb_path" devices | awk 'NR > 1 && $2 == "device" { print $1; exit }')
    export ANDROID_SERIAL
fi
if [ "$("$adb_path" -s "$ANDROID_SERIAL" get-state)" != device ]; then
    echo "The selected Android device is not authorized or connected." >&2
    exit 1
fi

echo "Installing Xenon 2 Go..."
"$adb_path" -s "$ANDROID_SERIAL" install -r "$apk_path"
echo "Launching Xenon 2 Go..."
"$adb_path" -s "$ANDROID_SERIAL" shell am start -S -W -n com.olivierh.xenon2/.MainActivity
