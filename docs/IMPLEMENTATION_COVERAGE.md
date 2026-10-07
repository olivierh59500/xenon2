# Game implementation coverage

The conversion uses an independent Go simulation and Ebitengine renderer.
This inventory distinguishes integrated game behavior from extracted artwork
and isolated controller comparisons. A resource export does not by itself
establish that the corresponding encounter is playable.

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

Actual Ebitengine GPU tests render six guardian/beam scenes. Pixel comparisons
cover aura alpha, terrain materialization, body tile placement and damage flash,
moving/effect layer order, all five palette strobes and Shades, and the 48-point
background starfield. A shader source-size panic and double source-origin
adjustment were found through those tests and corrected. Desktop menu, attract,
shop and five level captures have been inspected.

A real fifteen-second desktop run measured approximately 60 updates per second
and 120 draw calls per second on its 120 Hz display after warm-up. That run does
not establish dense-combat performance or a complete playthrough. A legal-input
replay completes the first level, defeats its guardian through normal firing,
and reaches its final shop after the exit coins drain. The independent replay
test validates all 8,198 recorded input/state steps with real terrain and legal
continues. The other four levels and complete frontend campaign still require
playthrough validation.

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
