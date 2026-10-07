# Expert gameplay forecast

The demonstration controller aims to complete all five levels through ordinary
controls, using known terrain routes, enemy formations, carrier rewards and boss
phases. The current carried route still loses too much energy and does not yet
complete the campaign. More accurate individual movement helpers cannot predict
all interactions between births, firing, combat, terrain and shared actor slots.

## Isolated full-world lookahead

A reusable typed state copy can run the same Go `World.Step` callbacks against a
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

A candidate API is `WorldForecast.Load(*World) error`, followed by explicit PAL
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
