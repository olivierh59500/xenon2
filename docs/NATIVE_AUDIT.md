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

## Original terrain contact

The ship's thirty-pixel collision stencil matches 10,455 sampled source contacts
across all five original maps, three ship rows and forty-one horizontal positions
over seventeen camera offsets. The comparison supplies both the map row and
fine-scroll offset, and reads the original collision condition rather than the
loop-count register left by the routine. This verifies terrain contact, not a
complete wall-sliding/crushing playthrough or every guardian's custom collider.

Twenty-four isolated rewind cases additionally compare the original history
selection, timer decrement, upper/lower screen clamps, adjusted reverse scroll
and terminal crushing condition. The integrated terrain-contact sequence still
requires a full original playthrough comparison.

## Remaining third-level encounters and fourth-level guardians

All third-level fixed selectors now have semantic Go controllers. The extending
chain matches 6,400 source part states, the tile-edge crawler matches 3,200 passes,
and the compound terrain cannon matches 2,000 passes through both damage stages.
The single initialized scenery actor displays its sparse original tile composite
only while the final tail requests it. Actual-resource tests exercise these
selectors, the cannon's two score awards and its final map removal.

The fourth-level middle and final guardians match 24,000 and 11,400 source part
states, respectively, including firing and shared random consumption. Twenty-two
source damage operations establish their outer targets, core locks, satellite
edge restrictions, closed eye poses, rewards and terrain changes. World tests
verify both original constructors, persistent checkpoint bindings, the middle
corridor and twenty final reward coins before stage exit. Full scene comparisons
remain necessary beyond these controller and integration checks.

## Fourth-level nest crawler and fifth-level guardians

The nest crawler matches 1,600 source updates and six damage operations. It
keeps world coordinates, the opening animation's held poses and its vertical
escape choice. Destruction awards score and a large coin before restoring its
nest and original health. The renderer carries its original three-tile cover.

The fifth-level middle and final guardians match 15,000 and 26,400 source part
states, plus 144 damage cases. Their growing columns, side shots and mouth
creatures match 1,867 additional source states. World integration retains
noncollidable wrecks, eighteen outer defenses before core exposure, mutable
arena tiles, persistent checkpoint damage and twenty final cash coins. Column
render descriptors preserve the source cap artwork and twelve-pixel palette
pattern. The ordinary fixed encounter families and complete runtime comparisons
remain separate unfinished work.

The fifth-level barrier and aiming turret additionally match 7,744 source update
states and sixty damage/map cases. The barrier uses three separately allocated
moving-list pieces with two vulnerable ends. The turret retains its eight-state
spin-up, one-octant aiming turn, carry-triggered fire, original shot artwork and
conditional neighboring-tile changes. Exported-resource tests exercise both
families and their shared allocator behavior.

The radial terrain turret adds 920 source update comparisons. Its bursts create
eight animated projectiles in descending direction order, while the tile
animation advances every two gameplay passes. Six additional damage cases
check its five-hundred-point reward and neighboring-tile restoration.

Five hundred and twelve contact cases distinguish the side seekers' displacement
point probe from the mouth creatures' sprite rectangle. They include dive and
invulnerability states and the side seekers' expiry explosion. Mouth creatures
enter the moving-list tail; side seekers enter its head. Their initialized slot
state and score values are checked with exported resources.

Twenty middle/final core damage cases additionally compare 450 created cash and
explosion entities, including their creation order, coordinates and final shared
random state. Middle defeat creates explosions before cash; final defeat creates
cash first. Nonlethal final-core damage selects the original health-dependent
image from its twelve-entry table. The common explosion factory also retains
its immediate audio dispatch alongside the two queued requests.

## Shared stage and completed-player admission

Eighty source routes cover all five stages, both current-player indices,
surviving/completed opponent combinations and Nashwan activity. After both
surviving players finish, the next level clears the completion flags and admits
the other player. The fresh worlds retain independent equipment and maps.

Twenty additional source routes cover an opponent declining its final continue
offer. An already completed surviving player still receives READY. It then
loads the next level, or receives the merchant ending if the completed stage was
the fifth. That reopened fifth-stage completion retains its source credit award.
Music stops before the first player's fifth-stage ending is skipped.

These route comparisons establish admission and completion branches. Integrated
fade/audio overlap, live scene composition and a full five-level playthrough
remain separate validation. Repeated checkpoint restoration across alternating
turns is also under its own state comparison.

HUD pixel comparisons now cover four score/lives/shield combinations under all
five level palettes, including the second player's layout. The renderer
prepares separate recolored HUD images and fonts for each level; it no longer
retains level-one colors when the playfield palette changes.

## Remaining fixed selectors and timed visuals

The last level's three ten-member formations match 27,030 native construction
and update states, including staggered entry, heading-dependent images and
eight-shot bursts. Eighty damage and re-entry cases verify the two persistent
turret selectors. Every fixed encounter selector in all five exported streams
is consumed by an explicit World implementation.

Thirty fixed-sprite constructors retain their original tags and slot values.
Fifty-six beam layouts compare the repeated masked shaft and displaced animated
tip. Eight Supernova traversal cases verify group-aware order, tag-20 projectile
removal and clearing the eight wave-reward buckets. Transferred bouncing attacks
continue their source callback after entering the effect list.

The invulnerability aura matches forty guarded updates and four constructor
cases. Its original animation advances while ship materialization freezes the
timer; consecutive pickups reuse one actor. Expiry keeps the final meter for
that pass before releasing the aura on its next list visit. Four forced-restart
cases retain an unusual source edge case: cleanup removes the aura while leaving
its timer unchanged. This does not represent an ordinary death path.

Three hundred and six image selections cover the diving and Nashwan meters.
Their countdown images appear at the source anchors, alongside the aura's
separate meter. One hundred and eighty consecutive source ship/terrain passes
also compare evolving position history, wall contact and rewind behavior.

## Final shared-state boundaries

Forty-eight incoming checkpoint states compare ship position, camera, wallet,
shield, firing advance and temporary timers. Alternating-turn tests keep the
outgoing game suspended until it is admitted again, retain guardian slots and
damage, and apply camera adjustments once. Death resets shield and firing
advance separately from checkpoint admission.

Fourteen second-level constructors verify emitter and offspring slot fields.
The small scenery pod uses tag 252; the hatch constructor reads launch positions
and initial images forward while assigning descending headings. Their stored
state preserves counters, directions and fractions instead of replacing them
with the invisible emitter's screen coordinates. Wave animation modes also
retain their path and firing state. Sixty-six image-center cases cover thirteen
banking positions and nine death images at three ship positions.

The second final guardian's crowded-scene direction inherits a named updater
decision. Comparisons cover eighty path-to-guardian chains, 1,200 bouncing
callbacks, 5,120 pod-creature callbacks, 341 hatch-creature callbacks, 14,400
final-minion returns and 416 player-shadow cases. The runtime carries semantic
direction outcomes rather than original register state.

Sixteen accepted-continue routes compare saved ships, zeroed score, consumed
credit and admission of the other surviving incomplete player. Single-player
restoration remains a single checkpoint operation. Gameplay Escape now restarts
the attract sequence, while P preserves the image and continuing audio until
the next key or fire input.

## Integrated draw timing and compound motion

The native backdrop is drawn before stage and actor callbacks. Materialization
rendering reads the live terrain after actor updates, before late encounter
constructors and scroll advancement. Two reusable map snapshots and their
camera positions preserve those distinct presentation boundaries without
changing the live collision map.

Compound image pieces retain their own previous endpoints for 60 Hz rendering.
The first guardian's eye and final mouth poses follow parent translation; beam
tips follow extension endpoints, while shaft image counts remain discrete.
Crawler nest covers follow the camera rather than the moving creature.
Checkpoint admission snaps the first resumed endpoints. Six composition and
restart checks pass, and all fifty-six native beam layouts remain unchanged.

Bounded runs using ordinary inputs exercised menu, loading, READY, ship losses,
initials, accepted/refused continues, alternating players and return to attract.
Separate shop fixtures verified sale, purchase, exit and ending presentation.
They are not full-game wins. Early navigation trials reached the first
middle-arena exit region; the subsequent first-level engine replay below
reached both shop boundaries and victory. A complete normal-menu five-level
run and integrated Amiga comparisons remain required.

## Actual desktop graphics verification

The Ebitengine test loop renders the production Game.Draw and reads GPU pixels.
Six real-resource fixtures display guardian bodies, eyes, chains, articulated
parts and the extending beam. Exact pixel checks cover aura transparency,
live-terrain materialization, masked body tiles and damage flashes, moving/effect
painter order, all five palette strobes and Shades, and the 48 background stars.
These tests found and fixed a shader source-size panic and a duplicated source
origin adjustment in multi-image sampling. The resulting desktop captures have
been inspected; they are composition fixtures rather than completed games.

The production frontend is also exercised from sampled controls through menu,
loading, READY, gameplay movement, pause and return to attract. Audio integration
checks verify immediate versus next-interrupt effect dispatch and delivery of
voice-ownership flags to gameplay.

An earlier legal-input replay reached the first middle shop and final arena and
reduced the final guardian's health from thirty to two before losing the last
ship. No full-level or five-level victory is inferred from that result.

A historical public-input replay completed level one before the later contact
correction: both shop boundaries,
guardian destruction and exit-coin exhaustion are reached after 8,198 steps.
The replay comparison checks every observed frame, camera, ship coordinate,
life count, shield, score and wallet value. It uses normal Session input,
continue APIs, actual terrain coverage and shop prices. The final state has one
ship, seven shield points, 5,300 score and 2,000 cash. This proved the earlier first-level
engine route; it does not prove the four remaining levels or full frontend
campaign. The replay is local and excluded from Git.

Materialization now uses pixel-unit shader triangles directly on the sprite and
the terrain mask. It avoids a playfield scratch clear, sprite copy and full-size
shader rectangle for each masked actor. Exact GPU checks preserve source mask
pixels and sprite clipping at all four playfield edges.

An actual realtime dense-scene probe uses nine visible materializing actors from
the second-level defense streams. A static scene isolates rendering cost. The
direct path reduced observed allocation per draw from about 167 KB and 6,081
objects to 85 KB and 3,005 objects on the tested Mac. Backend allocations remain
substantial. Both probes sustained approximately sixty update calls per second;
display-rate changes during the runs prevent claiming an FPS improvement.

## Live Amiga presentation and frontend boundaries

The supplied disk was booted in FS-UAE 3.1.66 with an A500, PAL video,
512 KiB chip memory, 512 KiB slow memory and cycle-exact CPU/blitter timing.
All five trainer options were set to NO and the starting level remained one.
The trainer uses vertical mouse movement and its mouse buttons; the game's
joystick controls are separate. The configuration and window captures remain
local.

A forty-second window-only capture contains 400 timestamped images. Five
successive credit-pair onsets are separated by approximately 6.11, 5.80, 5.80
and 6.30 seconds. The source director uses 93 passes between pairs, giving an
approximate observed presentation rate of 15.5 passes per second in this run.
Caption width and zoom affect threshold-based onset identification. The capture
begins inside a cycle, and it does not measure gameplay cadence. The source's
25-pass rate is a maximum when processing fits two PAL refreshes; it is not a
measurement of every original scene. The Go clock has not been changed solely
on this presentation observation.

The installed reference emulator's OpenAL backend failed to create its sources.
The capture used its silent fallback, so it provides no live soundtrack
comparison. Original audio-event comparisons and Go audio integration tests
remain separate evidence.

Integrated frontend checks enter through the ordinary menu and exercise actual
collision death, three-letter score entry, accepted and unanswered continues,
middle-shop quotes and purchases, same-stage reload, and the shared two-player
fifth-stage ending gate. Last-life/ranking and shop-completion arrangements are
explicit boundary fixtures, not a completed campaign.

GPU checks found an outgoing fade still covering the entering-shop caption and
the ending dot. Completed outgoing fades are now removed before callbacks can
install their next scene or fade. A terminal fade without a callback still
retains its black palette. New READY and final-loss states enter their original
presentation directors before the next draw, eliminating a one-frame static
logo and player-one caption on player-two admission.

A separate normal-menu run reached READY and started level one with an ordinary
fire edge. Matching opaque original terrain pixels against the exported map
locates camera positions 4604 and 4504 at timestamps 1.5447 and 7.5426 seconds
after capture start. The 100-pixel displacement over 5.9979 seconds gives
approximately 16.67 source pixels per second, or 50/3 gameplay passes per second
with the original one-pixel base step. Twelve half-second samples consistently
advance by eight or nine pixels. The interval ends before the passive ship's
collision death. A second attempted capture did not leave READY and provides no
confirming gameplay measurement.

The optional `-logic-pal-refreshes 3` profile represents this early A500 cadence
with an exact rational accumulator. It preserves the independent 60 Hz display
and 50 Hz audio/effect/fade clocks. The default remains the original two-refresh
maximum until further gameplay intervals establish the appropriate rates for
dense combat and other levels. Presentation and shop clocks are unaffected.

The rebuilt desktop executable was also started from its ordinary menu with
this measured gameplay profile. Separate Enter and fire inputs admitted READY
and gameplay; a bounded capture after the fade shows the normal ship, terrain
actors, starfield and HUD. The clock test preserves 60,000 logic passes and
180,000 independent PAL ticks over one hour, without fractional-rate rounding.
Frontend admission/fade tests retain the selected profile while presentation
and shop timing remain unchanged. This validates the selectable profile, not
complete original cadence equivalence.

## Independent PAL and gameplay scheduler boundaries

The optional three-refresh profile is checked at 50, 60, 120 and 144 display
updates per second. At each eligible logic boundary, its accumulated PAL ticks
match three prelogic ticks. A supernova strobe ends on its 31st PAL tick; the
next eligible gameplay pass resumes at tick 33. Logic-timed equipment remains
unchanged during the strobe, and paused updates retain both clock remainders.
Alternating-player death and READY admission retain the incoming checkpoint
and gameplay timers under this profile. These are explicit timing fixtures,
not additional campaign victories.

The historical first-level CSV checks every recorded input and observation
with two PAL ticks grouped after each logic step. It remains useful profile 2
engine-path evidence. It does not exercise the application's PAL-before-logic
dispatch at every display update, and must not be relabelled as profile 3 or
normal-menu campaign evidence. Grouping the same three ticks after logic
instead would delay supernova gameplay resumption by one logic slot.

## Lethal contact returns before enemy rewards

The original player callback returns immediately if moving-enemy contact kills
the ship. It does not damage that enemy, consume its wave reward, draw explosion
random values, move the ship, overwrite the one-pixel scroll request or shift
the four ship-history poses. The remaining actor/shadow callbacks still run.
The Go integration now preserves that boundary. 112 native branch comparisons
cover lethal/nonlethal damage, protection, invulnerability and Shades; a world
regression additionally checks the retained enemy slot, history, score and RNG.

This fidelity correction invalidates the historical 8,198-step first-level
observation CSV: its first mismatch is step 920, immediately after a lethal
contact. The old local CSV is retained unchanged and no longer selected
automatically by tests. Explicitly supplying it still reports the mismatch.
A new current-engine first-level victory and full campaign remain unproven.

## Integrated music admission and free DMA voices

The main music replay remains active during death, score initials, continue and
subsequent READY. The initial gameplay admission starts before its first fade;
shop return and newly loaded-stage admissions restart after their fade. Explicit
frontend state preserves these distinct native calls. PCM comparison over
428,505 output frames covers a real collision/continue route against an
independent uninterrupted musical replay.

Stopping all effects previously restored every musical voice and reset free
voices' sample positions. The native effect records restore only owned active
voices. The stream now preserves unaffected DMA positions, including queued
termination. Independent PCM tests distinguish both immediate and queued stops
from a continuous reference. Live emulator soundtrack/filter comparison remains
separate.

## Immediate sampled-effect admission

Eighteen gameplay samples and four merchant samples now carry separate named
admission events. The original initial period, volume, sample and reload-buffer
setup applies at a direct call rather than waiting for the next IRQ. Queued
admission performs that setup on its dispatch IRQ; sampled termination ages
and synthesized envelopes remain unchanged. Independent mid-tick PCM tests
check the known signed first sample and distinguish these three boundaries.

After rebuilding the excluded audio banks, all 12,300 gameplay-effect, 8,100
merchant-effect, 18,000 main-score and 6,000 menu-score reference rows still
match. JSON normalization verifies that only the initial four events moved for
the sampled sequences; other descriptors and timer events are unchanged.

## Current normal-menu pilot first-stage victory

The demonstration controller now caches world-coordinate terrain routes and
checks ship coverage after scrolling as well as before it. The additional
check prevents a repeat contact/rewind cycle at the bottom of the first middle
arena. Route planning and bonus/dive decisions leave game state untouched.
The verified second-stage opening still reaches checkpoint 4032 after 578
ordinary commands with three lives and 27 shield points.

A new current-engine frontend regression starts from the ordinary menu and
completes level one after 28,649 display updates with the three-PAL-refresh
profile. It explicitly requires the first guardian to be defeated, exit drops
to be exhausted, intermediate and final merchant visits, and normal admission
to level two. The next stage starts with one ship, 39 shield points and zero
cash after real purchases. This is approximately 477.5 seconds of displayed
gameplay/presentation, not a sped-up game.

This proof replaces the invalidated historical first-level observation as
current frontend progress. It does not prove the remaining four stages or
a strong full-campaign pilot. Cached navigation without a changing route
measured about 2 microseconds with no allocations on the tested Mac; route
replanning and dense-combat rendering have separate costs.

A longer bounded frontend run reaches the second-stage middle merchant and
continues into its second section, but loses the last ship near camera 1476.
After a fresh game, another first-stage route can still enter a terrain/rewind
cycle near camera 2619. The first-stage victory above is a reproducible specific
normal-menu run, not a general guarantee over arbitrary presentation RNG states.
The long-run check reports progress and does not treat exhausted lives or
restarting attract as campaign completion. Later guardian strategies and robust
recovery from terrain cycles remain required for the requested full-game pilot.

## Demonstration merchant budget

The pilot chooses one affordable compatible item across both merchant pages.
Original repairs add 20 or 40 and clamp to 39; at shield 19 or above the smaller
repair already reaches full energy. Repeated rear upgrades are capped while
autofire, one speed step, a base rear weapon, one cannon and primary power are
selected. Low ship-count savings are retained only where that merchant can
actually stock an extra ship.

Real-UI merchant fixtures verify the cheaper repair/base-rear budget and an
extra ship selected on the second page before affordable optional first-page
items. These fixtures do not claim guardian victories. The updated normal-menu
pilot still completes level one after 28,354 display updates; next-stage survival
and a complete autonomous campaign remain under validation.

## Ordinary targeting of the second middle arena

The pilot targets visible defense-stream heads while both stream flags shield
the nodes. After an actual head destruction, it crosses to each surviving node
using its semantic tile position, including when the current player side keeps
that node's collider closed. A visible firing window can be retained through
the game's normal reverse-scroll controls. No node health or flag is assigned
by the pilot.

A basic-equipment level-two public-session regression destroys all three nodes
through 36 observed hits and reaches the actual middle shop after 4,291 commands,
with one surviving ship, 31 shield points, 3,100 score and 1,100 cash. It consumes
one legal continue after five ship losses. The first opening checkpoint remains
unchanged at 578 commands. A normal-menu carried-loadout run also passes the
middle arena, but currently needs corridor recovery near camera 910 and a
validated final-barrier/guardian strategy. This is not a second-level victory.

## Second final barrier and guardian control

An explicit post-middle checkpoint fixture retains original terrain/stencil,
basic equipment and 75 HP guardian health. The pilot's ordinary firing opens
three columns of the lower destructible barrier. A cached safe path reaches
the activation threshold, then retreats through the real opening and aims
below the central rock. No tile, health or damage is assigned by the pilot.

The regression requires actual final-shop readiness after 754 public commands:
guardian HP 0, two surviving ships, 11 shield points, 11 cells cleared, no continue
and all 20 exit coins collected or expired. It also checks that a negative wait
during active guardian travel is not mistaken for the initial dormant phase.
This proves the arranged final-arena route, not a completed second stage or
five-stage campaign. Corridor recovery before this arena remains open.

## Connected first two stages through the normal frontend

The second-stage corridor policy plans beyond the right-hand dead end and
uses a short changing-command beam with a margin around predicted enemy
collisions. An original checkpoint fixture traverses it in 909 public commands
with one ship and 7 shield points. The final guardian controller now first
reaches its legal preparation position from cameras above 288 before opening
the barrier, rather than starting from a fixture-only pose.

A current frontend regression completes both original stages from the ordinary
menu after 55,330 display updates at the measured gameplay profile. It requires
both guardians defeated, all final drops drained, both merchants on each stage,
no diagnostic route and no trainer options. Level three starts with one ship,
39 shield points and zero cash after real purchases. This is a reproducible
two-stage route, not the requested complete five-stage autonomous run.

## Ordinary final-worm targeting in the third stage

The pilot selects the real worm head; neck, body and tail remain armored shot
blockers. Lead prediction copies path and random state without advancing the
world's RNG, and nearby body contacts retain a movement safety check.

An original final-checkpoint fixture retains basic equipment, terrain, paths
and 80 HP shared guardian health. Ordinary commands defeat the head, drain all
20 exit coins and reach the final shop after 1,614 commands with one ship, 23
shield points and one legal continue. The regression requires actual defeat,
ExitReady and zero pending drops. It is a final-boundary victory, not proof
of the unverified middle eyes, preceding terrain or complete third stage.

## Ordinary third middle-eye control

The pilot follows an intact eye during its animation's firing windows and
moves toward a safe side when the articulated body descends. Six copied
controller passes predict the actual body rectangles without changing source
RNG, poses or health.

An original encounter/checkpoint fixture reaches the genuine intermediate
merchant after 1,082 ordinary commands: both 20 HP eyes destroyed, two surviving
ships, two legal continues, no pending middle coins and no final-level flags.
The exact final-worm fixture still completes its separate arena. Neither
fixture proves the ordinary stage-three route from the previous merchant;
the full five-level campaign remains under validation.

## Fourth satellite contact collider

The original fourth middle satellites carry guardian tag 84. Ordinary ship
contact already skips their damage callback after applying shield damage.
Shades invokes the callback with its entire 128-pixel attacking rectangle;
the satellite's armored top/bottom border checks therefore remain active.
The contact route previously substituted the enemy center point and could
incorrectly destroy the satellite, decrement an outer target and award 300.
It now forwards the actual attacking rectangle. Projectile contacts retain
their existing true rectangle and still admit interior hits.

Forty-eight original player/callback cases compare shield, death, satellite
health, outer-target count, score, actor retention, explosion and sound. The
single named explosion does not consume randomness, and both admitted and
rejected routes preserve the native RNG state. The lethal-player early return
remains unchanged. Synthetic fixtures use the actual source tag rather than
a normal-enemy tag that would exercise a different branch.

## Third-stage opening policy

A temporary pilot configuration targets the verified first-checkpoint window
with faster ordinary trigger releases and postpones risky bonus pursuit. It
returns to the general controller after the checkpoint and never changes
health, equipment or the shared random stream.

A fresh original-resource session reaches checkpoint 4032 after 578 public
commands with all three ships and both continues; shield is 3. An independent
CSV replay checks every observation. In the carried frontend campaign, this
policy improves third-stage reach from camera 3644 to 3192 before the last ship
is lost. The normal-menu victories of stages one and two remain intact.
Neither result establishes completion of stage three or the five-stage pilot.

## Consumed laser traversal

The laser has a dedicated damage result for source callbacks that retire the
projectile. Ordered traversal stops immediately after those callbacks, and the
beam's physical slot becomes the pending-removal tag before its draw boundary.
Verified absorbing identities include first-guardian links, worm body links,
fourth-final arms, fifth-final barriers and rejected fourth-satellite armored
borders. Closed scalar cores and no-op callbacks are not classified by unchanged
HP or by a generic block-shot label. Mines and bombs keep their separate
multi-target behavior.

Four original ordered three-target cases compare beam retirement, each target's
health, score and RNG. Runtime tests also verify no damage behind a blocker,
proper compaction, no active rendered beam and a nonconsuming no-op route.
The full source/resource engine suite and the connected first-two-stage frontend
regression pass after this correction.

## Animated guardian projectile collision

Third- and fourth-stage shot factories use the guardian-parts image bank. The
ordinary projectile phase previously looked up only default/fixed/guardian
images, leaving these visible shots without a damage prefix. It now reads the
already exported guardian-parts prefixes as the native current-sprite collision
routine does. Colliding shots apply the original four-point damage and retire
their physical pool entry; diving skips contact and invulnerability suppresses
damage while still consuming the shot.

A real factory regression and 69 World.Step cases cover all 21 unique recovered
shot frames, including duplicate eye animations and normal/invulnerable/diving
states. The historical third middle/final victory counts preceded this fix.
Re-running them with real bullet damage still reaches the genuine merchants:
third middle after 1,021 commands with both eyes destroyed, two lives/full
shield and two continues; final worm after 1,848 commands with three lives, 35
shield and two continues. Every defeat/drop/exit gate remains required. The
counts were updated only after these current ordinary-input outcomes passed.

## Hidden fifth-stage seeker admission

New side and mouth seekers retain the original off-screen collision sentinel
until their first actor callback. Previously their zero-value rectangle could
intercept a shot or Shades near the origin while the sprite was still hidden.
The first callback replaces the sentinel with the recovered sprite prefix;
visible movement, health, rewards and physical pool tags remain unchanged.

Both factories are covered before and after admission, including the ordinary
World.Step player-contact ordering. The full original-resource engine suite,
race suite, vet and desktop build pass.

## First-stage pilot crossing and carried lives

The default pilot gives navigation priority over loose bonuses inside the first
defense-stream arena. Ordinary trigger releases and a lower formation target
reach the real crossing without rewriting gates, collisions, health or RNG.
Explicit pilot configurations retain their own preferences.

The complete resource engine run now reaches the intermediate merchant after
2,025 commands with all three ships, full shield and both continue credits.
Normal-menu frontend regressions cover both two- and three-PAL-refresh gameplay
profiles and all four first/second-stage merchants. Both routes admit stage
three with two ships, full shield and no credits, improving the previous one-ship
admission. Shops use their real dialogues, random stream, quotes and purchases.
A connected third-stage victory is still unverified; separate arena wins and
the excluded instantaneous-shop replay do not establish that campaign outcome.

## Turning-projectile pilot prediction

The third-stage sweeper's shots keep their live motion in the specialized
turning controller, not the generic projectile field. The pilot now forecasts
a copy of that controller, including its 128/192 turning thresholds, fractional
travel and removal boundary. Ordinary linear-shot forecasts retain their
existing conservative behavior. General and arena planning share this lookup.
Synthetic threshold, fractional-motion and retirement cases verify that live
shots and their state remain unchanged. Connected frontend validation of the
third-stage victory remains open. Both first/second-stage normal-menu regressions
still pass after this correction, with unchanged outcomes at both cadences.
The carried campaign now reaches the actual third middle guardian at frame
1,793/camera 2815 with both ships and 19 shield. A frontend regression requires
the living ship, checkpoint and genuine encounter admission. The first death
in the separate causal trace occurs in that arena rather than the earlier
turning-shot passage; arena survival still needs refinement.

## Fourth final eye/core pilot

The final guardian's core becomes continuously available after both eyes die;
the alarm counter does not gate it. The pilot follows the body's actual screen
height after the eyes and avoids bottom-edge reverse-scroll requests that would
push the core out of view. Eight copied player, guardian, camera and random-state
passes evaluate ordinary movement choices without modifying game rules.

An original final-checkpoint fixture activates the guardian through the real
encounter stream. The normal pilot dispatch destroys both 50 HP eyes and the
100 HP core, drains the 20 exit coins and reaches the genuine final merchant in
1,785 commands, with two ships, 23 shield and no continues. Admission/resource
guards and prediction immutability are also tested. The source/resource engine
suite passes; the excluded overlay race suite passes. Steady decision work
measures about 59 microseconds with zero allocations on the M4 Max. This is an
arena proof, not a complete fourth-stage or five-stage campaign victory.

## Remaining newborn collision constructors

The 43 production actor/projectile constructor literals were classified by
their admission and contact routes: 30 set explicit birth rectangles, six set
sprite-derived rectangles before insertion, four create transient actors outside
moving-target scans, and three create point/motion projectiles without target
rectangles. The ten generic fixed variants have collision prefixes for all 48
animation frames. All 22 initial hatch, pod, falling and aiming images also have
their recovered prefixes. No further origin-collider defect was confirmed with
the supplied resource graph.

Ordinary waves deliberately compute their collision before the first display
callback. The original constructor's rectangle call confirms this behavior;
a global Visible filter would incorrectly change those waves. The fifth hidden
seeker correction therefore remains specific to its distinct constructor.

## Fifth compound barrier and wreck integration

The final guardian's two invisible armor bands now publish the rectangles
computed by their source updater. RenderMode none suppresses their images, not
their player/projectile contacts. Ordinary shots are consumed without damaging
the bands; lasers retire before reaching a later overlapping target. The bottom
edge retains the source low-byte addition, including signed coordinates and
byte-boundary wrapping.

Both fifth guardian factories retain exported contact strength. Ordinary mounts
deal eight points, while the bands and strong plates use sixteen. Protection,
invulnerability, diving and the lethal-player early return retain their distinct
source routes. Player contacts see the preceding actor callback's rectangle.
Destroyed middle/final mounts retain their wreck and reward but disable their
collider immediately, allowing a later shot in the same projectile phase through.
The displayed wreck image still changes on the following actor callback.

Seven resource-backed regressions cover these boundaries, and the complete
engine suite with all available native fixtures passes under the race detector.
The terrain audit also confirms that levels one, three, four and five have no
ordinary shot-blocking fallback; level two retains its special cell handler.
This corrects gameplay integration and does not establish a fifth-stage victory.

## Permanent A500 output response

The main and shop initializers disable CIAA's optional LED low-pass. Game output
now models the two permanent A500 RC stages while preserving that choice.
Raw digital PCM remains available for source-register diagnostics. Independent
impulse/frequency checks, stereo isolation, silence decay, arbitrary reader
chunks and unchanged replay state cover the filter. The filtered frontend PCM
at the source maximum matches 428,505 uninterrupted music frames through death, score entry,
continue and READY, and merchant direct/queued dispatch checks pass.

Steady filtered output measures about 29 microseconds per 1,024 stereo frames
with no allocations on the M4 Max. This is a numerical reconstruction model;
physical-output and complete soundtrack comparisons remain separate work.

## Measured default gameplay and credit pacing

Default gameplay now uses three PAL refreshes per logic pass, matching the
observed early A500 terrain rate instead of the source's faster two-refresh
maximum. The explicit two-refresh option remains available. This establishes
the early-scene default, not unmeasured dense/later-level timing.

The recorded attract sequence has approximately six-second credit-pair intervals
for 93 source passes. Its zoom now uses a separate rational 31/2 Hz clock,
preserving that average with no accumulated rounding drift. Actual frontend
credit-pair transitions occur at display updates 482, 842, 1202 and 1562: every
subsequent interval is exactly six seconds at 60 display updates per second.
This deliberately replaces workload-dependent processor delays with their
measured average; individual native zoom passes need not have uniform duration.
Menu/shop clocks, 50 Hz audio and palette fades remain independent, and a
confirmation still leaves credits through the ordinary menu transition.

The complete app suite passes after this change. Filtered music now matches
511,560 uninterrupted output frames through the death/score/continue/READY route
at the default cadence. Explicit full-intro pilot checks retain all six credit
pairs, the real menu, four merchants and both guardian/exit gates. They reach
stage three with one ship at either gameplay setting. Direct-menu checks still
reach it with two; neither result proves the remaining campaign.

## Third compound cannon contact inheritance

The common cannon constructor leaves the physical slot's strength byte intact.
The Go factory now copies that retained value instead of replacing it with false.
A real horizontal-sweeper contact releases a strong slot; the compound cannon
then reclaims the same slot, publishes its collider and receives normal player
contact. That contact now costs sixteen shield points rather than eight. A fresh
slot still costs eight. Both cases retain the stage-one transition, 24 HP,
500-point reward and source pool residue. The resource/native engine suite and
race suite pass. Other common-constructor inheritance is audited separately.
