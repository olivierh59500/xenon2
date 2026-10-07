# Native game audit

The original disk is inspected offline to recover the artwork, game data and exact behavior. The finished game uses an independent Go engine. Original 68000 code, CPU state, emulators and executable loaders are not part of its runtime resources.

## Disk and executable

The supplied 901,120-byte disk is a standard Amiga OFS filesystem. Its main executable, `XenonII`, is a 200,128-byte Amiga HUNK file with a backwards ByteKiller loader. Decompression produces a 262,440-byte image with a zero XOR checksum. The final 296 bytes contain modifications added by the supplied disk release; these modifications are distinguished from the game itself during analysis.

The game image initially loads at address `0x30000`. The disk loader copies its first 256 KiB to address zero and enters the original game at address `0x400`. Consequently, the absolute references inside the game use the latter address space. This matters when identifying artwork, tables and routines.

## Level payloads

Six compressed payloads are present: five gameplay levels and a separate shop resource. Their names encode the original starting disk block and uncompressed block count. They have a 12-byte header followed by a backwards compressed stream. The header provides four decoding parameters, the decompressed length and the compressed length. The decoder combines literal runs, repeated bytes and backwards dictionary matches.

Two independently implemented decoders recover the same output and consume every stream exactly to its header boundary:

| Disk resource | Decoded size |
| --- | ---: |
| `000B00E5` | 117,248 bytes |
| `00FA00FE` | 130,048 bytes |
| `02020113` | 140,800 bytes |
| `031F0159` | 176,640 bytes |
| `04820138` | 159,744 bytes |
| `05c400f8` | 126,976 bytes |

These resources contain both artwork and level-specific program routines. A decoded payload is therefore an analysis artifact, not a resource suitable for embedding in the Go game. Exported game resources must describe only the recovered artwork, maps, paths, spawn events and other semantic data. Level-specific rules are rewritten in Go.

The first five payloads use a shared header at the original address `0x54e00`. Its named data will be documented as the artwork, map and encounter formats are verified. The sixth payload contains the shop routines, equipment catalogue, dialogue and artwork. Its layout is examined independently from the five level headers.

## Timing

The vertical-blank interrupt increments the display counter and samples the joystick. The main loop waits for four counter increments by default, or five in an alternate startup mode. The supplied trained release also increments this counter in an added interrupt hook, so four increments do not mean four display frames. The default threshold can represent two PAL refreshes, or 25 simulation passes per second. Trainer changes must be separated and live original measurements made before confirming the effective cadence of each screen and transition.

The Go version can draw at 60 frames per second while preserving the original simulation timing. Visual interpolation must not accelerate enemies, projectiles, level scrolling, firing delays or shop interactions.

## Analysis tools

Ghidra 12.1.4 and OpenJDK 21 are available for static disassembly. FS-UAE and vAmiga are available for original-game observations. Tool projects, recovered executable images and reference captures stay local. No additional tool installation was required for the initial disk and decompression analysis.

## Verification still in progress

The following require native-data or original-runtime comparisons before they can be described as faithful: level layouts and scroll bounds; enemy creation and trajectories; damage, collision and weapon timing; shop stock and prices; one- and two-player progression; music, effects and menus. A working subset does not establish a complete conversion.

## Verified level rendering data

The first five level headers describe a 320 × 192 scrolling background, a 20-column tile map, tile artwork and a 16-colour palette. The background uses two interleaved bitplanes. Each map cell is a big-endian 16-bit reference to a 16 × 16 tile. Empty cells preserve the background. Opaque tiles have four bitplanes; masked tiles have a separate one-bit coverage plane followed by four colour planes. Coverage, rather than palette index zero, determines transparency.

The game doubles each original three-bit colour component before writing the Amiga colour registers. The corresponding eight-bit components are therefore `component × 34`, including a maximum of 238. Expanding the palette to a full 255 would change the original image.

## Verified moving waves and paths

Moving-wave records contain seven 16-bit values: scroll trigger, enemy kind, number of enemies, one-based path selection, formation spacing, firing rate and movement budget. The movement budget determines how many path substeps are processed during one game pass. It must not be confused with a frame rate or a direct pixel speed.

Paths contain six semantic operations:

| Operation | Data |
| --- | --- |
| End | Remove the actor or its linked group |
| Curve | Initial heading, angular velocity, angular acceleration and substep duration |
| Pause | Substep duration |
| Random branch | Eight candidate targets, with unavailable candidates skipped |
| Jump | A target within the current path |
| Origin | Initial X and Y position |

Exported paths resolve original byte offsets into command indices and use these operation names. Runtime path execution does not retain native addresses or dispatch through original instructions.

Curve motion uses a recovered 256-entry signed sine table with amplitude 64. The X component uses the table a quarter-turn ahead of Y. Positions use 16.16 fixed-point values; heading fractions and signed angular velocity/acceleration are preserved. Rounding these values to whole pixels at every substep would change the trajectories.

Formation spacing below 100 delays following actors along the path. Values of 100 or above also offset their starting X positions; the remainder modulo 100 determines the separation. Enemy descriptors can link multiple pieces into a single composite actor, so a wave count is not necessarily the count of independently colliding sprites.
