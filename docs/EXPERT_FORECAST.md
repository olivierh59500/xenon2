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
