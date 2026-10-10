# Earlier complete campaign and ending validation

These connected campaign results apply to runtime `2f771a3`, before the latest
correction to retained actor storage and ship positions across level changes.
They are not a passing campaign check for the current source. The latest pilot
reaches level three without losing a ship, then fails its corridor survival
regression. The later recorded profiles and connected campaign need renewal.
Current original transition evidence is documented in
[SESSION_POOL_VALIDATION.md](SESSION_POOL_VALIDATION.md).

That independent Go build has a connected ordinary-input reference
journey from the default intro through all five stages, the final merchant,
every ending phase and the first playable pass of the second difficulty round.
The public desktop and Android expert tour remains limited to the first three
stages.

## Connected game flow

The reference uses the ordinary intro and current four-stage controller. It
earns the fifth-stage equipment, both middle merchant transactions and the final
encounter through real game callbacks. Its fifth opening and second half retain
all three starting ships and both continues. Final admission at frame 5,507 has
39 shield, Forward 2 / Laser 2 / Side 1, Protection, 850 cash and all eighteen
defenses plus the 20-health core.

At that boundary, recorded directions and fire take over through the same input
interface used by a human. The test does not assign health, equipment, RNG,
money, score, encounter completion or victory flags. The final fight loses one
ship; the real frontend runs its death presentation, checkpoint admission and
READY acknowledgement. Both continues remain unused.

| Current final boundary | Frame | Camera | Shield | Ships | Cash | Score |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Final admission | 5,507 | 415 | 39 | 3 | 850 | 280,050 |
| Core defeated | 6,562 | 206 | 9 | 2 | 850 | 284,750 |
| Final merchant | 6,605 | 163 | 9 | 2 | 2,350 | 284,750 |

Every outer defense is destroyed before the original core gate permits victory.
The death callback creates twenty native exit coins. Ordinary movements collect
all 1,500 cash before the real merchant opens; no reward counter or wallet is
assigned. Fifth completion enters the merchant ending directly, without exposing
an equipment sale or purchase page. The regression observes the merchant ending,
fade, ending dot, dot fade and ending wait individually.

The normal next-stage loader then creates difficulty round two. It preserves
score 284,750, two ships and nine shield. The original victory credit raises the
continue balance from two to three. The source weapon reset restores the basic
Forward Shot and clears mounted, rear and side weapons. The regression confirms
the ordinary level-one READY director and first live gameplay pass.

## Reproducible check

After [preparing the original disk resources](ASSET_SETUP.md), run with an active
graphics context:

```sh
GOWORK=off \
XENON2_RUNTIME_TEST_DIR="$PWD/assets/runtime" \
XENON2_HUMAN_PRESENTATION_CHECK=1 \
go test ./internal/app \
  -run '^TestCurrentFiveStageReferenceCompletesEndingAndStartsNextRoundOptional$' \
  -count=1 -v -timeout=5m
```

The earlier Ebitengine/Xvfb/Mesa run passes in 44.30 seconds. This is accelerated
test execution, not the duration of real-time gameplay. Its 1,099 final inputs
contain directions and fire only; a single marked READY acknowledgement crosses
the actual director after the ship loss. The controls contain no original artwork
or executable bytes. The optional Android logic runner retains the game-flow
assertions while omitting GPU captures. The physical Pixel 10a passes that
complete reference in 145.94 seconds on runtime `2f771a3`, including the same
score, final reward drain, one ship loss, unchanged continue spending and
next-round admission.
This locked-device check establishes ARM64 game-flow agreement; it does not
measure frame pacing, sound output or touchscreen interaction.

Set `XENON2_RENDER_CAPTURE_DIR` to an excluded local directory to retain four
actual GPU frames: final victory, stable merchant ending, next-round READY and
next-round gameplay. These captures have been inspected locally. Original
resources and diagnostic captures remain excluded from Git.

## Scope and fidelity limits

This proves connected Go progression in that earlier build through the complete campaign and
ending. The preceding captured-entry final fixture also wins, but its missing
frontend presentation timing produces a different post-death random stream,
shield balance and reward motion. Its 19-shield result must not replace the
connected frontend's nine-shield result.

The recording is a reference validation, rather than a new public five-stage
expert feature. It does not establish an optimal final fight, collection of every
earlier bonus, integrated Amiga audiovisual equivalence or real-time hardware
smoothness. The bounded original-controller, physical-state, rendering and audio
comparisons remain documented in [NATIVE_AUDIT.md](NATIVE_AUDIT.md).
