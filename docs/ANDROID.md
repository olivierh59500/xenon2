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

## Frontend logic tests on a locked device

The Android ARM64 test executable can run the actual frontend and gameplay
logic without starting an Android view. It works while the host or phone is
locked. Select the authorized device explicitly; the runner never chooses a
different connected device:

```sh
ANDROID_SERIAL='your-authorized-device-serial' ./scripts/check-android-logic.sh
```

The default runs `TestTitleIdleStartsExpertDemoAtSixtySeconds`, including its
ordinary menu and READY admission. To select other frontend tests, supply a Go
test regular expression. This example also checks the connected expert route
through the first two stages from the complete intro:

```sh
ANDROID_SERIAL='your-authorized-device-serial' ./scripts/check-android-logic.sh \
  --run '^(TestTitleIdleStartsExpertDemoAtSixtySeconds|TestPresentationPilotCompletesFirstTwoLevelsFromDefaultIntroOptional)$'
```

The longer connected third-stage checkpoint regression is
`TestExpertThirdPostMerchantRouteReachesCannonCheckpointsWithoutShipLossOptional`.
The script enables the optional presentation/progression checks when selected;
each run has an eight-minute wall-clock test timeout. Regexes are base64 encoded
on the host and decoded into Go's `test.run` flag, so their pipes and other
syntax never enter the remote shell command.

Use `--build-only` to compile without any device access. The runner uses the
existing Go toolchain and NDK ARM64/API23 C and C++ compilers, produces a PIE
test executable, and disables module/toolchain downloads. `ANDROID_HOME`,
`ANDROID_SDK_ROOT`, `ANDROID_NDK_HOME` and `ADB` can select existing tools.
The normal runtime exports supply 169 JSON, PNG and PCM resource files, copied
through a temporary local staging directory into the selected device's
`/data/local/tmp/xenon2-frontend-logic/runtime`. `XENON2_RUNTIME_TEST_DIR` can
select a different existing export directory. Test binaries and captured
output stay under `.local/android-frontend-logic`; `test.log` and `status.txt`
preserve the run's output and exit status, including Android GoLog output.

The `android && xenon2_logic_only` test build calls `m.Run` directly. Dedicated
graphics fixtures skip, and optional pixel captures in frontend boundary tests
do nothing so their remaining logic assertions still execute. Normal desktop
tests retain their Ebitengine graphics loop. This runner does not verify Draw,
ReadPixels, device rendering, audio playback or manual touch interaction, and
passing a bounded route test does not establish completion of all five stages.

The 8 October locked-Pixel check passes all title-idle and takeover cases:
automatic expert admission after sixty idle seconds, idle reset for keyboard,
pointer and held touch activity, suspended-screen exclusions, and applying the
same keyboard or touch action when manual control takes over. The installed
debug build uses verified runtime commit `8e10b88`; signature, 16 KiB alignment,
installation, cold launch and fatal-log checks also pass. Later documentation
and equipment-fixture corrections do not change that runtime. Neither check
establishes rendered performance or a complete five-stage expert route.
