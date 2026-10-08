# Fourth-level opening validation

The carried presentation completes the first three levels, including the real
third final merchant, before entering level four. It enters with one ship, no
continue credits, 39 shield, double shot tier 1, rear shot tier 0, speed 2 and
native autofire advance 3 / period 8. No cheats or diagnostic level selection are
used in this route.

The fourth pre-middle opening now uses the existing six-pass isolated world
forecast. Its eligibility is level four, no admitted middle guardian, camera
above 176, an active player and no READY screen. The nine candidate directions,
ranking, fire/dive inputs and third-level scopes remain unchanged.

## Original-resource regressions

The unit scenes retain the carried equipment shape and original terrain, ship,
projectile and actor resources. They isolate individual decisions rather than
reconstructing the complete campaign random stream.

- Two diagonal projectiles retain the fixed-point coordinates observed at the
  original carried run's frame 137/138 impact passes. Their source advances and actual
  collision prefixes cause four shield damage under held controls; the guard
  preserves 39 shield.
- The original kind 4/path 31 three-member formation retains resource tag 236 and
  its source motion callbacks. Its curved approach causes eight contact damage
  under held controls; the guard preserves 39 shield.
- The guard leaves the live world, equipment and random stream unchanged. It
  does not enter an admitted middle guardian, the final boundary, READY, a
  destroyed player or another level.

The previous fourth-ineligible guard fails the projectile and curved-wave
regressions. Targeted race checks, including the retained third-level sequence
and final-guardian cases, pass.

## Connected Pixel result

The comparison uses the complete default-intro frontend route on the connected
Pixel 10a. It runs game logic without drawing or global input control; this is a
functional route check rather than a graphical or frame-rate measurement.

Without the fourth opening guard, six bullets and two actor contacts consume
40 shield damage against the available 39. The first ship loss occurs at
frame 679, camera 3933, before the middle guardian.

With the guard, the route survives with 27 shield. Three ordinary bullets cause
four damage each at frames 530, 533 and 681. The two previous actor contacts are
avoided. The route then stops making forward progress at camera 4018 and enters
an ordinary terrain-rewind cycle around camera 4019/4020, ship positions (263/272, 176). At
frame 2179 it still has its carried ship and 27 shield, but no navigation path and
no middle guardian admission. A 90-second forward-progress stop ends the check.

## Preparing the left forest fork

The practiced controller now prepares the original left corridor before the
right-hand pocket closes. Edge-safe geometric routing supplies the next point;
the native-motion planner retains the actual commands needed to reach it. The
safety guard previews that retained command sequence rather than holding its
first direction for all six forecast passes. Tactical aiming does not replace
the active corridor specialist.

Two original-resource scenes execute nine and eight commands from observed
earlier poses. Every ordinary World.Step matches the predicted position and
camera, retains rewind zero and clears the complete ship stencil. A later pose
inside the closed right pocket remains unreachable; no exit is invented.

The connected Pixel run again passes the strict first-three-level helper and
clears the previous camera4018 stall. It reaches camera3863, then stops making
forward progress for90 seconds with the same ship and11 shield. Its geometric
route begins with a rearward turn, but generic controls remain Down and cycle
through terrain rewind. The full diagnostic takes70.32 seconds without drawing.

The opening safety and left-fork improvements are verified. Executing subsequent
terrain turns, the middle guardian and the remainder of levels four and five
remain open. The logical runner does not establish visual frame pacing.
