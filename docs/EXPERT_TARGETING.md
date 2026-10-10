# Expert targeting improvements

The expert controller clears destructible emitters, uses its installed rear and
side weapons, and plans cash and equipment collection alongside combat. These
changes apply to the public three-level tour. A Rear Shot and a Side Shot replace
one another under the original equipment rules; the pilot only uses the weapon
that is actually installed.

The targeting changes cover four previously excluded opportunities:

- Rear and side firing windows survive a forward burst pause and the final dodge.
- Trailing formations can become rear engagements with enough room for the muzzle.
- Terrain cannons remain targets when their artwork is drawn into the map rather
  than as a standalone sprite.
- Linked body sections forwarding real damage to their leader remain aim targets.

Central rewards are no longer excluded solely because the ship is more than
65 pixels below them. Stationary emitters receive additional targeting priority.
The first two levels use a twelve-pass safety forecast for ordinary combat.
Planning buffers retain occupancy data while clearing previous candidate routes.

## Current validation

The mixed enemy/projectile audit restores first-stage burst artwork and the
first guardian's physical constructor/curve state. Ordinary burst shots now
have their intended collision prefix as well as their image. The current public
journey checks fail their unchanged survival gates, so the controller needs
further adaptation to the corrected simulation. The private fifth-stage
practice recording is also stale; its existing replay checks remain failing.

The source comparisons and resource-independent regressions for these fixes
pass, including the mixed phase/list boundaries and the ship prefix used by
ordinary projectiles in all five stages. These checks do not establish current
complete-journey survival.

## Last verified three-level baseline

Focused aiming, rear engagement, terrain-cannon, linked-damage and central-bonus
regressions pass. Existing preparation/isolation and fourth-middle ownership
checks also pass after keeping these improvements within the public three-level
scope. Survival requirements in the complete journey tests remain unchanged.

At `8146a5b`, both full frontend journey regressions finished the three levels
with three ships and two continue credits intact. The idle-start journey finished at
1,510.00 seconds with 159,970 points; explicit admission finishes at 1,463.60
seconds with 164,350 points. The two independent scorecard observation runs also
finished with no ship losses or spent continues. The figures below describe
that baseline, before the mixed-phase fixes. Its original-resource engine suite
already exposed the stale fifth-stage recording; it was not fully green.

Nearby compound cannons receive candidate lanes at both edges and the center
of their actual weak point. Screen-space attack goals yield to rearward terrain
legs rather than keeping the ship safely circling in front of a blocked passage.
Four reachable bubble goals are reserved before generic placement candidates
fill the fixed planning bank. Rehearsals rank real remaining shield, damage and
terrain contact before destruction, cash, equipment collection and progression.
This allows an ordinary health pickup to improve a route's safety; it does not
grant health or change pickup behavior.

The current simulation includes original inherited bubble headings and effect
voices, verified separately in [REWARD_CONSTRUCTORS.md](REWARD_CONSTRUCTORS.md).
The controller credits real partial emitter damage before a destruction award
and reconsiders a retained route if its next callbacks predict shield loss.
Earlier scorecards therefore differ in simulation fidelity as well as controls.

The scorecard also distinguishes observed cash and equipment bubbles. It tracks
known live objects until collection, expiry at Y=200 or eviction. A bubble born
and collected within one unobserved update may not enter that census, so these
counts complement the exact money totals rather than claiming every creation.

| Route | Level | Cash seen / collected / missed | Equipment seen / collected / missed |
| --- | ---: | --- | --- |
| Published idle start | 1 | 51 / 40 / 11 | 7 / 6 / 1 |
| Published idle start | 2 | 84 / 63 / 21 | 5 / 4 / 1 |
| Published idle start | 3 | 101 / 87 / 14 | 6 / 5 / 1 |
| Published explicit start | 1 | 47 / 37 / 10 | 6 / 6 / 0 |
| Published explicit start | 2 | 84 / 66 / 18 | 5 / 5 / 0 |
| Published explicit start | 3 | 95 / 77 / 18 | 6 / 4 / 2 |
| Updated idle start | 1 | 67 / 58 / 9 | 7 / 7 / 0 |
| Updated idle start | 2 | 83 / 51 / 32 | 5 / 3 / 2 |
| Updated idle start | 3 | 95 / 83 / 12 | 7 / 6 / 1 |
| Updated explicit start | 1 | 57 / 49 / 8 | 7 / 5 / 2 |
| Updated explicit start | 2 | 83 / 59 / 24 | 4 / 2 / 2 |
| Updated explicit start | 3 | 96 / 78 / 18 | 7 / 6 / 1 |

Here, "published" identifies the preceding controller at `2c3a6a8`.

| Controller / admission | Scoring removals | Collected cash | Ship losses |
| --- | ---: | ---: | ---: |
| Previous / idle | 706 | 13,650 | 1 |
| Updated / idle | 723 | 13,900 | 0 |
| Previous / explicit | 694 | 13,200 | 1 |
| Updated / explicit | 744 | 13,800 | 0 |

The first level collects more observed rewards, while later levels still miss
too many. In particular, second-level collection regresses even though both
complete tours destroy more actors without ship losses. Cash improves on the
explicit and idle tours. Changed combat also changes which bubbles are created;
the current simulation includes all eight original extending-chain explosions
and immediate terrain-cannon constructor state. These figures do not establish
that every destructible or reward is reached.

The fixed bank still contains at most 24 candidates over 72 simulation passes,
and only three verified commands are retained before reassessment. A warm
20-decision third-opening sample at `cf9f9c7` measured 3.09 ms per decision on an M4 Max;
it is a scene-specific CPU measurement, not a Pixel frame-rate result.

The controller assigns no health, money, collision changes or random values.
Further collection improvements need complete-journey validation as well as
local aiming and interception tests.
