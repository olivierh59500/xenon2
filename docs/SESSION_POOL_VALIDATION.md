# Actor storage across alternating players

The Amiga game has one reserve of 159 reusable objects. Each saved player owns
separate lists of equipment, enemies, projectiles and scenery, but both games
allocate from the same free stack. The ship uses dedicated state outside this
reserve; its four thrust silhouettes use ordinary protected object records.

The previous Go session created a complete reserve for each player. This doubled
available storage and gave both players' initial equipment the same physical
positions. The second player's objects therefore did not influence the first
player's allocation order or capacity as they do in the original.

## Original initialization evidence

An offline reference executes the original game and level initializers for all
five levels, with both one and two players. It observes fifteen saved-player
admissions from ten fresh sessions. The Go comparison checks free-stack heads,
list heads, all four shadow positions and the primary weapon position. Every
physical resource tag is checked from each player's view: 2,385 comparisons.

The original first creates both players' shadows and basic equipment, then
creates each game's initial level actors. Go now follows that order. For example,
level one with two players reserves twelve objects: five basic objects for each
player, followed by the two persistent guardian controllers. The second player's
weapon occupies slot nine; the guardian controllers occupy slots ten and eleven.
With one player, the corresponding six-object layout remains unchanged.

## Shared capacity and independent saved games

Each Go pool view retains its own list heads and refers to common physical
records. Allocation and release update one free stack. Reclamation visits only
the active player's expendable lists, so capacity pressure cannot steal an
inactive player's equipment or guardians. Creation identities remain unique
across the shared reserve, avoiding ambiguous stale bindings after slot reuse.

A saturation regression fills the remaining 149 objects after both players'
basic five-object initialization. Both views then report a full reserve. The
next allocation reclaims an active enemy without changing the inactive player's
protected records. Turn admission restores the saved game through the ordinary
session API. Loading a new shared level also retains one reserve for both games.
Maps, wallets, inventory, continues, advice and logic counters remain independent.
A separate original-resource fixture constructs both fifth-middle guardians,
saturates the reserve and performs eight active-player reclaims. Every physical
record belonging to the inactive guardian, its equipment and shadows remains
unchanged.

Forecasts copy the complete physical reserve and the active view's list heads.
Neither a first-player nor a second-player prediction can allocate from or change
the live free stack. Prepared encounter validation compares physical state rather
than the identity of a shared-storage pointer. Ordinary single-player predictions
retain their existing value-copy path.

## Retained storage across level changes

Original cleanup, equipment restoration, player switching and level initialization
retain the physical reserve across all five level transitions. The Go stage
loader now retains that same storage, its free stack and reused record fields.
Each new world gets its own list view over the surviving reserve.

Completion first performs ordinary turn cleanup, then releases remaining moving
actors, scenery and projectiles. The four ship shadows survive. Before building
the next level actors, the outgoing current player's saved weapons are restored.
The current game is initialized before the other saved game. Post-load cleanup
and READY then restore the admitted player. A player's first-ever turn restores
over its initial equipment without an extra outgoing cleanup.

Fifteen basic transition observations, 75 equipment observations and 45 ship-pose
observations match the original. Equipment cases cover basic weapons, cannon,
rear shot, laser, side shot, protection and the temporary Nashwan suite. An
arranged previously used free record retains its fractional position and counter.
The sparse one-player boundaries now agree:

| Transition | Free head, original and Go | Primary slot, original and Go |
| --- | ---: | ---: |
| 1 to 2 | 9 | 5 |
| 2 to 3 | 7 | 5 |
| 3 to 4 | 4 | 5 |
| 4 to 5 | 5 | 4 |
| 5 to next-round 1 | 6 | 4 |

New levels preserve the outgoing ship's X and save its previous Y in the
checkpoint. The admitted ship enters READY at Y=176; the other saved game's
pending position remains independent. READY also resets all four thrust-history
positions. Third-level persistent scenery now survives checkpoint cleanup and
has its own isolated forecast copy.

The offline fixtures initialize the known inactive temporary-suite timer before
original game setup. Uninitialized memory from a generic capture must not be
treated as an active saved loadout. File access, presentation drawing and the
stopped audio device are isolated; allocator, cleanup, weapon and level-actor
routines execute unchanged.

## Reproducible checks

With prepared runtime resources and the local original references:

```sh
GOWORK=off \
XENON2_RUNTIME_ASSET_DIR="$PWD/assets/runtime" \
XENON2_NATIVE_TRACE_DIR="$PWD/.local/analysis" \
go test ./internal/engine \
  -run '^(TestSessionPoolsMatchOriginalInitializersOptional|TestStagePoolRetentionMatchesOriginalTransitionsOptional|TestStagePoolEquipmentMatchesOriginalTransitionsOptional|TestStagePlayerPosesMatchOriginalAdmissionOptional|TestThirdScenerySurvivesCheckpointAndForecastRestorationOptional|TestTwoPlayerPoolHasOneCapacityAndProtectsInactiveLists|TestTwoPlayerForecastOwnsPhysicalStorageForEitherActivePlayer|TestNextStageRetainsOneArenaForBothSavedGames)$' \
  -count=1 -v
```

The native references and helpers remain excluded locally. Ordinary capacity and
forecast-isolation tests also run without original resources. The latest complete
original-resource engine, artwork and merchant suites pass in 44.182, 0.792 and
1.024 seconds respectively. Focused race checks pass for the transition,
equipment, pose, persistent-scenery, second-level guard and forecast-isolation
regressions. The desktop game compiles successfully.

Bounded graphics/frontend checks pass in 1.664 seconds for normal menu/READY/
gameplay/pause, two-player collision turn changes, and the arranged fifth-stage
ending/next-round boundary. The last fixture assigns stage completion and does
not constitute a campaign playthrough.

The explicit-start pilot completes the first two stages through real guardians,
rewards and merchants, reaching third-level READY with score 100,910, full shield,
three ships and both continue credits. Its second-level safety forecast now also
covers the late corridor and both final flanks. Unsafe final dodges use the
actual firing lane rather than a forecast endpoint blocked by the rock.

The current three-level frontend regression fails later in the third corridor,
at frame 9,159 and camera 714. The earlier four/five-stage recorded admission
profiles also predate the corrected transition behavior. No current complete
three-level or five-level automatic journey is claimed. The configured public
tour remains three levels; its survival and reliable menu return need renewal.

## Fidelity boundaries

The original comparisons establish fresh-session layout and sparse completion
boundaries, including retained fields, list/free-stack order, equipment and ship
poses. They are arranged offline states, not an original full campaign. They do
not establish every crowded two-player combat interaction, real-time frame
pacing, audio continuity or integrated audiovisual equivalence. Independent
one-player native constructor, update and damage comparisons remain applicable
to their documented states.

## Android confirmation

Runtime `2f771a3` passes the complete default-intro five-stage reference on the
physical Pixel 10a in 145.94 seconds. The two-player collision/READY and final
ending/next-round checks pass in 1.31 and 1.36 seconds. The built APK is installed,
its digest matches the local build and cold launch succeeds. These are locked
device logic checks; they do not measure frame pacing or audio output.

That installed runtime predates the cross-level storage and position corrections
described above. Its passing campaign must not be used to certify the newer
source. The newer source has not replaced the installed APK.
