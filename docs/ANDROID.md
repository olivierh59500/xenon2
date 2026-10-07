# Android development build

The Android application uses the same Go game, original canvas, sound stream and
embedded local exports as the desktop program. The Java activity only handles
the Android view, landscape orientation, immersive mode and lifecycle. Artwork,
audio, generated APKs/AARs and local device captures remain outside Git.

## Build and install

Prepare the user-owned runtime exports as described in [ASSET_SETUP.md](ASSET_SETUP.md)
before building. Connect and authorize one ARM64 Android device, then run:

```sh
./scripts/run-android.sh
```

To produce the debug APK without using ADB:

```sh
./scripts/run-android.sh --build-only
```

The output is `android/app/build/outputs/apk/debug/app-debug.apk`, signed with the
Android development key. The application ID is `com.olivierh.xenon2`.
`ANDROID_SERIAL` selects a device when several are connected, and `ADB` can
select an existing platform-tools executable separately from the build SDK.

The script requires existing Go 1.26 or later, Java 17, SDK platform 36, Build
Tools 36.0.0 and NDK 28.2.13676358. It detects the validated Homebrew SDK and JDK
on this machine; `ANDROID_HOME` and `JAVA_HOME` override detection. It does not
install development tools. The checked-in wrapper pins Gradle 8.11.1 and AGP
8.10.1; the Go mobile binding pins the game's Ebitengine 2.9.11. The minimum
Android API is 23 and the debug APK contains only `arm64-v8a` native code.

The wrapper follows the [official Ebitengine mobile integration](https://ebitengine.org/en/documents/mobile.html),
including `Seq.setContext` for 2.9 and suspension/resumption of the view. The
Gradle/JDK/API combination follows the [AGP 8.10 compatibility table](https://developer.android.com/build/releases/agp-8-10-0-release-notes).

## Touch controls

The original 320 × 200 canvas is scaled uniformly and centered. The side panels
contain a retained eight-way virtual joystick, FIRE and DIVE, with independent
finger capture. The joystick's thumb follows the contact, including drags beyond
its base; releasing it returns to neutral without adopting another held finger.

FIRE is held for shooting and also confirms menus, READY and purchases. DIVE is
a one-press action and needs the ordinary dive equipment. ENTER confirms without
firing. MENU uses the original Escape return to the presentation loop; pressing
it from the title returns to the intro. PAUSE pauses play, and a new contact
resumes it. CHEATS opens the optional trainer, which also accepts joystick
navigation, FIRE/ENTER and direct taps. Original menus and shop cells accept
taps translated to the original canvas coordinates. The controls change color
while held.

The ordinary demonstration pilot remains reachable from the title menu.
Any new touch takes manual control. The Android host starts in the original
presentation sequence and opens audio during its first Update, after Android
has installed the view and context.

## Verification

```sh
GOWORK=off go test ./internal/controls
GOWORK=off go vet ./internal/controls ./mobile
./scripts/check-android-touch.sh
./android/gradlew -p android --console=plain :app:lintDebug
```

After installation, inspect the process and application logs with the selected
ADB executable. An authorized but locked phone supports install and process
checks; rendering and manual multitouch need an unlocked device. Test joystick
diagonals together with fire/dive, menu navigation, pause/resume, two landscape
rotations, and suspend/resume before treating the device interaction as fully
verified.

On 7 October 2026, the debug build, `lintDebug`, signature verification and
16 KiB APK alignment checks passed. A USB-authorized Pixel 10a (`stallion`),
running Android 17 with `arm64-v8a`, accepted installation and cold launch.
The process remained alive, the native library loaded, and the application logs
reported no game crash. The phone was locked, so device graphics, audio playback
and manual multitouch were not verified in that check. The joystick/control
geometry and multitouch tests, including the race check, passed separately.
The native touch adapter also releases all contacts on cancellation, suspension
and focus loss, covering a retained-touch issue in the pinned Ebitengine 2.9
bridge. A regression check exercises three fingers and duplicate lifecycle
callbacks using the existing JDK.
