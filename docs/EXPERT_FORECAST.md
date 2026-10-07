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
loss, retaining 39 and 31 shield respectively. Later middle-guardian victory is
not established; an unrestricted trial can choose a stationary safe position.

This first implementation prioritizes correct ownership over allocation reuse.
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
