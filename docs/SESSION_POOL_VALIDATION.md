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

## Reproducible checks

With prepared runtime resources and the local original references:

```sh
GOWORK=off \
XENON2_RUNTIME_ASSET_DIR="$PWD/assets/runtime" \
XENON2_NATIVE_TRACE_DIR="$PWD/.local/analysis" \
go test ./internal/engine \
  -run '^(TestSessionPoolsMatchOriginalInitializersOptional|TestTwoPlayerPoolHasOneCapacityAndProtectsInactiveLists|TestTwoPlayerForecastOwnsPhysicalStorageForEitherActivePlayer|TestNextStageRetainsOneArenaForBothSavedGames)$' \
  -count=1 -v
```

The native references and helper remain excluded locally. Ordinary capacity and
forecast-isolation tests also run without original resources. The complete
original-resource engine, artwork and merchant suites pass in 38.770, 0.720 and
0.931 seconds respectively. The connected default-intro five-stage frontend
reference still completes the ending and next-round admission with score
284,750, two surviving ships and no continue spent. Its accelerated GPU run
passes in 42.60 seconds. Focused two-player collision/READY, final merchant/ending
and Android input-policy frontend checks pass in 1.529 seconds.
Focused race checks cover shared capacity, both forecast views and asynchronous
encounter preparation. Strengthened frontend assertions also check that ordinary
menu admission, collision turn changes and the second difficulty round retain
one physical reserve; that focused graphics run passes in 0.961 seconds.

## Fidelity boundaries

The original comparisons establish fresh-session allocation layout and physical
resource tags. Saturation, forecast isolation, shared stage loading and frontend
flow have separate Go regressions. These checks do not establish retained slot
residue or free-stack order across a complete original stage transition, every
crowded two-player combat interaction, real-time frame pacing or audio continuity.
The independent one-player native constructor, update and damage comparisons
remain applicable to their documented states.

## Android confirmation

Runtime `2f771a3` passes the complete default-intro five-stage reference on the
physical Pixel 10a in 145.94 seconds. The two-player collision/READY and final
ending/next-round checks pass in 1.31 and 1.36 seconds. The built APK is installed,
its digest matches the local build and cold launch succeeds. These are locked
device logic checks; they do not measure frame pacing or audio output.

## Remaining original stage-transition comparison

A subsequent offline probe executes original cleanup, equipment restoration,
player switching and level initialization at arranged completion boundaries.
All five transitions, with one and two players, retain known fractional-position
and counter values placed in a previously used free object record. File access,
presentation drawing and the stopped audio device are isolated. No allocator,
cleanup, equipment or level-actor routine is replaced.

The one-player sparse boundaries also expose physical allocation differences:

| Transition | Original free head | Original primary slot | Fresh Go primary slot |
| --- | ---: | ---: | ---: |
| 1 to 2 | 9 | 5 | 4 |
| 2 to 3 | 7 | 5 | 4 |
| 3 to 4 | 4 | 5 | 4 |
| 4 to 5 | 5 | 4 | 4 |
| 5 to next-round 1 | 6 | 4 | 4 |

This is a sparse completion-boundary fixture, not an original campaign
playthrough. It confirms that the original keeps physical storage across level
loads; the current Go stage loader still creates a fresh shared reserve.
Preserving its free-stack order, surviving records and reused named fields
through those loads remains unfinished. Fresh-session matching and successful
Go campaign progression must not be presented as proof of that behavior.
