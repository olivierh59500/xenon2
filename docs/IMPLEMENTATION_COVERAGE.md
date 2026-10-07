# Game implementation coverage

The conversion uses an independent Go simulation and Ebitengine renderer.
This inventory distinguishes integrated game behavior from extracted artwork
and isolated controller comparisons. A resource export does not by itself
establish that the corresponding encounter is playable.

## Fixed encounters and scripted arenas

| Level | Integrated families | Remaining work |
| --- | --- | --- |
| 1 | Three terrain cannon types, bouncing attackers, five-stream middle arena, final body and articulated chain | Complete damage/removal effects and checkpoint/playthrough comparisons |
| 2 | Terrain cannon, bouncing attackers, middle nodes and articulated defense waves, final body and transforming minions | Fixed hatches and their eight creatures, two additional fixed sprite families |
| 3 | Terrain cannon, sweeping attacker and turning projectile | Middle flying guardian and final worm integration; three additional fixed families |
| 4 | Terrain cannon and extending beam | Middle and final compound guardians; ground encounter family |
| 5 | Terrain cannon and vertical attacker with aiming projectile | Middle and final compound guardians; six additional fixed families |

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
being connected and checked independently of these simulation tests.

## Validation boundaries

Private source comparisons run only when the local decoded disk resources and
trace directories are supplied. Those resources and traces remain excluded
from Git. Ordinary tests verify the independent logic without an emulator.

The build draws at 60 updates per second and uses a separate gameplay clock.
Integrated visual comparisons and a complete five-level run remain required.
The current build is in development and is not yet a complete conversion.
