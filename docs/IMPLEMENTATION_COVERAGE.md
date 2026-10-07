# Game implementation coverage

The conversion uses an independent Go simulation and Ebitengine renderer.
This inventory distinguishes integrated game behavior from extracted artwork
and isolated controller comparisons. A resource export does not by itself
establish that the corresponding encounter is playable.

## Fixed encounters and scripted arenas

| Level | Integrated families | Remaining work |
| --- | --- | --- |
| 1 | Three terrain cannon types, bouncing attackers, five-stream middle arena, final body and articulated chain | Complete damage/removal effects and checkpoint/playthrough comparisons |
| 2 | Terrain cannon, bouncing attackers, hatches, both pod emitters and their creatures, middle defense waves/nodes, final body and transforming minions | Complete source effects and full playthrough comparisons |
| 3 | Terrain cannons, sweeping attacker, turning projectile, extending chain, tile-edge crawler, middle flying guardian, final worm and conditional scenery | Full phase-order and playthrough comparisons |
| 4 | Terrain cannon, extending beam, nest crawler, twenty-part middle guardian and nineteen-part final guardian | Two stage-generated encounter families; full playthrough comparisons |
| 5 | Terrain cannon, destructible barrier, aiming turret, vertical attacker, aiming projectile, both compound guardians, growing laser columns and mouth creatures | Three additional fixed selector families; full playthrough comparisons |

The supplied encounter streams contain respectively 5, 6, 7, 6 and 10 selectors,
including the checkpoint selector in each level. Every selector requires an
explicit semantic implementation; unknown selectors must not be mistaken for
empty encounters.

## Shared systems

Movement, path commands, deterministic randomness, equipment, projectiles,
terrain coverage, cash and carrier rewards, checkpoint recovery, alternating
players and the shared 159-slot allocator have source comparisons. Terrain and
sprite transparency use their distinct original coverage formats.

The session has explicit intermediate-shop and stage-completion boundaries.
The two-player stage gate preserves each saved game; level five increases
difficulty and loops to level one. Frontend loading, shop and ending routes are
connected. Live scene comparisons remain separate from simulation tests.

## Validation boundaries

Private source comparisons run only when the local decoded disk resources and
trace directories are supplied. Those resources and traces remain excluded
from Git. Ordinary tests verify the independent logic without an emulator.

The build draws at 60 updates per second and uses a separate gameplay clock.
Integrated visual comparisons and a complete five-level run remain required.
The current build is in development and is not yet a complete conversion.
