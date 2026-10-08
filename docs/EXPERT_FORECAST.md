# Expert gameplay forecast

The demonstration controller aims to complete all five levels through ordinary
controls, using known terrain routes, enemy formations, carrier rewards and boss
phases. The current carried route still loses too much energy and does not yet
complete the campaign. More accurate individual movement helpers cannot predict
all interactions between births, firing, combat, terrain and shared actor slots.

## Isolated full-world lookahead

The typed `WorldForecast` state copy runs the same Go `World.Step` callbacks against a
forecast world. The live game remains unchanged. Immutable images, animation
and path descriptors and terrain coverage lookup masks may be shared. Mutable
terrain, the 159-slot pool, weapons, entities, controllers, RNG, encounter cursor,
reward cache and transition flags must be copied.

Actor identity needs a two-pass remap so leaders, articulated chains, guardian
arrays and marker groups retain their original relationships. Shared controller
pointers need typed memo tables. Per-instance actor parts and writable tile patch
slices must not alias the live game. The forecast coverage map and its level map
must reference the same copied storage. Weapon ID/context callbacks must be
rebuilt against the forecast world rather than retaining closures over the live
world.

The API is `WorldForecast.Load(*World) error`, followed by explicit PAL
ticks and `Advance(Input) (ForecastResult, error)`. A forecast stops at ship loss,
READY, merchant admission or stage completion; it must not silently cross a
frontend/session transition. Audio channel activity is an external input to
source effect arbitration and must not be represented by a live callback.

## Verification and performance

Exact-step comparisons must cover all five levels, ordinary/compound actors,
shared chains, weapons, terrain destruction and slot stealing. Tests must verify
that the live game is unchanged after loading, advancing and reusing the branch.
Small isolated fixtures are insufficient evidence for a connected victory.

Reusable branch storage should bound allocations. Measure copy plus six/24
future callbacks on dense source scenes, then on the Pixel. A budget of a few
milliseconds per decision is a design target, not an established device result.
Prepared routes and trigger timing remain necessary: a short risk horizon alone
can knowingly trade immediate damage for an easier later position.

## Current validation and pilot use

Ten original-resource scenes cover the openings and compound arenas of all five
levels. Up to 24 exact gameplay passes after explicit 3-PAL-tick scheduling match
the full semantic state, while the live game remains unchanged. A 31-tick palette
strobe, reload/unloaded rejection and actual lethal contact exercise timing and
lifecycle boundaries. Independent mutation tests cover actor parts, patches,
shared chains/markers, physical slots, maps, controllers and weapon callbacks.
Metadata identity is excluded from value comparison; mutable identity is checked
separately by the isolation assertions.

The practiced controller uses a six-pass guard in the verified third opening
window. It retains its prepared command unless the full callback simulation
predicts damage, then ranks ordinary movements by survival and remaining shield.
The guard advances the host-configured PAL cadence and does not enter a frontend.
The genuine carried route reaches checkpoints 4032 and 3408 without another ship
loss, retaining 39 and 31 shield respectively.

The initial implementation prioritized correct ownership over allocation reuse.
On the M4 Max, arena Load measures 17–41 microseconds and 91–147 KiB of allocations;
Load plus 6 passes measures 52–142 microseconds with 126–275 KiB. The tested safe-opening
guard measures 46 microseconds/133 KiB; searching nine alternatives costs more.
These are scene-specific measurements, not Pixel frame-rate guarantees. Reusable
branch storage and mobile measurements remain required before broadening use.

## Pixel measurement

An ARM64 test binary built from a2022de ran on the USB Pixel 10a (Android 17),
with the game stopped during measurement. All 169 local runtime exports were used;
no graphics loop or original executable ran. Load plus six real callbacks takes
2.249/1.377/1.222/1.236/0.865 ms for levels 1–5. The safe-opening guard measures
0.966 ms, 132,952 bytes and 447 allocations. The nine-alternative fallback is not
measured by that guard case, and rendered FPS remains unverified.

The longer rows take 7.223/4.442/3.563/2.996/2.980 ms. Levels 1/2/3/5 complete all 24
callbacks; level 4 reaches actual player death at 21 and intentionally stops. These
are single 100 ms source-scene samples, not general device budgets. Branch storage
reuse remains the next performance task.

## Reusable copy storage

Forecast loads now retain typed storage for world/pool/controller records, maps,
entity lists, actor parts, patches and cyclic links. Pointer-stable chunks prevent
arena growth from relocating referenced actors. Reload clears memo tables and
unused pointer tails, and assigns every source record before reconnecting links.
Self-State loading is an isolated no-op; overlapping shallow wrappers are staged
before arena reset. Separate forecasts retain independent storage.

Two reload cycles cover sparse/arena scenes across all five levels, predicted
births followed by frozen-source reload, cross-forecast copying, self/wrapper
loads, nil invalidation and a 40→1→40 cyclic actor graph with shrinking attachments.
Retained weapon callbacks still update only the current forecast. Existing
all-five parity and isolation pass under the race detector. Warm Load measures
about 3–8 microseconds and 386–4,358 bytes on M4 Max, compared with 17–41 microseconds
and 91–147 KiB initially. Future World.Step callbacks still allocate; this change
reduces copy pressure, not the whole tactical policy's cost.

## Third middle guardian

The boss controller compares nine ordinary movements over 36 real gameplay
passes, including the complete arm lunge and recovery. Each branch holds its
initial movement for three simulated passes, then follows the source-based
six-pass policy. Only its first command is applied to the live game. Survival
and remaining shield rank before signed eye health and firing alignment. Trigger
release and diving still suppress fire normally.

The complete-intro regression carries the ordinary inventory, damage, merchant
purchases and credits through the first two levels. It defeats both eyes of the
third middle guardian and collects all real exit drops at frame 2448/camera 2739,
retaining its admission ship and 19 shield points. The constructor also clears
the entire source emitter word for all 17 parts: preserving a reused slot's low
byte previously created an extra shot and changed the shared RNG stream.

The continuation proposal caches terrain coverage as row words; 18,980 comparisons
across the five source maps and a live patch match the original stencil test.
This cache only proposes commands. Every branch still executes actual mutable-map
collision through World.Step. With persistent proposal storage, a warm M4 Max
decision measures about 3.9 ms and 20,234 allocations. An earlier word-cache-only
Pixel sample measured 64.7 ms, down from 209.5 ms before that optimization. Those
versions differ: neither number establishes a mobile frame budget for the final
policy. CPU and allocation reduction remain necessary before calling the full
boss controller ready for smooth Android playback.

The next optimization tests terrain by aligned tile-row words rather than
individual pixels. It reads the current map on each call, so destruction and
direct tile writes remain visible immediately. Independent comparisons cover
305,600 alignment/edge cases and 22,440 original-map positions, plus native
collision and rewind traces. Weapon contexts reuse callbacks only when their
equipment pointer belongs to the current world; input and phase fields are
rebuilt. Forecast and copied-runtime contexts rebind to their new world. Tests
verify current RNG, sound routing, changed phase flags and source isolation.
Together these changes reduce the measured warm M4 Max boss decision to about
3.05 ms, 63.8 KB and 792 allocations. The full-intro boss victory is unchanged
at frame 2448 with all 19 admission shield points intact. Pixel measurements
remain separate from host measurements and from rendered frame pacing.

The combined implementation's Pixel sample measures 63.4 ms over three
decisions, 169,330 bytes and 807 allocations. A subsequent 20-decision CPU-profiled
sample measures 35.1 ms, 76,884 bytes and 794 allocations. Initialization and
device scheduling affect these short samples; they must not be compared as a
rendered FPS result. The device reports no active thermal cap, but idle frequency
readings do not establish its clocks during simulation. The profile identifies
the continuation policy and its terrain queries as major remaining costs.

The boss proposal can memoize exact stencil queries in a 256-row window using
20 KiB of known/solid bits. Every source-map change and window move invalidates
the corresponding memo. Queries outside the window use the unchanged row test.
This is private planning storage; gameplay collision still reads the current
terrain directly. Original-map mutation tests and the unchanged complete-intro
victory cover reuse and invalidation.

## Independent worker execution

Nine private forecast/policy instances evaluate the nine movements concurrently.
The shared evaluator retains all 36 callbacks, source PAL scheduling, candidate
order and score ties. The caller waits for every worker before reading results.
One-processor execution uses the serial path. A branch requiring stateful
post-defeat navigation triggers a serial re-evaluation in the original order;
its latency is not covered by the fast admission measurement.

Sixty-four consecutive source decisions match both serial commands and the
complete final-candidate snapshot, with no live-state changes under the race
detector. Tests also cover stateful continuation and one-processor fallback.
The full-intro boss victory remains frame 2448 with all 19 admission shield
points intact. A matched warm Pixel prototype comparison measures 19.76 ms with
three workers and 12.97 ms with nine, versus about 67 ms for the sequential
comparison. These are source-scene CPU samples; rendering, other fight phases
and serial fallback can still exceed a 60 Hz frame budget.

## Later-stage target roles

Terrain-rendered fifth-stage barrier posts remain damageable even without a
visible sprite; their linking band does not. A native-motion alignment helper
holds the legal reverse allowance while shooting and releases it when the
barrier breaks. With all original encounters and starting equipment, the first
barrier opens at frame 77 with three ships and all 39 shield points. This is a
standalone fifth-stage opening test, not a carried campaign result.

Fourth/fifth guardian aiming now follows the source damage roles and gates.
The fourth middle core waits for the tail and four satellites; satellite outer
armor lines are excluded. The fourth final core waits for both eyes. Fifth
middle mounts and the cannon are targets; fifth final core admission waits for
all eighteen outer parts. Tiled vulnerable parts do not require sprite visibility,
and retained wrecks are rejected. Original-scene tests invoke real destruction
callbacks to open the gates and compare roles with copied damage callbacks.
These role tests do not establish complete fourth/fifth boss victories or correct
firing lanes through intervening armor bands.

Guardian primary-fire selection now observes the actual first new volley inside
an isolated forecast, immediately before each ordinary point-hit callback. It
retains the original firing cadence, projectile ID, equipment slot and source
moving-list order. An armor hit remains a consumed shot even when an older
projectile subsequently destroys a satellite and unlocks a core in the same
pass. Looking only at the final snapshot would falsely credit that core hit.

The observer is absent from normal gameplay, cleared after the predicted pass
and dropped when loading another forecast. Observed/plain simulations match in
all five original arena scenes. Clear strips, blocked bands, actual double-shot
offsets, rear/side ownership and older-shot interference have native-callback
regressions. The runtime pilot reuses its private aim forecast. This helper
checks forward/double primary volleys in active fourth/fifth guardian scenes;
other scenes and weapon kinds retain their existing aiming path. Eighteen
future passes apply the selected movement first, then coast while holding fire.
These are explicit counterfactual controls, not a claim that every future pilot
turn or complete boss strategy is already optimal.

The six-pass exact guard also protects the post-merchant third corridor. Cannon
preparation derives firing lanes from each live source instance, selects the
nearest passed row and tries another reachable target when a neighboring gun is
behind solid terrain. Before the cannon-aim correction, the full-intro route
reached checkpoint 1696 with the same ship and credits and 27 shield after the
ordinary merchant repair. Source-anchored aiming now destroys the central cannon,
but the changed combat/RNG route reaches that checkpoint with only 15 shield.
The original 27-point reserve assertion was retained while correcting this
regression. Native-motion route execution now restores it and extends the
connected route through checkpoint 1152, as described below. Later cannon combat
remains under validation. Third-stage completion, levels
four and five, ending and near-lossless campaign play are not established by
these results. A diagnostic time limit must be reported separately from a genuine
lack of camera progress.

## Committed native route commands

A clear geometric path does not guarantee that a fast ship can follow its
corners with one continuously replanned direction. An observed eight-command
cycle matched the real motion and rewind state exactly but made no progress.
The short native search uses the same movement, inertia, camera and collision
routines to retain a complete command sequence to an intermediate waypoint.
It invalidates a commitment when actual motion or future coverage differs;
checking another cannon candidate does not erase a still-valid sequence.

The source encounter owns cannon admission. A redundant upper camera cutoff
previously dropped a living target during its required rearward U-turn. Removing
that cutoff retains the legal route without enlarging a numeric window.

The search has a 9,000-expansion/32-pass limit and reusable storage. Its optimistic
time bound includes pre-check vertical overshoot and every doubled reverse pass
after the thirty-fifth deviation. Tests cover 37,800 source movement bounds,
actual five-command corner and seven-command exact three-pixel alignment,
remaining-segment clearance and commitment invalidation. Exact alignment avoids
accepting a zero-command arrival that cannot enter the next narrow passage.

The real complete-intro frontend test passes on the locked Pixel using an
excluded logic-only runner: both original cannon checkpoints are reached with
the same ship and credits and 27 shield. Checkpoint 1152 is reached at frame
6473/camera 1151; the entire simulated intro/carried route takes 46.63 seconds.
This runner performs no drawing and does not establish visual frame pacing.
Subsequent edge and sequence validation advances the same carried route to the
real final guardian launch and victory, as described below. Campaign completion
and rendered mobile frame pacing remain unverified.

## Geometric edges and safety previews

Point search and cached routes now validate the complete ship stencil along
each edge. The original masked gap at 251/world805 to 254/world808 has clear
endpoints but a covered interior pixel; accepting its nodes alone returned an
unusable waypoint indefinitely. Tests cover that original edge and a changed
tile invalidating a previously clear segment without covering either endpoint.

The safety guard previews the retained native command sequence for its planned
branch, then idles after completion. Repeating the first direction for all six
passes could predict a collision that the actual turn avoids and replace the
required command with Up. Final commands are checked against their before-state,
and previewing never consumes the plan. Invalid plans retain ordinary protection;
the nine alternative held directions and opening policy are unchanged.

This retained-sequence preview also applies after the third final guardian
launches. A corrected Supernova collection produces a different actual approach:
at camera 206 the preparer still owns its ten-command route, but the old final
guard interpreted its next command as a six-pass hold. At camera 205 that false
preview replaced Down-Right with Up and led to damage. Using the owned sequence
keeps the original six-pass horizon and ordinary controls. An exact frontend
replay defeats the final guardian with all 35 admission shield; the connected
Pixel route also passes its unchanged third-stage reserve assertions. The
corrected shared random state requires a new fourth-opening strategy, so these
results do not establish completion of the remaining campaign.

The actual source tile-gun fixture distinguishes a harmful held direction from
the safe retained corner. Its old guard fails the new regression. On the locked
Pixel, the complete default-intro route reaches both cannon checkpoints with
27 shield and admits the first final worm at frame 9633/camera 207 with 19 shield.
The same carried ship and zero credits are preserved; no cheat or diagnostic
state is used. That full logical run takes 68.76 seconds and stops on launch.
The subsequent complete-intro regression below establishes final victory;
remaining stages and rendered frame pacing remain open.

## Exact horizontal bound under evaluation

The relaxed horizontal machine derives every transition from Player.Advance,
including signed drift, held movement and the two screen clamps. Reverse BFS
caches the exact travel time to each requested horizontal coordinate, with any
final inertia. All 3,302,559 state/target distance equations pass; the three-pixel
speed-two alignment requires seven commands, while the same displacement into
the right clamp requires one.

Using this bound in the route search reduces the Pixel alignment fixture from
52.68 ms/1,257 expansions to 4.01 ms/119 expansions. It also changes an equally
short chosen route, costing four additional shield points in the connected
checkpoint test. The integration therefore remains disabled; the helper is
available for further evaluation and the validated route is retained. Neither
sample is a general frame-pacing guarantee.

## Third final head and fan prediction

The active final arena now uses the six-pass exact safety guard and the cached
first-new-primary observer. The earlier controller compared future ship positions
only with current worm rectangles and omitted projectiles. A contact already
present at the start of the source player callback cannot be escaped by that
pass's movement. A linear head lead also missed curved-path firing windows.

Original checkpoint fixtures distinguish a real useful primary from a linear
false positive. Ordinary point callbacks confirm both outcomes within the same
eighteen-pass horizon. A native fan case retains full shield under the guard;
a second-path incoming-head case avoids three pre-movement contacts and retains
its source-earned 27 shield. Prediction leaves live state unchanged and the
scope is limited to a launched, undefeated final guardian at camera 208 or below.
The bounded regressions isolate the causes; the connected result follows below.

The clean permanent complete-intro three-stage frontend regression now passes
on the locked Pixel in 71.68 seconds, without a diagnostic source overlay. It
validates both third-stage merchants, the 27-point cannon reserves, all final
rewards and unchanged shield throughout the final fight, then enters stage four
with the same one ship, zero credits and repaired 39 shield. The diagnostic
fight uses paths 61/65/63/66 and reduces shared health from 80 to zero at frame
10253, retaining all 19 admission shield points. The real final merchant is
admitted at frame 10313 with no pending drops. Earlier first/two-stage recovery
still applies; this does not establish near-lossless play across all stages.

## Fifth guardian laser hazards

Movement scoring now includes projectile-list laser columns, including newborn
columns whose collision rectangle has not yet been published. A private copy
uses the original growth, signed travel and expiry callbacks. As with ordinary
shot projection, this short preview holds the current scroll delta; it does not
predict subsequent camera reversals or contact removing a column.

Column contact uses the ship position and bank from before that pass's movement.
World.Step publishes this prefix before moving the ship, and the later projectile
callback retains it. Testing against the new ship position would allow a turn
one pass too late. Other moving-actor reaction margins remain unchanged.

Eight original projectile-phase scenes cover both travel directions and scroll
deltas -2, 0, 1 and 2 through growth and expiration. A separate eight-pass scene
with two ordinary speed upgrades compares held controls with replanned tactical
movement: held controls lose six shield; the tactical path retains all 39.
Prediction leaves the live pool, controller and random stream unchanged. These
isolated tests do not establish a fifth-stage boss victory.

## Mounted weapon rectangle queries

`WorldForecast.AdvanceWeaponObserved` can report both point and rectangle queries
immediately before their native hit callbacks. The existing point-only method
retains its behavior. Logical cannon/laser equipment ownership is separate from
physical owner fields, which still control their original update and slot state.
Other rectangle families report unknown logical ownership until supported.

The optional observer is absent from live gameplay, cleared after each observed
pass and discarded on forecast reload, including cross-forecast copies. The
read-only usefulness predicate follows moving-list order, shared groups, native
guardian damage eligibility and laser absorption. A satellite border or an armor
band does not become a valid target merely because a wider rectangle also covers
a vulnerable region behind it.

Isolated original-resource fifth-middle fixtures initialize a hypothetical left
cannon or laser. Their offset shots reduce core health from 200 to 198 and 197
respectively while the basic frontal ray misses. Observed and unobserved source
steps produce identical state, and the live source world remains unchanged.
These are weapon-capability proofs, not the captured campaign equipment or a
connected boss victory. The live pilot has not yet adopted rectangle aiming.

## Exact final firing-lane alignment

A narrow terminal solver handles a residual horizontal offset of at most six
pixels at the original upper ship bound. It uses the source horizontal transition
graph to construct release/braking commands, pads only the required forward
camera time, and validates every command with the complete motion and terrain
forecast. It preserves the existing search budget, route ordering and geometric
tolerance; it does not enable a different A* heuristic.

The captured third-corridor pose (255,16), camera 1836, reaches the exact firing
target (256,1832) in 20 ordinary passes. Full source steps reproduce every retained
state without terrain contact or rewind. Blocked and unsupported poses retain
the existing planner, and guardian worker caches remain independently owned.
Original-resource movement tests pass under the race detector, and the full
engine suite passes with this solver alone. A connected experimental route
destroys the previously blocking 1696 cannon and advances to subsequent guns
with 35 shield; this is not yet proof of completing that route or the campaign.

## Avoiding an unused terrain query

Terrain rewind reads its contact argument only while its timer is negative.
The motion forecast now skips the initial stencil query when that argument
cannot be used, retaining the later movement and camera contact checks.
A 360-case comparison covers clear and covered poses on all five original maps,
timers −17/−1/0/1 and all nine controls. Complete forecast values, including every
history entry, match the earlier implementation. Three native search scenes
retain identical nodes, queue, visited states, command sequences and results,
including an exhausted 9000-node search. Focused resource tests and race checks
pass. Single M4 Max samples reduce the rear-corner benchmark 54.37→35.43 µs
and the lane 2.306→1.959 ms; these are not Pixel frame-rate measurements.

The same-binary Pixel 10a comparison confirms complete search parity on all three
poses. Single 100 ms samples measure rear-corner 1.211→0.907 ms, lane 57.50→45.48 ms,
and exhausted 9000-node search 403.80→320.81 ms. The costly search still exceeds a
frame budget; this modest exact optimization does not make the whole controller
ready for smooth mobile playback. No drawing was performed.

The combined early-stage candidate also now checks its final guarded gun pose
before retaining a level-one Fire command. It vetoes an unusable shot without
advancing the burst state machine twice. The full original-resource engine suite
passes, and its genuine first-two-stage result remains unchanged on Pixel.
A strict third-stage run ends near the eight-minute limit without a completion
status or test verdict, so downstream campaign validation remains incomplete.
The candidate remains excluded from the default controller.

## Compact clear-route search states

Clear supported third/fourth-stage searches retain only the full player and
camera values in their search nodes, reducing a node from 728 to 136 bytes.
The original full forecast still advances each successor. Accepted nodes always
retain rewind timer zero, so their histories are never read by that search.
Selected commands are replayed from the actual complete start state; every
replayed player/camera value must match its parent chain before publishing the
full retained histories. Unsupported, covered or rewind starts use the unchanged
full search, as does any replay mismatch.

Independent reconstruction compares every node and history, heap order, visited
state, expansion count, commands and retained states. Original 5/7-command routes,
20-command terminal alignment, fourth-fork legs, exhausted 9000-node search and
rewind fallbacks pass. Worker buffers remain independently owned. The full
engine/native-resource suite and focused race checks pass.

The same-binary Pixel comparison retains exact parity. Single 100 ms samples reduce
the lane 44.70→38.21 ms and exhausted 9000-node search 321.55→259.97 ms. This improves
search cost and memory pressure while retaining the policy; it still does not
establish smooth rendering or a complete connected expert campaign.

## Improved early-route carried outcome

A lightweight Pixel frontend diagnostic now carries the improved early-stage
policy through the genuine third final merchant into level four. Both first
stages still consume two ships total and preserve both continues. The third
stage keeps its admission ship and both credits, reaches the cannon checkpoints
with 35 shield, defeats the native guardians and collects their exit rewards.
Its final approach is weaker than the retained controller: only 7 shield remains
at final admission, and the fight loses 4 more before repair. It therefore does
not satisfy the existing 19-shield/unchanged-final-shield regression.

The bounded run takes 81.19 s without drawing. Repeated unsuccessful native
searches remain costly. Completing this diagnostic does not establish the
strict controller criteria or either of the last two stages, so the improved
early policy remains excluded pending further correction.

## Preserve admitted commands near firing positions

The third-corridor firing hold previously appended Down after the native planner
had admitted a command. A clear original right 1328 cannon scene demonstrates
the error: at ship (255,176), camera 1306, the planned Up becomes Up+Down, leaving
the ship at 176 instead of 171 and invalidating both the guard preview and next
retained command. The corridor now returns an admitted command unchanged; the
Down hold remains available only for uncommitted fallback movement.

The regression fails before and passes after, including the actual World.Step
position, guard preview and second-command retention. Existing fallback hold,
full resource/source suite and race checks pass. This corrects execution of an
approved maneuver; it does not claim improved complete-campaign survival.

## Search-local exact terrain memo

Each compact search now memoizes the original ship-stencil query by its exact
horizontal/world-vertical position in a bounded private window. The map and
stencil cannot change during that synchronous search. The memo resets before
every search, and positions outside its window use the original query.
Successful-route replay and the full fallback remain uncached.

Tests compare every node, queue, visited state, command and retained history
with the uncached implementation. They cover all five maps and nine controls,
an actual tile patch between reused searches, out-of-window queries and
independent guardian worker ownership. Full engine/source and race checks pass.
A same-binary Pixel sample retains exact parity and reduces the 1257-node lane
33.89→17.27 ms and exhausted 9000-node search 265.38→122.12 ms. These are single
100 ms samples; the costly case still exceeds a 60 Hz frame budget.

## Fourth-middle late reverse-bound prediction

The live fourth-middle core renews the camera's maximum to 2480 during the
moving-actor phase, after player movement and before final scrolling. The
motion forecast previously omitted that update. It now applies it at the same
phase only while the actual core actor is active and its part enabled.
Six original-resource Down/idle cases compare complete player, camera and
rewind state with World.Step for active, disabled and absent cores. Active
cases fail before the correction and pass afterward; other scopes stay
unchanged. Full engine/source and focused race checks pass. This corrects
prediction, separately from the still-unverified fourth-middle rear strategy.

## Prepare the late reinforced formation

The third-stage controller now prepares a clear position before the original
camera 576 seven-part strong formation on path55. Eligibility requires the last
640 cannon's real destroyed patch, its passed encounter cursor, and a clear
source stencil. It stops after that formation clears. Movement is proposed
before the single firing decision and still passes through the exact guard.

The original-resource comparison admits all seven parts and retains 35 shield,
versus 19 without preparation; both branches survive 32 more passes with rewind
zero. A genuine complete-intro Pixel comparison also avoids the observed
16-point contact and reaches the final guardian with 23 shield instead of 7.
It completes level three with the same ship and two continues, then repairs
to 39 for level four. The final fight still loses energy, so the unchanged-shield
quality check and the full five-stage expert target remain unmet.

## Current source-event and damage evidence

The improved route's remaining third-final damage is now localized to two
ordinary projectiles at frames 9243 and 9244. Their copied source steps match
the live game exactly. Every held six-pass direction already loses shield on
its first pass at those states: the player reaches the arena's left edge in a
terrain rewind, and its pre-movement collision prefix cannot escape either hit.
A source-terrain route toward the native final checkpoint is under evaluation
to prepare that entry earlier. No collision or damage rule is changed.

The fourth-middle rearward terminal probe exposes a satellite, but its normal
tactical handoff finds no useful firing position. Center placements fall within
the original wall; nearby cannon-offset probes are not established safe. Those
strategies remain excluded pending an actual carried-state comparison.

## Native checkpoint preparation before the third final

After the final cannon and reinforced formation are finished, the controller
can prepare the source final restart point (152, world 352) before the first
fan reaches the left terrain fork. The point comes from the original checkpoint
X/camera plus the native restart ship height. It uses existing full-stencil
navigation and retained source commands, before one selective-firing decision.
The exact guard still owns avoidance. The helper requires the real destroyed
cannon patch and passed encounter gates, and does not move the camera or ship
directly.

The original-resource route retains its initial shield and zero rewind through
160 ordinary passes and the native first fan. Scope checks reject nil, changed
patch, unvisited formation and outside-window states without mutating them.
The isolated comparison does not claim better shield than its simplified
baseline. The genuine connected comparison removes both observed 4-point hits,
and the unchanged strict first-three-level regression passes with 23 final-entry
shield, zero final damage and both continues retained. Fourth-route regression
caused by the subsequent changed RNG remains under correction.

## Default early-stage controller

The previously excluded early-stage guard and second-arena changes are now
integrated with the validated third-stage preparation. The arena proposal uses
actual published pre-movement contact rectangles, includes hidden defense nodes,
retains a legal upper passage to the opposite node and charges one moving
contact per pass, matching the source's first-contact callback. Projectile risk
remains separate. Native collisions, gates, health, firing cadence and random
state are unchanged. A guarded level-one movement rechecks its final gun pose
without running the firing-burst state machine twice.

The connected strict regression verifies the real first-three-stage campaign
with the original reserve requirements. Level one is completed without ship
loss; total first-two-stage losses fall from eight to two. The same admission
ship and both continues survive stage three, including the unchanged-shield
final fight. The subsequent narrow fourth-corner continuation reaches its
middle guardian with seven shield and then loses the ship. This improves the
default controller while leaving the full expert campaign incomplete.

## Complete a retained fork maneuver before releasing it

The fourth forest helper previously checked its coarse finish condition before
consulting an already admitted native route. A source-map regression crosses
that condition after the first of three commands, while two matching commands
remain. The old helper abandons them; the guard still has their preview.
The helper now consumes a valid retained route before deciding whether to
release the completed fork. Eligibility and the finish condition are unchanged.

The regression fails before and passes after, with exact source motion/history
through all three commands and unchanged 39 shield. Existing corridor, corner
and guard regressions pass under the race detector. This fixes control/preview
consistency; its effect on the five observed approach impacts still needs the
genuine connected comparison.
