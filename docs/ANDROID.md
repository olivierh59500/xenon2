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
Sixty seconds without input in that passive logo, credits and scores loop also
start the expert demo. The interval spans its phase changes; touching a control
restarts the interval or immediately takes over an active demo. Interactive
messages, READY, shops, fades and pause do not count toward that deadline.

## Verification

The narrow-barrier update from runtime `c6559a3` is installed on the Pixel 10a.
Its three original-resource barrier regressions pass on the device, including
the impulse/release alignment that fails with the former code. The APK passes
signature and 16 KiB alignment checks; its installed bytes match the verified
build. It was installed without opening a view or interrupting another app.
The complete engine suite passes in 28.320 seconds and the unchanged desktop
four-stage regression in 14.45 seconds.

The preceding 8 October build from runtime `584b90c` completed the default-intro
Pixel logic replay through all four stages, visited their
real merchants and entered stage five with 39 shield, one ship and two continue
credits in 206.87 seconds. The ARM64 debug APK passes signature and 16 KiB
alignment checks. Installation does not establish visual smoothness, and the
earned fifth-stage route remains unfinished.

The 8 October build from runtime `cbb3860` passes the complete-intro locked-device
logic check through all four fourth-middle satellites in 38.26 seconds. It
retains 27 shield, one ship and two continue credits at frame 3983; the core
still has 175 health. The ARM64 debug APK passes signature and 16 KiB alignment
checks and is installed on the Pixel 10a. Cold launch takes 1,475 ms and its
process remains active. This installation check does not verify device graphics
or smoothness, and the remaining boss/campaign controller is unfinished.

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
each run has an eight-minute Go timeout. An independent 7m50s watchdog writes
all goroutine stacks, syncs the output and returns `exit=124` before that limit.
`XENON2_TEST_WATCHDOG_MS` can select 1–470000 milliseconds for a shorter diagnostic.
A runner timeout or missing completion status is not a gameplay assertion failure.
Regexes are base64 encoded
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
debug build now uses runtime commit `0a9efa0`, including the validated early-stage
controller, third-final preparation, complete fourth-fork command sequence and
native fourth-middle contact/terrain guard. Its APK passes
signature and 16 KiB alignment checks, installs on the authorized Pixel 10a and
cold-launches in 952 ms. A full default-intro device logic replay preserves all
strict first-three-stage reserves and reaches the native fourth-middle tail kill
with 23 shield, the same ship and both continues in 175.55 seconds. The forest
minimum is still 3 shield before a genuine health pickup. This is a startup/logic
check, not visual verification, dense-combat frame
pacing or a complete five-stage expert route.
