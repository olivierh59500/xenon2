# Game implementation coverage

The conversion uses an independent Go simulation and Ebitengine renderer.
This inventory distinguishes integrated game behavior from extracted artwork
and isolated controller comparisons. A resource export does not by itself
establish that the corresponding encounter is playable.

The desktop and Android expert demonstration is limited to levels one through
three. Its completed third-level final merchant returns to the original menu;
sixty idle seconds start another tour. Tests cover both automatic idle admission
and an explicit demo launch, the real third guardian/rewards/merchant sequence,
menu return and a fresh ordinary level-one READY admission.

The current complete-intro and idle-start expert tours finish all three supported
demonstration stages with two ships and two continue credits intact. Each loses
one ship in level two and keeps its carried ships throughout level three. Both use real guardian deaths,
rewards and merchants before returning to the menu. The measured scorecard is
recorded in [EXPERT_FORECAST.md](EXPERT_FORECAST.md).

Later-stage reference fixtures remain available but use an explicit test-only
tour limit. Their earlier carried loadouts and random states are capability
records, not current whole-campaign proof. In those recorded routes the fourth
stage reaches its real final merchant; the fifth reaches its final guardian.
The complete final fight and integrated Amiga playthrough comparison remain
unfinished. Ordinary demonstration launches do not enter level four.

## Fixed encounters and scripted arenas

| Level | Integrated families | Remaining work |
| --- | --- | --- |
| 1 | Three terrain cannon types, bouncing attackers, five-stream middle arena, final body and articulated chain | Integrated scene and complete playthrough comparisons |
| 2 | Terrain cannon, bouncing attackers, hatches, both pod emitters and their creatures, middle defense waves/nodes, final body and transforming minions | Integrated scene and complete playthrough comparisons |
| 3 | Terrain cannons, sweeping attacker, turning projectile, extending chain, tile-edge crawler, middle flying guardian, final worm and conditional scenery | Integrated scene and complete playthrough comparisons |
| 4 | Terrain cannon, extending beam with shaft and animated tip, nest crawler, falling hatches and guns, capsule offspring, twenty-part middle guardian and nineteen-part final guardian | Integrated scene and complete playthrough comparisons |
| 5 | Terrain cannon, destructible barrier, aiming and radial turrets, persistent turrets, ten-member radial formations, vertical attacker, aiming projectile, both compound guardians, growing laser columns and mouth creatures | Integrated scene and complete playthrough comparisons |

The supplied encounter streams contain respectively 5, 6, 7, 6 and 10 selectors,
including the checkpoint selector in each level. Every selector requires an
explicit semantic implementation; unknown selectors must not be mistaken for
empty encounters.

All fixed selector families have explicit implementations. The integration
audit still checks their shared state, drawing, contacts and lifetime boundaries;
selector coverage alone does not establish a complete playable conversion.

The terrain-cannon audit executes the original constructors for all 71 actual
placements across the five stages before comparing 11,360 updates and 339 shots.
It checks orientation tags, origins, health, collision, animated tile writes,
firing positions, headings, speed and RNG. This exposed reversed orientation tags
and animation tables in the first stage's opening plants: record variant zero
must fire right from the left wall, and variant one must fire left from the right
wall. Earlier isolated callback traces supplied their tags directly and did not
verify this record-to-constructor mapping. A permanent regression also verifies
that both orientations of all three first-stage cannon families emit inward.

## Shared systems

Movement, path commands, deterministic randomness, equipment, projectiles,
terrain coverage, cash and carrier rewards, checkpoint recovery, alternating
players and the shared 159-slot allocator have source comparisons. Terrain and
sprite transparency use their distinct original coverage formats.

Pending terrain rewind now continues across dive admission, including history
restoration and crushing; diving suppresses new contacts only. The world and
movement forecast agree with 648 original ship/scroll boundaries on the first
three maps. A normal-input regression requests a dive on the wall-contact pass
and verifies the saved position is restored on the following pass.

Invulnerability uses its original animated aura in the projectile list. Its
counter freezes during ship materialization and successive pickups extend the
existing aura. The source meters for invulnerability, diving and Nashwan use the
original ten images and include their final zero-count frame.

The session has explicit intermediate-shop and stage-completion boundaries.
The two-player stage gate preserves each saved game; level five increases
difficulty and loops to level one. Frontend loading, shop and ending routes are
connected. Live scene comparisons remain separate from simulation tests.

Stage scripts execute before actor updates. New stage actors therefore move in
their construction pass; encounter-table actors are created afterward and move
on the following pass. Enemy counting includes pending-removal entries until
their owning list releases them. Damage and rendering use physical list order.

All 601 actual moving-wave records now have original-constructor comparisons
with fresh and deliberately reused slots. The 1,202 cases include 3,048 actor
births and the original empty wave. Initial image/collision, physical allocation
order, positions, delays, movement budgets, health, firing accumulators, body
links, shared RNG and bonus buckets agree. The comparison found missing immediate
publication of constructor fields and an early heading-image selection in the
third level. Both are corrected. An ordinary bullet reclaiming a wave slot before
its first movement now inherits the constructor's cleared coordinate fractions.
These checks cover construction with available capacity; they do not establish
initial unassigned Amiga RAM contents or every crowded-pool interaction.

## Validation boundaries

Private source comparisons run only when the local decoded disk resources and
trace directories are supplied. Those resources and traces remain excluded
from Git. Ordinary tests verify the independent logic without an emulator.

The build updates at 60 Hz and uses a separate gameplay clock. Ebitengine may
draw at the monitor refresh rate; this does not accelerate gameplay.
Integrated Amiga comparisons and a complete five-level run remain required.
The current build is in development and is not yet a complete conversion.

Checkpoint restoration across alternating turns, the second guardian's
crowded-scene direction, retained hatch/pod state, and death-image attachment
centers now have explicit implementations and source comparisons. Accepted
continue admission follows the original surviving-player route.

Resource checks exercise every fixed record over 96 callback passes and every
moving wave over 64 passes, verifying animation images, emitted shots, body
patches and overlays against the exported atlases. They do not replace an actual
renderer comparison or a complete game played through its normal controls.

Actual Ebitengine GPU tests render all ten middle/final arena families and the
fourth-level extending beam. These fixtures use original stage/encounter births
and production drawing; their arrangements isolate composition rather than
establishing complete-stage victories. Pixel comparisons
cover aura alpha, terrain materialization, body tile placement and damage flash,
moving/effect layer order, all five palette strobes and Shades, and the 48-point
background starfield. A shader source-size panic and double source-origin
adjustment were found through those tests and corrected. Desktop menu, attract,
shop and five level captures have been inspected.

A real fifteen-second desktop run measured approximately 60 updates per second
and 120 draw calls per second on its 120 Hz display after warm-up. That run does
not establish dense-combat performance or a complete playthrough. The current
complete-intro logical regression completes the first three stages with their
real merchants, guardian destruction and exit rewards. Its limits are detailed
below; a complete campaign and rendered mobile performance remain unverified.

Finite common explosions and converted fourth-stage pods now retire and release
their physical slots. Animation-only effects preserve untouched slot fields;
third terrain crawlers and compound cannons publish their native physical state.
Resource regressions cover actual slot release/reuse. Fifth core hits flash
their owning tiled body, and the middle body retains its last normal muzzle
table during flash callbacks. A real basic-gun hit also passes the production
GPU pixel check for the final body.

The practiced presentation and the development reference controller are distinct.
The current public three-level tour has connected near-lossless observations:
one second-stage ship loss for each admission path, with no third-stage loss.
Neither spends a continue. These results do not establish every starting state,
maximum enemy/bonus clearance or complete five-level fidelity.

The latest first-stage presentation MP4 is 381.53 seconds and includes the production
intro, known left junction, both genuine merchants, final guardian destruction
and collected exit drops, with all three starting ships remaining. It ends before
playing level two. The exported file contains 22,892 1280 × 800 frames at 60 FPS,
stereo 44.1 kHz AAC and scene chapters. Video frames were inspected during the
opening, middle arena, final guardian and exit. The desktop application/GPU
suite passes in 48.961 seconds with the original runtime exports present.

Fifth guardian/column creation and updates now preserve the physical emitter,
phase, direction and untouched fields written or retained by the native callbacks.
Actual sold-cannon/expired-column slot reuse and fresh/strong middle contact cases
verify the effects on first firing and later weapon admission.

Third chains now retain their physical marker/body fields across natural expiry
and slot reuse. Actual diagonal-bullet lifecycles establish the retained fractions
and the next projectile's trajectory, while all 6,400 original chain states still
match. The second-defense presentation model also forecasts current staggered
members in source callback order: 1,152 positions, images and collision prefixes
match actual world steps. It does not forecast new scheduler births or combat
deaths; using it in an arena policy still requires a connected victory proof.

Second-defense nodes now retain tile-unit coordinates through normal updates and
damage flash, publish their source phase/index/health, and preserve unused slot
fractions, rewards and strength. Post-bind initialization retains the original
body and descending node allocation identities. An actual open-phase node,
ordinary lethal hit and same-slot flamer reuse verify the later native effect
cleanup; the original generic storage fails that regression. Constructor and
live node comparisons pass alongside 1,152 native node and 16,890 defense-wave
passes. These are source-state fixtures, not additional campaign progress.

The expert opening now reaches the first third-stage checkpoint with its carried
ship and credits intact, using changing directions within a copied source horizon.
A separate reference continuation destroys the terrain cannon that blocks the
later passage by ordinary shots. The connected expert now also defeats both eyes
of the third middle guardian, collects its real drops and reaches the middle
merchant with its carried ship intact. The
36-pass boss forecast covers the arm's extension and recovery, but its mobile
cost does not establish a smooth-frame guarantee. Private concurrent branches
preserve serial decisions, and native command sequences now clear the narrow
later routes. The current Pixel frontend reaches checkpoints 1696 and 1152 with
35 shield, then the final third guardian with 35 shield, without another ship
loss. Its logical runner works on the locked device; this does not verify drawing.
The clean complete-intro regression now also defeats the final third guardian
without shield loss, collects its exit rewards, visits the real final merchant
and enters stage four with the same ship and repaired shield. The later
fourth-stage completion is detailed below. Stage five and low-loss play across
the whole campaign remain open. Automatic title
admission after sixty idle seconds is shared by the desktop and Android frontend,
with immediate manual takeover.

The fourth opening uses the exact six-pass safety guard, terminal moving
contact checks and practiced native command sequences through the forest. Before
the Supernova callback correction, a full default-intro Pixel run reached its
middle guardian with 23 shield and destroyed the native tail without losing
shield, keeping the same ship and both continues. The corrected third-stage
history changes fourth-stage admission. Retaining a verified six-command
fallback now reaches the middle guardian with 27 shield and destroys its tail
without shield loss in the complete-intro desktop frontend. The forest minimum
is 11 before a genuine health pickup. The old entry remains a recorded capability
fixture, not current campaign proof.
The current controller also destroys the remaining satellites and core, reaches
the real middle merchant, then defeats the final guardian and enters level five;
the current completion evidence is recorded below. See the
[fourth opening validation](FOURTH_OPENING_VALIDATION.md) for earlier comparisons. Fifth laser columns
are included in movement scoring with their native growth and pre-movement ship
prefix; isolated source tests establish avoidance, not a complete fifth boss.

The remaining live checks are:

- Play all five stages from the normal menu, including both shop boundaries,
  ending and the next difficulty loop.
- Exercise actual deaths, score entry, accepted/refused continues and alternating
  two-player checkpoints.
- Compare integrated artwork, palette fades, sound transitions and elapsed
  cadence with the Amiga reference.
- Check 60 Hz display smoothness and continuous audio during dense combat.

Frontend boundary checks now cover collision death, score initials, accepted and
unanswered continues, real merchant quote/purchase controls, same-stage reload
and the two-player final ending gate. Explicit boundary fixtures do not count
as victories. GPU regressions verify that outgoing fades uncover the new
shop caption and ending dot, and that READY uses the correct player's director
on its first draw. A bounded native A500 capture also establishes that the
credit sequence's elapsed cadence differs from the maximum source clock; full
gameplay timing and soundtrack comparisons remain open.

An arranged fifth-final integration fixture now retains the native defense/core
health and uses source-factory in-flight point bullets through actual World.Step
callbacks. It destroys all eighteen defenses and the core, drains all twenty
real exit coins, reaches the engine merchant boundary, buys Nashwan with emitted
cash and starts the next difficulty loop. It checks the single victory credit,
wallet reset, released guardian slots, independent new pools, regular-loadout
restoration, retained temporary timer and doubled first-wave health. The fixture
does not assign completion/drop counters or call direct damage to win.
Its arranged inventory/projectiles are explicit; it is not earned fifth-stage
play or a frontend/graphics victory. Focused native and race checks pass.

Fifth guardian event traces now independently check 2,700 complete frames,
1,007 ordinary mount shots and 1,513 final queued sound requests. Per-component
shot counters are summed, while signed sound queues are compared after the
last component. Sampled requests such as 0x84 remain distinct from -1. No event
mismatch was found. The historical trace column named `lasers` has ambiguous
factory-hook semantics and does not establish growing-column emission coverage;
this new comparison explicitly excludes that field.

An original-resource fourth-middle boundary fixture now crosses the real fixed
selector and verifies projectile callbacks through all five defenses and the
175-health core. A consumed shot cannot damage the locked core. The actual
lethal callback clears the original terrain corridor, releases guardian slots
and creates ten ordinary coins. Their collection/expiry opens only the
same-stage merchant; ResumeShop preserves the living World, and ordinary
scrolling continues in stage four. No completion/drop/health fields are assigned
to win. Inventory, edge position and in-flight factory bullets are explicit
arrangements, so this validates integration rather than an earned boss victory.
Focused native/race checks and the complete engine/source suite pass.

A fourth-final boundary fixture now retains both 50-health eyes and the locked
100-health core. Native-factory point bullets verify the absent locked collider,
close each eye through its real callback while retaining its artwork/actor, and
defeat the exposed core. Twenty real coins drain through normal lifetimes before
the final merchant admits stage five. Gear, shield, lives, credits and RNG carry
forward without an early fifth-victory credit or difficulty increase. The clear
initial pose and in-flight bullets are explicit arrangements; no guardian health
or completion/drop field is assigned. This proves connected engine boundaries,
not an earned fourth-stage win. Native/race and full source suites pass.

A fifth-middle boundary fixture retains the four 40-health mounts and the
200-health narrow core. An ordinary point bullet can damage that core beside
living mounts; the middle guardian has no final-style armored collision band
around it. A bullet just outside the core therefore remains active without
damage or flash. Real projectile callbacks destroy the mounts/core, clear the
original terrain rectangle and release their physical slots. Ten native coins
drain before the same-stage merchant returns to the same living World, with no
fifth-victory credit, stage-completion flag or difficulty increase. The explicit
on-screen arena and in-flight bullet packet isolate this boundary from firing
cadence and navigation; they do not establish an earned campaign victory.
Focused original/race checks pass in 2.341 seconds; the complete original-resource
engine suite, including all retained native traces, passes in 17.977 seconds.

## Blocking screen-clear collection

An integrated original-resource test exposed a screen-clear ordering defect:
after collecting reward 18, an older enemy shot still advanced and damaged the
player in the same projectile traversal. The source callback instead blocks
for 31 PAL ticks, then marks enemy shots dead before returning to that traversal.

The Go world now retains its original input and physical saved-next cursor while
the palette strobe runs. At tick 31, it performs the native sweep and resumes
the remaining callbacks of the same frame. Friendly shots advance once; player,
moving actors, equipment and frame initialization are not repeated. Synthetic
unbound diagnostic lists retain their own cursors, and forecasts clone both
forms of pending continuation without sharing live state.

The demo director also waits for that frame to finish before sampling its next
controls. Manual takeover remains immediate. Source tests cover all five levels,
consecutive pickups, original-input retention, timer/sound ordering and clone
isolation. The original failing collection case now retains 39 shield, with the
enemy shot stationary during the flash and retired at its end. The complete
engine suite passes in 19.815 seconds; focused source/race checks pass in 2.674
seconds. The frontend/GPU suite passes in 106.885 seconds. Its focused sampling
test fails with the old director and passes with the correction, including
manual takeover and per-PAL palette publication. A first-level recording replay
still completes with three ships and
39 shield in 381.55 seconds and collects no screen-clear bonus, so the existing
381.53-second video does not require regeneration for this fix.

The real third-stage screen-clear collection at frame 3391 preserves 35 shield
through both later checkpoints and final-guardian admission. Its corrected
callback order changes the subsequent shared random state and approach. An
exact third-stage frontend replay initially exposed four shield lost at frame
9271: the guard previewed a retained mixed-direction route as six passes holding
one direction after the final guardian launched. It now previews the existing
route's actual commands in that encounter, as it already does in the preceding
corridor. No route goal, lookahead length, firing rule or reserve is changed.
The same replay reaches the genuine final merchant at frame 9793 with all 35
shield, one ship and two continue credits. Its launch pose and RNG match the
failing replay exactly. A fresh Pixel frontend replay also passes every unchanged
first-three-stage assertion and reaches level four with 39 shield, one ship and
two continue credits. The corrected callback history produces a new fourth-stage
READY random state, 1818979822/680038254. The existing fourth-opening strategy
then fails at frame 805 in the forest. The earlier tail-destruction proof remains
a valid original-resource capability test with its recorded entry state, but no
longer proves the currently connected route. Fourth-stage pilot adaptation
has since improved through the verified-maneuver controller described below;
no campaign-wide success or new Pixel installation is claimed.

An exact fourth-stage frontend reconstruction matches all six observed protection
losses and the first death at frame 805, including the current carried inventory,
score, pose and random stream. Passive guard traces classify two earlier traps:
at frame 325 every six-pass held action either loses protection or ends in an
already published moving-actor contact; by frame 330 all next commands take the
same projectile hit. At frame 360 the only held action preserving protection
ends in a body contact, and the real contact occurs at frame 363. Neither case
is a forecast-versus-runtime collision discrepancy. A separate native terrain
route from frame 318 reaches its ordinary waypoint with exact motion and no
terrain rewind, but takes eight protection points from enemies. It is rejected
as a pilot improvement; terrain planning alone does not solve these traps.

The current complete original-resource engine suite passes in 21.970 seconds.
The frontend controls checks pass in 16.163 seconds, including automatic
title admission, keyboard/touch takeover, blocking-frame control sampling and the
one-PAL resume boundary. The sixth-command forecast now honors retained native
routes in the third final encounter; unowned and foreign-owner routes retain
their existing behavior.

## Verified fourth-opening maneuvers

The controller now retains a changed fallback only after all six ordinary source
passes remain safe. Every remaining command is rechecked from the actual world
before use; changed state, terrain, damage, rewind or lifecycle boundaries reject
the commitment. One-shot Dive requests are not retained. Safe planned moves,
other levels and the admitted middle guardian keep their existing controllers.

The complete default-intro desktop frontend passes all unchanged first-three-stage
and fourth-tail assertions in 13.131 seconds. Native middle admission is frame
1953 with 27 shield; the real tail dies at frame 2542 with the same 27. The forest
minimum is 11 before an ordinary health pickup. A main-code replay after lifecycle
cleanup matches both endpoints exactly in 1.116 seconds. The permanent strategy
regression now uses this current recorded entry, retaining its original reserve
requirement and additionally asserting the exact 27-shield tail endpoint.
The complete original-resource engine suite passes in 22.379 seconds, the full
frontend/GPU suite in 113.250 seconds and focused race checks in 7.887 seconds.
Tests also change an existing projectile while leaving the entire compact key
unchanged: the remaining-callback replay still detects the danger and rejects
the commitment. Native shot creation, terrain obstruction, repeated sampling,
ownership, PAL cadence and lifecycle invalidation are covered.
The rest of the guardian and the complete five-stage expert route remain open.

## Fresh disk extraction

A fresh isolated import from the supplied supported ADF verifies all six packed
asset containers and the offline executable image, then runs every exporter.
All 169 generated runtime files match the live resource tree byte for byte;
the only live-only files are the tracked Go embedding wrapper and generated-data
notice. No executable bytes are published as runtime logic.

The newly generated tree also passes frontend checks for bundle/world loading,
normal menu/READY/gameplay/pause/attract, ordinary middle-shop purchase/reload,
and the fifth merchant/ending/next-stage boundary in 2.219 seconds. These are
resource-reproducibility and flow checks, not an earned five-level victory.

The complete package suite also passes against that freshly generated tree,
including the engine in 23.520 seconds and frontend/GPU tests in 122.821 seconds.
With local native audio references enabled, the main/menu comparisons cover
18,000 and 6,000 original music ticks on four voices; the shop comparison covers
8,100 ticks across all 27 effects. The effect-voice trace also passes. These
source-state comparisons do not replace an integrated audible Amiga comparison.

## Fourth-middle weapon positioning

Separate ordinary-control prototypes continue from the earlier recorded tail
endpoint with 23 shield and the earned Forward 1, Cannon 0 and Rear 0 loadout.
A camera-aware route travels around the guardian, reaches the upper screen edge
and defeats both upper satellites through 40 new native Rear-shot hits. It ends
at frame 3195 with all 23 shield. A subsequent route respects the complete ship
terrain stencil on the right flank and positions Cannon 0's actual first query
rectangle against the lower-right weak point. Ten new native two-damage Cannon
hits defeat that satellite at frame 3359, still with 23 shield. The lower-left
satellite retains 20 health and the core retains 175.

These tests compare each command against direct World.Step results and verify
that forecasts leave their source untouched. Focused race checks pass. They are
recorded-entry weapon/navigation capabilities, not a connected current campaign
victory or a production pilot upgrade. The lower-left strategy still loses
protection to the animated companion body and has not been promoted.

The current 27-shield entry has since produced a stronger ordinary-control
continuation. Native coast-and-release steering stops within the actual clear
left corridor before reversing the camera. Stable firing axes come from the
intersection of every satellite heading's collision rectangle. During alignment
and firing, the controller re-evaluates its six-pass choices every native pass
instead of committing to an approaching projectile lane.

Both upper satellites die at frame 3555 through 40 new Rear-shot hits, with all
27 shield intact. The lower-right target already has four health at the tail
boundary: seven ordinary Forward 1 hits and one Cannon hit inflict its earlier
16 damage during frames 1978–2026. The upper continuation does not cause that
collateral. Two further native Cannon hits destroy it at frame 3673, still with
27 shield. A receding version of the lower-left controller then applies ten new
ordinary primary hits and destroys the last outer target at frame 3983, retaining
all 27 shield, one ship and two continue credits. The core still has 175 health.
Each stage compares actual callbacks with direct World.Step and checks forecast
isolation; none assigns guardian health, completion, player resources or position
to win. The production integration also passes the complete default-intro Pixel
frontend route in 38.26 seconds, preserving every unchanged first-three-stage
assertion. All four outer targets are genuinely destroyed at frame 3983 with
27 shield, the same ship and both continue credits; the core remains at 175.
This is logical device validation without drawing. The production engine/source
suite, including the permanent new regressions, passes in 27.131 seconds. The
complete frontend/GPU suite passes in 101.924 seconds. Upper/right focused race
checks pass in 24.253 seconds; left checks pass in 69.168 seconds. Tests retain
native hit ownership, exact endpoints, repeated input sampling, world/frame
reset, lifecycle/resource scope and all 3,783 horizontal states at each alignment
target.

The exposed-core controller now times its approach and retreat from the native
head's Counter/Direction/Budget cycle. Four linked curve segments and the largest
companion/ship offsets give a conservative future floor; when that floor cannot
fit the arena's rear bound, the ordinary target becomes the clear left flank.
The same six-pass, nine-input native forecasts retain authority over collisions.

The core dies at frame 4496 through 73 new Forward 1 and 15 Cannon two-damage
hits. Its final native unsigned subtraction stores 0xffff, with Defeated true.
Ten real coins drain before the actual middle merchant at frame 4544, with all
27 shield, one ship, both continue credits and 1,050 cash. The complete-intro
Pixel frontend confirms this connected boundary in 29.75 seconds without drawing.
No guardian damage, health or drops are assigned by the controller. Original-resource
engine tests pass in 28.255 seconds and focused core race checks in 19.243 seconds.
The second half of level four, final guardian and earned fifth-stage route remain
unfinished.

The existing six-pass guard now also covers ordinary play after the fourth
middle merchant, stopping before the final arena. The former route died at frame
5014 because an original kind-5, tag-272 right-extending beam hit the stationary
ship for six shield on every callback. Its scene regression retains the native
factory, animation phase, rectangle and saved player bank: held input loses six,
while guarded input preserves all 39. The old guard fails this regression; focused
race checks pass in 2.558 seconds.

The complete-intro desktop frontend now reaches the genuine fourth final guardian
at frame 6560, camera 143, with 31 shield, the same ship and both continue credits.
Its real middle merchant repairs shield for 500; ordinary rewards raise the wallet
to 3,550 before final admission. The connected check passes in 14.794 seconds.
The final guardian is not yet defeated. A separate final-arena guard experiment
survives longer with 15 shield but does not win; it remains excluded.

A newer final-arena controller evaluates all candidate commands through complete
native World.Step callbacks instead of predicting moving rectangles separately.
It compares surviving shield first, then real remaining eye/core health and
alignment. The original eye locks, closed-eye colliders, armored arms, projectile
lifetimes and weapon callbacks remain intact. The ordinary fight defeats the
final guardian at frame 7366 with 27 shield, down from 31 on admission, without
consuming its carried ship or continue credits.

Twenty real coins drain before the final merchant at frame 7424 with 5,050 cash.
Normal repair and purchases then admit level five with 39 shield, Forward 1,
Cannon 0 and Rear 1, one ship and two continue credits. This earlier recorded-fourth
frontend check passes in 2.376 seconds. The production controller also passes
the full default-intro route, including the unchanged first-three-stage assertions,
in 15.005 seconds on desktop and 206.87 seconds in the Pixel logic runner. The
permanent complete-four-stage frontend regression passes in 15.116 seconds.
The original-resource engine suite passes in 27.661 seconds, the full frontend/GPU
suite in 171.027 seconds and focused final/source race checks in 2.186 seconds.
The game reaches level five through ordinary merchants and repairs; the earned
fifth-stage route is unfinished.

The first fifth-stage replay loses its carried ship at frame 526 to ordinary
projectiles. Additional isolated guard and barrier-alignment comparisons improve
survival but stall before the middle guardian or introduce earlier losses. They
remain excluded; no complete five-stage or near-lossless campaign is claimed.

The fifth-opening barrier stall has two verified causes. At the recorded post
lane 83..93, the ship rests at X75. Three held Right passes end at X102, so the
old distance score prefers staying 13 pixels away over overshooting by 14. A
single Right pass followed by native release coast stops inside the firing lane.
That local correction opens the barrier. Its impulse/release alignment is now
integrated with an original-resource regression; the earlier combined combat
strategy remained excluded.

One combined diagnostic reaches a later terrain-rewind loop at camera 3941,
ship X85/Y176, with 31 shield and upgraded Forward/Rear weapons. Its geometric
route is clear. After the native recovery callback restores X94 at camera 3942,
the existing motion search finds six ordinary commands to X115/world4094 in
65 expansions. Executing that route removes the loop; subsequent enemy waves
still exhaust the carried ship. No position, health or terrain is assigned by
the route executor.

Further native callback beam and longer rollout experiments either lose the
ship or stay alive without useful forward progress. Such diagnostic package
passes only confirm recorded outcomes; they are not successful game tests.
Their controllers remain local and excluded. Fifth-stage completion requires
both safe traversal and real guardian/merchant progression.

## Automatic startup demonstration

The Android startup configuration opens the original logo/credit/score loop;
it reaches the selection menu only after input. The earlier title-only inactivity
counter therefore never started a demo when that loop was left untouched. The
same 60-second counter now includes those passive presentation phases on both
desktop and Android. It spans phase changes, resets on actual input and excludes
READY, score entry, continues, game-over/ending messages, pause, fades and shops.

A full startup regression reaches ordinary level-one expert play without cheats
or diagnostic admission, then verifies same-action manual takeover. The new
startup, interactive-phase and reset checks all pass on the Pixel. This fixes
admission to the existing controller; it does not claim a complete expert campaign.
The complete frontend/GPU suite passes in 151.826 seconds after the startup fix
and core integration. The corrected APK is installed. The timeout is verified by
the native Android frontend tests; an interrupted real-screen observation is not
counted as additional visual confirmation.

## Native growing-column emission coverage

A separate completed-factory recorder executes the original inline constructors
at `0x56144` and `0x5679e`. It observes the newly initialized tag-272 projectile
record rather than the historical mixed factory counter. Across 2,700 guardian
frames, Go matches all 224 births: 49 downward columns from the middle body and
175 upward columns from final components 11 through 14. Exact emission order,
X/Y, zero initial length, signed speed, sound queues and shared RNG agree.

These checks complement the existing growing/full-length motion, collision,
physical residue, expiry and slot-reuse comparisons. Allocation is deliberately
available in this isolated emitter fixture; it does not prove behavior under a
full actor pool or a complete fifth-stage playthrough.

## Column pixels and merchant sprite anchors

The original column shaft rasterizer has been executed at 160 combinations of
horizontal alignment, vertical clipping, growing/full length, signed travel
direction and black/patterned background. Production Ebitengine rendering agrees
with its planar output. A second pass adds the raw original cap images at the
recorded original call coordinates, independently of the exported atlas decoder.
Together these checks compare 19,660,800 pixels. They found and corrected swapped
shaft palette indices and a double application of the cap anchors.

Merchant cursors, buttons and hands now use the same single-anchor drawing path.
A separate original-call recorder covers 50 selected and 50 normal cursor/button
selections, all 35 introduction-hand passes and all 47 sale-hand passes. Exported
positions, animation images and durations agree; 11,648,000 GPU pixels match the
raw masked artwork. The hand callback temporarily limits the original blitter to
row 104, and Go now applies that same inclusive limit when drawing the hand.
This restores its entry and exit through the portrait instead of covering the
interface below it.

An additional resource-independent GPU regression checks positive/negative
anchors, nonzero atlas origins, transparency and screen clipping. The original
column shaft itself is executed in the offline reference; caps and merchant
images use recorded arguments and independently decoded raw masks. These are
isolated drawing comparisons, not full Amiga framebuffer captures or evidence
of a completed five-stage campaign.
