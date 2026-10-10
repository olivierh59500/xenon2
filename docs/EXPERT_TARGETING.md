# Expert targeting improvements

The expert controller is being extended to clear destructible emitters, use its
installed rear and side weapons, and collect more cash and equipment bubbles.
This branch is experimental. The published three-level tour remains on `main`.

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

Focused aiming, rear engagement, terrain-cannon, linked-damage and central-bonus
regressions pass. Existing preparation/isolation and fourth-middle ownership
checks also pass after keeping these improvements within the public three-level
scope. Survival requirements in the complete journey tests remain unchanged.

The current idle-start experiment finishes all three levels with its three ships
and two continue credits intact. It records 221, 285 and 226 scoring removals,
with 4,150, 3,800 and 5,700 collected cash. The explicit-start experiment improves
combat in the first two levels but still loses five ships during the third and
fails the existing survival gate. It is not ready to replace the published tour.

The scorecard also distinguishes observed cash and equipment bubbles. It tracks
known live objects until collection, expiry at Y=200 or eviction. A bubble born
and collected within one unobserved update may not enter that census, so these
counts complement the exact money totals rather than claiming every creation.

| Route | Level | Cash seen / collected / missed | Equipment seen / collected / missed |
| --- | ---: | --- | --- |
| Published idle start | 1 | 51 / 40 / 11 | 7 / 6 / 1 |
| Published idle start | 2 | 84 / 63 / 21 | 5 / 4 / 1 |
| Published idle start | 3 | 101 / 87 / 14 | 6 / 5 / 1 |
| Experimental idle start | 1 | 67 / 57 / 10 | 7 / 7 / 0 |
| Experimental idle start | 2 | 83 / 52 / 31 | 5 / 3 / 2 |
| Experimental idle start | 3 | 98 / 81 / 17 | 6 / 4 / 2 |

The first level collects more observed rewards, while later levels still miss
too many. Further work must improve collection and emitter clearance while
preserving safe complete journeys for both admission paths. Gameplay rules,
health, money, collisions and the shared random stream remain unchanged.
