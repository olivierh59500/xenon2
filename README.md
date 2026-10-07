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
and F2 opens the shop inspection route. Escape returns to the menu. M toggles
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
