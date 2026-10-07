# Xenon 2 Go

An independent Go/Ebitengine recreation of Xenon 2: Megablast for the Amiga.
The original disk is used only by local extraction and comparison tools.
The game engine uses ordinary Go state and exported audiovisual data;
it does not execute the original program or load its code as game logic.

The conversion is in development. The current reference build displays the
original menu and five level maps, moves the ship and enemy formations, and
replays the original soundtrack. It includes source-derived weapons,
carrier rewards, checkpoint recovery, alternating players, and both sections
of the five levels, including their compound guardians. Original shop, attract,
HUD, loading and ending presentation are connected. All fixed encounter families
are implemented. Shared-state and artwork audits and full-game validation remain
in progress; this build is not yet the complete playable game.

## Demonstration pilot

```sh
GOWORK=off go run ./cmd/xenon2 -demo
```

The menu also offers DEMO MODE. This starts an ordinary single-player session at level one
with trainer aids disabled. The development pilot uses ordinary movement, fire
and dive commands. It enters
through the normal menu and READY, handles score entry and continues, and buys
available upgrades through the original merchant quote/confirmation interface.
It chooses the cheapest sufficient shield repair, saves for extra ships when
the current merchant stocks them, and avoids repeatedly upgrading rear weapons
at the expense of the basic loadout. Priorities work across both shop pages.
A key or click immediately returns control of the current game to the player.
It does not grant health, equipment or money, skip guardians or change terrain.

Full-game autonomous playback is still in development. The current pilot
completes levels one and two from the normal menu, including guardian destruction
and both merchants on each stage, and starts level three. The full-intro startup
also traverses all six credit pairs and wins both stages, carrying one ship into
stage three; the direct-menu regression now carries one after finite explosions
were corrected. Its carried third-stage route still loses its ships before the
middle guardian, including after a legal continue. Cached terrain routes, ordinary bonus
collection, dive requests and shop purchases retain normal game rules. A
separate second-stage opening reaches its first checkpoint with all three ships.
The remaining stages and full campaign still need validated pilot strategies;
this is not yet a completed five-level demonstration.

## MP4 recording

```sh
GOWORK=off go run ./cmd/video -duration 3m -output recordings/xenon2-presentation.mp4
```

The recorder captures the original game canvas at 1280 × 800 and 60 FPS, with
the game's own stereo soundtrack and effects. It uses DCK v1.0.14's offline
video/audio route and FFmpeg for H.264/AAC encoding; other windows and system
audio are never captured. The default three-minute presentation follows the
intro, menu and first-level play. Its presentation controller approaches actual
enemies and reachable bonuses, commits to short tactical routes and holds aimed
firing bursts with reassessment pauses. It retains ordinary damage and can lose
a ship; complete-stage and campaign success remain unproven for this controller.
Generated MP4, PNG poster and chapter JSON files stay
under locally excluded `recordings/`. The export command requires Go 1.26 or
newer and FFmpeg. A different positive `-duration` records a longer excerpt.

## Android

```sh
./scripts/run-android.sh
```

The landscape version keeps the original game canvas and adds a virtual joystick,
FIRE/DIVE buttons and ENTER/MENU/PAUSE/CHEATS controls in the side margins.
The joystick stays captured while dragging, supports diagonals and works with
simultaneous fire and dive touches. Menus and merchant cells also accept direct
taps. The activity keeps the screen awake while the game is visible.

The script builds an ARM64 test APK, installs it on the authorized USB device
and starts the app. See [Android build and controls](docs/ANDROID.md) for SDK,
device and validation details. Generated APK/AAR/Gradle outputs remain excluded.

## Optional trainer settings

Select CHEATS on the menu, use `-cheats` at launch, or press F3 during a game. The submenu provides
infinite ships, continues, money and energy, optional equipment keys, and a
starting level from one to five. All aids are disabled by default; active aids
are marked on the playfield. F3 or Escape returns to the caller.

The settings reproduce the supplied Amiga trainer's boundaries: infinite energy
suppresses shield damage but does not prevent terrain crushing. Infinite money
grants 5,000,000 at merchant entry with a 30,000 stock limit; purchases still cost
the original prices and retain equipment compatibility. Infinite ships retain
death/checkpoint recovery and alternating player turns.

Equipment keys work only when KEY FUNCTIONS is ON. The KEY HELP page lists the
adapted desktop controls. The shortcut layout reserves F3 for settings and uses F1/F2/F4–F10 for common
upgrades, numeric keypad 0–9 for weapons/dive, B/H/O/V for bomb, homing missile,
protection and Shades, and Delete/Insert for energy.
These aids are optional player choices; ordinary demo validation keeps them off.

## Local resources

Supply a compatible original Amiga ADF in a local directory. Original disks,
compressed containers, recovered programs and generated assets are not included
in Git. Extraction tools rebuild the local resources reproducibly.

```sh
./scripts/prepare-assets.sh -adf "/path/to/Xenon 2.adf"
GOWORK=off go run ./cmd/xenon2
```

Use the arrow keys or WASD for movement and Space or Control for firing; Alt
requests a dive. In reference views only, keys 1–5 select a level
and F2 opens the shop inspection route. Escape restarts the attract sequence;
from the menu it closes the desktop window. M toggles
music. P pauses gameplay; any key or a fire click resumes it while the soundtrack
continues. The normal menu supports one or two alternating players. Full source
timing and framebuffer comparisons remain in
progress for integrated scenes and the complete five-level run.

```sh
GOWORK=off go run ./cmd/xenon2 -view level -level 1
GOWORK=off go run ./cmd/xenon2 -view attract
GOWORK=off go run ./cmd/xenon2 -view menu -frames 120 -screenshot .local/menu.png
GOWORK=off go test ./...
```

`./scripts/check-desktop.sh` runs the renderer tests and bounded menu, attract,
shop and five-level captures. It needs an active desktop session and rebuilt
local assets. Captures are saved under `.local/captures/desktop-check/`.

Rendering runs at 60 display updates per second. Ship motion, paths, firing and
scrolling use a separate simulation clock; interpolation does not speed up the
game. Audio follows its own original 50 Hz replay clock.

The first native A500 gameplay capture scrolls about 16.7 source pixels per
second, corresponding to three PAL refreshes per pass. This measured early-game
profile is now the default. Use `-logic-pal-refreshes 2` to select the source's
two-refresh maximum (25 passes/s) for comparison. Dense-scene and later-level
cadence still need integrated native measurements.

The intro's expensive credit zoom uses the recorded A500 average: 15.5 passes/s,
giving six seconds per 93-pass credit pair. This is a stable average adaptation
of the observed workload-dependent cadence. Menu, shop, audio and palette fades
retain their separate clocks; changing gameplay timing does not accelerate them.

See [implementation coverage](docs/IMPLEMENTATION_COVERAGE.md),
[reference notes](docs/REFERENCE_NOTES.md) and the
[native-data audit](docs/NATIVE_AUDIT.md) for verified findings and comparison
boundaries. The supplied disk is a trained revision; trainer behavior is kept
separate from the original game's rules.

The [asset setup guide](docs/ASSET_SETUP.md) describes the reproducible extraction
pipeline, and the [audio notes](docs/AUDIO.md) document the independent replay.

To build a standalone desktop executable after extraction:

```sh
mkdir -p bin
GOWORK=off go build -o bin/xenon2 ./cmd/xenon2
./bin/xenon2
```

The exported assets are embedded at build time. A clean checkout can compile
the tools and game before extraction, but playing requires the local resources.

Generated references and captures remain local. The original shop's static
bitmaps and portrait poses, all sixteen caption/logo sampling steps, the HUD and
starfield state have source comparisons. The first final guardian controller
matches 1,200 original passes, and its articulated chain matches 9,280 segment
states. These checks do not establish the complete five-level game.
