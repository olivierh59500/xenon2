# Game implementation coverage

The conversion uses an independent Go simulation and Ebitengine renderer.
This inventory distinguishes integrated game behavior from extracted artwork
and isolated controller comparisons. A resource export does not by itself
establish that the corresponding encounter is playable.

The current complete-intro Go frontend completes levels one through three,
including their real merchants, guardians and exit rewards. The third level
retains its carried ship and shield through the final fight. Levels four and
five and whole-level Amiga visual/playthrough comparisons remain under audit.

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

## Shared systems

Movement, path commands, deterministic randomness, equipment, projectiles,
terrain coverage, cash and carrier rewards, checkpoint recovery, alternating
players and the shared 159-slot allocator have source comparisons. Terrain and
sprite transparency use their distinct original coverage formats.

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
not establish dense-combat performance or a complete playthrough. A historical
public-input replay completed the first level before the lethal-contact
correction. Its observed state now diverges at step 920, so it no longer proves
current first-level completion. Current normal-menu pilot regressions complete
levels one and two through the corrected engine, including all four merchants
and guardian/exit-coin gates, then admit level three. The third opening, middle
and final arenas have separate ordinary-input resource proofs. The remaining
connected stages and complete campaign require current validation.

Finite common explosions and converted fourth-stage pods now retire and release
their physical slots. Animation-only effects preserve untouched slot fields;
third terrain crawlers and compound cannons publish their native physical state.
Resource regressions cover actual slot release/reuse. Fifth core hits flash
their owning tiled body, and the middle body retains its last normal muzzle
table during flash callbacks. A real basic-gun hit also passes the production
GPU pixel check for the final body.

After native fractional enemy-shot coordinates were corrected, the current
direct-menu route enters stage three with three ships; the complete-intro route
enters it with two. Both gameplay cadences retain real shop/guardian/exit gates.
The separate no-loss third-middle regression still detects a lost ship before
the arena. The earlier legal-continue probe predates the fractional correction
and does not establish the current carried route. Isolated arena wins are not
substitutes for a connected victory through ordinary controls.

The practiced presentation controller now also completes the entire first level
from the production intro: known left junction, both genuine merchants, final
guardian destruction and collected exit drops. Its exported MP4 is 413.05 seconds
and ends before playing level two. The practiced controller now carries the
real first-stage result through the second arena and enters its middle merchant,
with ordinary deaths and one continue. Hazard-aware final preparation now also
completes stage two, its final merchant and exit rewards, then admits stage three
with one ship after ordinary recovery. A permanent real-intro regression covers
that entire route. The separate reference controller and practiced presentation
have distinct survival results; connected victory through stages three to five
remains unverified. A current full-intro reference observation reaches and
defeats the third middle guardian after ordinary recovery and visits its
merchant, but then stalls in a terrain turn at camera 2,440. An experimental
route clears that turn and loses its last ship farther ahead; it is not part
of the production pilot or evidence of a third-stage victory.

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
merchant with its carried ship and all 19 admission shield points intact. The
36-pass boss forecast covers the arm's extension and recovery, but its mobile
cost does not establish a smooth-frame guarantee. Private concurrent branches
preserve serial decisions, and native command sequences now clear the narrow
later routes. The actual Pixel frontend reaches checkpoints 1696 and 1152 with
27 shield, then the final third guardian with 19 shield, without another ship
loss. Its logical runner works on the locked device; this does not verify drawing.
The clean complete-intro regression now also defeats the final third guardian
without shield loss, collects its exit rewards, visits the real final merchant
and enters stage four with the same ship and repaired shield. Stages four/five
and low-loss play across the whole campaign remain open. Automatic title admission after sixty idle seconds is shared by the
desktop and Android frontend, with immediate manual takeover.

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
