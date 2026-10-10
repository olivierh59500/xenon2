package app

import (
	"image"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"xenon2/internal/engine"
)

// A window-only capture of ordinary first-level gameplay supplies independent
// HUD pixels. The shared viewport is fixed by the original 960-by-628 capture
// geometry; score glyphs are not fitted separately to improve their agreement.
func TestOriginalSoloGameplayHUDMatchesCaptureOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_GAMEPLAY_DIR")
	if root == "" {
		t.Skip("local original gameplay captures not supplied")
	}
	file, err := os.Open(filepath.Join(root, "00025.png"))
	if err != nil {
		t.Fatal(err)
	}
	native, _, err := image.Decode(file)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if native.Bounds().Dx() != 960 || native.Bounds().Dy() != 628 {
		t.Fatal("gameplay reference does not use the calibrated window geometry")
	}
	g, bundle := renderIntegrationGame(t)
	d, err := newWorldDriver(bundle, 1)
	if err != nil {
		t.Fatal(err)
	}
	for range 20 {
		for range 3 {
			d.world.AdvancePALTick()
		}
		if err := d.world.Step(engine.Input{}); err != nil {
			t.Fatal(err)
		}
	}
	g.Driver, g.View = d, d.Frame()
	g.rememberFrameHistory()
	if g.View.PlayerCount != 1 || g.View.Score != 0 || g.View.PlayerScores != ([2]int{}) {
		t.Fatal("HUD fixture no longer represents ordinary initial solo gameplay")
	}
	actual := renderIntegrationPixels(t, g, "native-solo-gameplay-hud")
	flat, paletteDifferences := 0, 0
	var scoreMasks, nativeScorePixels [2]int
	for y := 192; y < 200; y++ {
		for x := 0; x < 320; x++ {
			nx := int(math.Floor(124.25 + (float64(x)+.5)*2.22))
			ny := int(math.Floor(75 + (float64(y)+.5)*20/9))
			want, got := introRGB(native, nx, ny), introRGB(actual, x, y)
			if x >= 24 && x < 80 || x >= 240 && x < 296 {
				player := 0
				if x >= 240 {
					player = 1
				}
				nativeBright := max(want[0], max(want[1], want[2])) > 100
				if nativeBright {
					nativeScorePixels[player]++
				}
				if (max(got[0], max(got[1], got[2])) > 100) != nativeBright {
					scoreMasks[player]++
				}
			}
			if nativeFlatPatch(native, nx, ny, want) {
				flat++
				if got != want {
					paletteDifferences++
				}
			}
		}
	}
	// Integer source pixels become fractional-sized window pixels. The active
	// score controls the same eight-pixel edge tolerance as the inactive score;
	// ignoring glyph edges entirely would miss a completely blank second score.
	if flat < 300 || paletteDifferences != 0 || nativeScorePixels[0] < 240 || nativeScorePixels[1] < 240 || scoreMasks[0] > 8 || scoreMasks[1] > 8 {
		t.Fatalf("native HUD differs: flat%d palette%d scorePixels%v scoreMasks%v", flat, paletteDifferences, nativeScorePixels, scoreMasks)
	}
	t.Logf("Original solo HUD: %d exact flat palette pixels; score edge differences%v; both seven-digit fields remain visible", flat, scoreMasks)
}

// The native HUD routine independently renders player two's nonzero score,
// lives and shield into a planar screen. Check its actual GPU presentation in
// every level palette so a solo-display fix cannot overwrite an active player.
func TestOriginalTwoPlayerHUDMatchesNativeRasterGPUPixelsOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("local native HUD raster not supplied")
	}
	native, err := os.ReadFile(filepath.Join(root, "hud-native-3.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if len(native) != 32000 {
		t.Fatal("native HUD raster is incomplete")
	}
	g, _ := renderIntegrationGame(t)
	screen := ebiten.NewImage(ScreenWidth, ScreenHeight)
	defer screen.Dispose()
	pixels := make([]byte, ScreenWidth*ScreenHeight*4)
	for level := 1; level <= 5; level++ {
		g.View = SceneFrame{Level: level, PlayerCount: 2, PlayerScores: [2]int{0, 7654321}, PlayerLives: [2]int{3, 4}, PlayerShields: [2]int{39, 17}}
		screen.Clear()
		g.drawOriginalHUD(screen)
		screen.ReadPixels(pixels)
		for y := 192; y < 200; y++ {
			for x := 176; x < 320; x++ {
				index := 0
				for plane := 0; plane < 4; plane++ {
					if native[y*160+plane*40+x/8]&(1<<uint(7-x%8)) != 0 {
						index |= 1 << uint(plane)
					}
				}
				want := g.Bundle.Levels[level-1].Terrain.Palette[index]
				at := (y*ScreenWidth + x) * 4
				got := [4]uint8{pixels[at], pixels[at+1], pixels[at+2], pixels[at+3]}
				if got != want {
					t.Fatalf("level%d native player-two HUD at%d/%d: Go%v original%v", level, x, y, got, want)
				}
			}
		}
	}
}
