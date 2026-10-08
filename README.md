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

After sixty seconds without input on the title menu or passive logo/credits/scores
presentation, the expert controller starts
an ordinary single-player demo at level one with cheats disabled. DEMO MODE and
`-demo` select the same controller. Keyboard, mouse movement/buttons/wheel, held
controls and touch restart the idle interval and immediately return control of
an active demo; that same action is passed to the game. Fades, pause, the cheat
menu, interactive game messages and merchants do not count as idle time. The controller remains in
development; its complete five-level route is not yet validated.

The controller completes the first three stages from the full default intro,
visits their real merchants and enters level four with one ship and both
continues. Level one loses no ships; level two loses two. The third stage keeps
its admission ship, crosses both cannon checkpoints with 35 shield,
reaches its final guardian with 35 shield and defeats it without losing shield.
Normal rewards and repair restore 39 shield before level four.

In level four the retained native corner commands clear the known forest forks.
The controller rejects forecasts that already end in an unavoidable published
enemy contact, and its exact guard now continues during the middle guardian.
A corrected screen-clear callback exposed a fourth-opening control trap. A
verified maneuver-retention controller now completes the actual intro and
three-stage route, reaches that fight with 27 shield and destroys the tail
without losing shield. It rechecks each remaining maneuver against the live
world before using ordinary controls. The forest minimum remains 11 before a
genuine health pickup; this is not a near-lossless route.
The current carried Pixel route also destroys both upper satellites through Rear
shots, the lower-right satellite through Cannon shots and the lower-left satellite
through primary shots. All 27 shield, the same ship and both continues remain at
the exposed core. It then defeats the core and reaches the real middle merchant
with the same 27 shield. After ordinary repair and rewards, the current desktop
route also reaches the fourth final guardian with 31 shield, defeats it with
27 shield and enters level five after its final merchant and normal repair.
The earned fifth-stage controller and near-lossless campaign remain unfinished.
Beating the remaining guardians and validating the whole campaign remain active
work.
Cached routes, bonus collection, dive requests and merchant purchases use
ordinary game rules. This is not yet a near-lossless five-level demonstration.

## MP4 recording

```sh
GOWORK=off go run ./cmd/video -duration 3m -output recordings/xenon2-presentation.mp4
GOWORK=off go run ./cmd/video -complete-level1 -output recordings/xenon2-level1-presentation.mp4
```

The recorder captures the original game canvas at 1280 × 800 and 60 FPS, with
the game's own stereo soundtrack and effects. It uses DCK v1.0.14's offline
video/audio route and FFmpeg for H.264/AAC encoding; other windows and system
audio are never captured. The default three-minute presentation follows the
intro, menu and first-level play. Its presentation controller approaches actual
enemies and reachable bonuses, commits to short tactical routes and holds aimed
firing bursts with reassessment pauses. Known terrain junctions guide the route
before a wrong branch closes. A trapped ship can reverse to the junction and
rejoin its forward route. Targeting forecasts the source paths and the next gun
position; reward collection follows the moving coin rather than its old anchor.
Upcoming formations can also guide preparatory movement, without firing early.

The complete-first-level option includes the intro, both merchants, final guardian
and exit drops, then stops before playing level two. The latest recording is
381.53 seconds (6 minutes 21 seconds), with all three starting ships remaining.
Its MP4 contains 22,892 frames at 1280 × 800 and 60 FPS, with stereo AAC audio
and embedded scene chapters. The current desktop graphics tests also pass.
This capture was generated from runtime `0a9efa0`.
Ordinary damage and purchases still apply. The current controller's first-three-stage
validation uses the real intro, merchants, native guardian damage and exit drops;
its third-final fight retains all 35 admission shield points. The broader expert
route still needs the fourth/fifth strategies and mobile frame-pacing work. See
[expert forecast design](docs/EXPERT_FORECAST.md) for the verified boundaries.
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
