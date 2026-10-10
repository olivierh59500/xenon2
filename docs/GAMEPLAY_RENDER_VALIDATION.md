# Original gameplay rendering comparisons

The first-level reference is a window-only PNG capture of ordinary Amiga
gameplay. It uses the original 960-by-628 emulator window and the viewport
calibration already established for that window. The twenty-pass Go scene
retains default equipment and ordinary controls. The capture's HUD shows two
remaining ships; its preceding controls and background phase are not known.
The capture comparison concerns the scores and flat HUD colors. It does not
equate those two gameplay states or certify a complete scene match.

## Solo score fields

The original keeps both seven-digit score fields visible in solo play. Player
two's inactive score is zero. The Go renderer previously drew only the active
player's field, leaving the other score blank. A gameplay capture exposed 248
different score-mask pixels; the active field had seven edge differences under
the same fixed viewport sampling.

The renderer now draws the inactive zero score as well. The reference comparison
retains 322 exact flat palette samples. The two score masks differ at seven and
four edge pixels respectively. The shared eight-pixel edge tolerance accounts
for the noninteger window scaling; no score-specific fitting is performed.
Checking only flat patches would have missed the blank score, so the regression
also checks every glyph-mask pixel and requires visible digits in both fields.

## Active second player

A separate original HUD-routine raster contains player two's score 7,654,321,
four lives and 17 shield. The GPU regression compares its entire right-hand
HUD region in all five level palettes: 5,760 exact RGBA pixels. This confirms
that the solo correction preserves an active second player's displayed state.
The original planar raster and capture remain local and excluded from Git.

## Background and terrain rasterization

A separate offline reference executes the original game's background/map draw
routine for sixty views across all five levels. Its hardware adapter implements
only the synchronous copy, clear and masked blits requested by that routine.
The reference reads original map and planar graphics bytes directly; it does
not construct expected pixels from the exported Go tile atlas.

Each level contributes twelve camera/background pairs. They include the opening,
middle and final areas, camera positions on both sides of a sixteen-pixel tile
boundary, and background offsets zero, one, 190 and 191. The renderer's full
320-by-192 output matches every reference pixel: 3,686,400 exact RGBA comparisons.
This covers palette selection, opaque and masked terrain, top/bottom clipping
and circular background copying. There is no image fitting or color tolerance.

These arranged draw states establish static background/map composition. They
do not establish enemy drawing, mutable terrain during combat, interpolation
between frames, previous player inputs or physical Amiga blitter timing.
The original first-level window capture also confirms that the exported
background artwork matches the original at a different retained scroll phase;
that phase must not be imposed on every new Go game.

With the prepared runtime resources and private references, run:

```sh
GOWORK=off \
XENON2_RUNTIME_TEST_DIR="$PWD/assets/runtime" \
XENON2_NATIVE_GAMEPLAY_DIR="$PWD/.local/captures/native/first-gameplay" \
XENON2_NATIVE_TRACE_DIR="$PWD/.local/analysis" \
go test ./internal/app \
  -run '^TestOriginal(SoloGameplayHUDMatchesCapture|TwoPlayerHUDMatchesNativeRasterGPUPixels)Optional$' \
  -count=1 -v
```

After generating the private terrain rasters under the same trace directory:

```sh
GOWORK=off \
XENON2_RUNTIME_TEST_DIR="$PWD/assets/runtime" \
XENON2_NATIVE_TRACE_DIR="$PWD/.local/analysis" \
go test ./internal/app \
  -run '^TestOriginalTerrainRastersMatchProductionGPUOptional$' \
  -count=1 -v
```

The checks require an active graphics context. Their current Xvfb/Mesa run
passes in 1.038 seconds for the HUD and 0.559 seconds for all terrain rasters.
These are bounded rendering comparisons; integrated actors, animation, audio
and real-time pacing checks remain separate.
