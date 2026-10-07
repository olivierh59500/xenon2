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

The vertical-blank interrupt increments the display counter and samples the joystick. Its called hook increments the same counter again before an optional delay. The main loop waits for four counter increments by default, or five in an alternate startup mode, so four increments do not mean four display frames. The default threshold represents two PAL refreshes, or a maximum of 25 simulation passes per second when processing fits that interval. The hook's presence alone does not establish that it is a trainer modification. Live original measurements remain necessary for screens whose processing exceeds the wait interval.

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

## Independent movement verification

The Go path follower has been compared with bounded executions of the original isolated path routine. All 418 recovered paths across the five levels match over 42,408 recorded gameplay passes, including 16.16 positions, fractional headings, angular velocity and acceleration, command transitions, pauses, branches and removal. Random branches consume the original recorded random outputs during the comparison.

The independent ship-motion routine also matches 240 original directional-control passes across all three speed tiers. This covers held controls, release drift, reversals and vertical limits. The original checks vertical limits before movement, so an overshooting step reaches the boundary on the next pass; immediate clamping would differ.

These comparisons establish the isolated movement rules. They do not yet establish complete enemy combat, linked-body behavior, guardians, terrain collisions, shops or full-game progression.

## Equipment and shop rules

The Go equipment model retains the original seven positions: a primary gun, four additional mounts, a rear attachment and a side attachment. Each weapon keeps its own power tier. The firing period is shared and depends on the order of equipment changes; it is not recalculated from the final inventory.

The shop's compatibility checks match 1,325 isolated original cases. Buying and sale calculations match 150 original quote cases across ordinary and final-level shops. Equipment installation, power selection, temporary loadout replacement and timers match 234 original steps, including the complete 170-pass Nashwan duration. Buying Nashwan starts its timer in the shop; its temporary suite is installed after departure, so the saved loadout includes intervening trades.

Powerup advances one eligible weapon with the lowest current tier. Ties use the original position order: primary, four mounts, rear and side. Shop checks remain separate from pickup initializers because pickups can replace equipment that must first be sold in the shop.

Level five halves buying prices. Sale refunds use half the undiscounted catalogue price plus 1,000 per weapon power tier. Stock limits are independent of the wallet:

| Level | Middle shop limit | Final shop limit |
| --- | ---: | ---: |
| 1 | 600 | 3,000 |
| 2 | 1,200 | 4,000 |
| 3 | 2,000 | 5,000 |
| 4 | 2,000 | 6,000 |
| 5 | 6,000 | 6,000 |

The basic forward gun does not appear in the sale inventory. If the primary gun is sold, the basic gun is restored when leaving the shop. The initial inventory has three ships, a shield value of 39, speed tier zero, firing period eight and firing advance one.

## Artwork format distinction

Ordinary masked sprites store four colour planes followed by their coverage plane. Masked terrain tiles place coverage before their colour planes. These two formats require different decoders. Sprite collision rectangles precede the image header and are independent of the visible coverage mask.

The attract-loop logo is a 208 × 54 image whose sixteen-pixel words interleave four colour planes. Its stable display position is X = 48, Y = 20, with a separate attract-loop palette. Its zoom animation must be compared separately from this static artwork.

## Offline instruction verification

The installed Ghidra 68000 model does not copy carry into the extend flag after `ADDX`. The offline comparison harness corrects that flag according to the [Motorola instruction reference](https://www.nxp.com/docs/en/reference-manual/M68000PRM.pdf), before recording the shared random stream. The Go random generator matches 320 corrected original calls across four seed cases, including carry boundaries. This correction is confined to local analysis and does not introduce an emulator dependency into the game.

## Combat and scrolling comparisons

The independent combat helpers match bounded executions of the original routines:

| Behavior | Original comparisons |
| --- | ---: |
| Eight-direction targeting | 13,225 target deltas |
| Inclusive actor collisions | 625 cases |
| Shield, protection and damage suppression | 192 cases |
| Shared firing cadence | 720 passes |
| Directional projectiles and clipping | 552 passes |
| Formation initialization | 78 actor parts |
| Enemy firing and random consumption | 4,320 passes |
| Four ordinary player weapon patterns | 18 projectile creations |
| Cash and pickup movement | 277 passes |
| Carrier reward selectors | 19 complete equipment outcomes |
| Eight-entry wave reward cache | 39 operations |
| Moving scroll bounds | 224 cases |

Actor collision rectangles include their final pixel on each edge. Ordinary
player bullets use point collisions against the newest enemy first, then the
level's explicit fixed-object collider. Generic opaque terrain does not stop
every player bullet. Enemy bullets retain fractional direction vectors and the
scenery's actual scroll displacement.

The upper scroll limit moves during play. Sustained reverse movement doubles
after thirty-five passes, while holding down at the upper limit omits the normal
sixteen-pixel reverse buffer. Guardian controllers can expand these bounds again.

## Carriers and rewards

Moving encounter kind zero creates a power-up carrier. Its path budget is three;
the encounter's nominal movement-budget value instead selects one of nineteen
rewards. These carriers use the common artwork bank and their own damage callback.

Reward 13 selects homing missiles. Reward 16 adds 170
passes of invulnerability, independently of the Nashwan timer used by the shop.
The screen-clear reward strobes the palette for thirty-one PAL vertical blanks,
then applies damage through eligible enemies' own callbacks. Guardian resource
tags 80 and 84 are excluded. This is a timed game sequence, not an unconditional
delete-all operation.

Waves created within one four-pass period can share an entry in an eight-entry
reward cache. Killing its last member creates cash; allowing any member to escape
invalidates that shared entry. Carrier creation registers a count before clearing
the carrier's own token, so its orphan count can suppress a shared wave reward.
Preserving that order also preserves subsequent token allocation.

These comparisons verify the listed helpers and exported data. They do not yet
prove complete guardian behavior, all advanced weapons, the full shop interface,
death and continue screens, multiplayer progression or a complete five-level run.

The four ordinary weapon families cover all three power tiers, including the
creation order of double and side shots. Bitmap Shades darken the original
three-bit palette and apply a short-range damage area around the ship; they are
separate from both invulnerability and Nashwan equipment.

## First final guardian

The Go controller matches 1,200 original passes through the original waiting,
extension, recovery and vertical movement cycles. Its eight articulated pieces
match 9,280 recorded segment states, including fractional positions, headings,
angular velocity/acceleration, movement budgets, the final piece's target-facing
artwork selector, firing accumulator and random consumption.

The broad body collision is separate from its vulnerable eye. A projectile can
hit the body and disappear without reducing health when it misses that eye. The
articulated pieces block shots and damage the player; their ordinary collision
does not award a fictitious destruction score. Death releases nine pairs of
small and large cash rewards, appended in alternating order at the source
projectile-list tail and tracked by the global pending-exit count.

The integrated controller remains subject to full scene, checkpoint and stage
progression comparisons. Other guardians and scripted scenery require their
own independent Go controllers; their extracted artwork does not establish
complete behavior.

## Saved player state

The original player-switch routine exchanges 202 four-byte values from the
saved gameplay region, the complete ship state and all 6,000 map cells. The
continue counter and advice offset are inside the saved region, so both belong
to each player's game. The global random generator remains outside it and is
shared across turns and menu/shop animation.

This confirms the Go session's independent mutable maps, inventory, checkpoint,
advice and continue credits. Two available continue offers correspond to the
original counter initialized to three and decremented before displaying an offer.

## Second-level arena and fixed-object attacks

The second level's body controller matches 12,000 source passes, including its
preceding-pass moving-enemy count. Node callbacks, two defense streams, segment
breakup, hatch creatures and turret conversion have separate source comparisons.
The Go World connects their damage callbacks, gate counters, reversible camera,
mutable tiles and reward counts. Its integration tests distinguish the middle
shop request from final guardian completion and delayed stage exit.

Fixed scenery attacks now use their source movement and animation state rather
than a generic scrolling placeholder. The third level's turning shots retain
fractional motion, while fifth-level aiming shots retain their lifetime and
octant steering. Expiration creates the original centered explosion and sound.
Shared-capacity actor allocation is verified separately; its complete World
integration remains in progress.

## Shared actor storage and terrain cannons

World now routes enemies, equipment, projectiles, collectibles and transient
effects through the shared 159-slot allocator. The ship has separate state;
four thrust silhouettes reserve protected player-list slots. Reusing a slot
retains named fractional, firing and drift residue while assigning a new entity
identity. Eviction does not run damage, score or reward callbacks.

Moving, scenery and projectile traversal save the next physical slot before
each update. Self-removal remains linked until the next visit, while an entry
marked dead by an earlier actor is released when reached in the current pass.
Checkpoint cleanup preserves surviving scripted actors and reconstructs the
saved equipment without leaking slots.

The terrain cannon families across all five levels match 2,240 original
updates. Comparisons include animation phase, tile writes, firing triggers,
random consumption, shot origin and direction, and inclusive collision bounds.
World integration installs and changes their mutable map patches, restores
destroyed tiles and uses the exported shot artwork. These checks cover the
cannons; they do not establish every gate, hatch or guardian in the game.

## First-level middle arena

The five defense streams use independent rotated seeds and sixteen launch
paths. Each stream has eleven visible followers, an invisible path anchor and
two collision-list markers. The followers copy the next member's preceding
position in source traversal order, preserving their integer position writes
and inherited fractions rather than independently advancing the path.

The scheduler and sixteen gate counters match 160 original passes. The anchor
and follower helpers match 17,280 source states over all sixteen launch paths;
an additional World comparison verifies their displayed positions and visibility
in the integrated physical-slot list. Crossing the middle region requests the
shop and sets the original camera bounds without finishing the level.

## Second-level fixed hatches

The scenery hatch opens its original twenty terrain frames before releasing
eight one-hit creatures. The two hatch variants match 48 original updates,
including their tile writes and spawn boundary. Creature motion matches 341
source passes across all eight initial headings, preserving the finite timer,
periodic inverted aim, direction-specific animation and screen clipping.

World creates the creatures after the moving phase, so they first advance on
the following pass. Each uses the shared actor allocator, its original launch
offset and random delay, source artwork, collision and destruction effect.

## Second-level pods and third-level guardians

Both pod emitter cycles match the original terrain writes and two-creature
spawn limit. Their ship-seeking creatures match 5,120 source updates over both
variants and sixteen allocation phases. The allocator carries a named phase
value for per-object wobble; native addresses are never used at runtime.

The third-level middle guardian has seventeen independently allocated parts
and two shared eye health values. Its controller matches 20,400 part updates.
The final eleven-member worm matches 13,627 member updates; its creation retains
reused fractional positions and the inherited firing-rate value. Stage-boundary
logic matches 60 original decisions. Integrated tests verify source activation,
checkpoint preservation, middle rewards/shop admission and final delayed exit.

## Combat rendering and prolonged arena state

Ordinary damage now flashes surviving enemies and emits the source-centered
small or large explosion at destruction. Eight isolated source callback cases
verify the health result, effect origin, sampled sound and score. Linked groups
omit the invisible linked pieces' explosions. The first final guardian also
uses its four-phase lower-body tile decoration and twenty-explosion death burst;
the decoration matches 1,200 source passes.

A prolonged first-arena run exposed retained collision-list markers after their
chain disappeared. The cleanup now preserves the source pair: the trailing
marker is released with the last follower, then the leading marker is released
on its next visit. A 400-pass comparison over all sixteen paths checks actor
types, visible positions and exact removal timing. A 10,000-pass stationary arena
test checks that completed groups do not accumulate reserved slots.

Direct physical-slot lookups now serve projectile updates. The local simulation
benchmark exercises the original middle arena and weapon fire; renderer/GPU
performance still requires live testing and is not established by this benchmark.
