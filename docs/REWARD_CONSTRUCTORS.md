# Reward and explosion construction

The original constructors are compared with the independent Go engine for both
cash sizes, all nineteen carried equipment selectors, five exit-reward batches
and the two common explosion sizes. Three explicit fixtures leave the shared
pool empty, leave two slots free, or fill it with ordinary enemy point shots.
Initially free entries contain distinguishable retained gameplay words.

The 84 constructor cases produce 237 objects. Their initialized state and twelve
subsequent isolated callbacks match across 1,092 boundaries and 3,081 image/state
rows. Comparisons cover physical allocation/list links, tags, retained words,
coordinates, animation image/countdown, RNG, pending rewards and queued sounds.
Player contact and unrelated actor updates are disabled in these fixtures.

The comparison corrected several differences:

- Cash and equipment publish their initialized coordinates and movement state
  before another constructor can reclaim the entry.
- A bubble retains a complete sixteen-bit direction word. Movement masks the
  vector lookup; only the original steering/spiral operations replace that word.
- Exit cash and carried equipment inherit the preceding owner's direction.
  Wave cash instead stores the complete random output word.
- Equipment publishes its reward selector at creation. Other reused words and
  coordinate fractions remain untouched.
- Explosions publish their initial coordinates and finish with a zero animation
  countdown when the final frame expires.
- Reward creation and collection use effect voice one. Explosions retain their
  separate effect voice two.

Fourth-level crawler cash now uses the same production cash factory. Independent
original crawler damage and cash-motion comparisons still pass. Actual collection
of every equipment selector also matches the original queued sound while leaving
the other effect voice untouched. Resource-independent tests cover a pickup
reclaimed before its first update and full-word direction lookup/wrap behavior.
The preceding production source fails the independent factory comparison.

The corrected exit headings change collection timing. The arranged fourth-final
ordinary-input fixture reaches its merchant ten passes earlier, preserving the
same ships, shield and guardian damage. The fifth-final projectile fixture creates
all ten pairs, but one coin is already collected during the lethal projectile
pass. It now checks twenty original coin records, nineteen pending coins and
conservation of all 1,500 cash across live rewards and the wallet.

Complete engine/artwork comparisons and focused race checks pass. The public
idle-start expert tour currently needs adaptation to the corrected trajectories;
its unchanged third-stage survival assertion detects the regression. These
changes remain on the development branch until both admission journeys pass.

The fixtures validate constructors and isolated callbacks, not every crowded
parent-death interaction, hardware drawing, audible mixing or a complete
five-stage playthrough. The original resources and reference traces stay local.
