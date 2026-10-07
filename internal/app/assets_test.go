package app

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"image/color"
	"os"
	"testing"
	"xenon2/internal/engine"
)

func TestOriginalPaletteShaderCompiles(t *testing.T) {
	if _, err := ebiten.NewShader([]byte(paletteShaderSource)); err != nil {
		t.Fatal(err)
	}
	if _, err := ebiten.NewShader([]byte(sparkShaderSource)); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{terrainClippedSpriteShaderSource, backgroundStarShaderSource, fadeShaderSource} {
		if _, err := ebiten.NewShader([]byte(source)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNormalSessionSnapshotDisablesReferenceShortcuts(t *testing.T) {
	dir := os.Getenv("XENON2_RUNTIME_TEST_DIR")
	if dir == "" {
		t.Skip("local exported resources not supplied")
	}
	bundle, err := LoadFS(os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	game := &Game{Bundle: bundle}
	session, err := engine.NewSession(game.levelData(1), 1, engine.NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	driver := &worldDriver{world: session.ActiveWorld(), session: session}
	if driver.Frame().Diagnostic {
		t.Fatal("normal game exposed reference-level and shop shortcuts")
	}
	driver.diagnostic = true
	if !driver.Frame().Diagnostic {
		t.Fatal("reference views must retain their inspection shortcuts")
	}
}

func TestDiveAndNashwanMetersReachGameplaySnapshots(t *testing.T) {
	dir := os.Getenv("XENON2_RUNTIME_TEST_DIR")
	if dir == "" {
		t.Skip("local exported resources not supplied")
	}
	bundle, err := LoadFS(os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	driver, err := newWorldDriver(bundle, 1)
	if err != nil {
		t.Fatal(err)
	}
	w := driver.world
	w.MaterializationFrames = 0
	w.Equipment.SuperFrames, w.Dive.Remaining = 1, 1
	if err := driver.Advance(Input{}); err != nil {
		t.Fatal(err)
	}
	frame := driver.Frame()
	if len(frame.HUD) != 2 || frame.HUD[0].Sprite != bundle.Common.TimerFrames[0] || frame.HUD[0].X != 152 || frame.HUD[0].Y != 8 || frame.HUD[1].Y != 160 {
		t.Fatal("source expiry meters did not reach the rendered HUD snapshot")
	}
	if err := driver.Advance(Input{}); err != nil {
		t.Fatal(err)
	}
	if len(driver.Frame().HUD) != 0 {
		t.Fatal("expired meters persisted into the next snapshot")
	}
}

func TestWrapInterpolationKeepsParallaxContinuous(t *testing.T) {
	if got := wrapLerp(191, 0, .5, 192); got != 191.5 {
		t.Fatalf("forward wrap traverses image: %g", got)
	}
	if got := wrapLerp(0, 191, .5, 192); got != 191.5 {
		t.Fatalf("reverse wrap traverses image: %g", got)
	}
	if got := wrapLerp(18, 20, .5, 192); got != 19 {
		t.Fatal(got)
	}
}

func TestFlashUsesPaletteFifteenOnlyInsideOriginalCoverage(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	source.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	source.SetNRGBA(1, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 0})
	result := flashPixels(source, source.Bounds(), [4]uint8{170, 68, 34, 255})
	if result.NRGBAAt(0, 0) != (color.NRGBA{170, 68, 34, 255}) || result.NRGBAAt(1, 0).A != 0 {
		t.Fatal("flash altered source coverage or selected a different palette color")
	}
}

func TestPrivateExportedBundleAndWorldSnapshots(t *testing.T) {
	dir := os.Getenv("XENON2_RUNTIME_TEST_DIR")
	if dir == "" {
		t.Skip("local exported resources not supplied")
	}
	bundle, err := LoadFS(os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	for level := 1; level <= 5; level++ {
		driver, err := newWorldDriver(bundle, level)
		if err != nil {
			t.Fatalf("level %d: %v", level, err)
		}
		frame := driver.Frame()
		if frame.Player.X != 160 || frame.Player.Y != 176 || frame.CameraY != 4608 || !frame.Diagnostic {
			t.Fatalf("level %d initial view %+v", level, frame)
		}
		for n := 0; n < 160; n++ {
			if err = driver.Advance(Input{}); err != nil {
				t.Fatalf("level %d frame %d: %v", level, n, err)
			}
		}
		frame = driver.Frame()
		if len(frame.TerrainMap) != 6000 || frame.Level != level {
			t.Fatalf("level %d snapshot lost its terrain", level)
		}
		for _, sprite := range frame.Sprites {
			if sprite.ID == 0 {
				t.Fatal("unstable sprite identity")
			}
			if sprite.Kind == "tiles" {
				if len(sprite.Patch.Tiles) != sprite.Patch.Columns*sprite.Patch.Rows {
					t.Fatal("guardian body patch incomplete")
				}
				continue
			}
			var found bool
			atlas := bundle.Levels[level-1].Actors.Atlas
			switch sprite.Atlas {
			case "fixed":
				atlas = bundle.Levels[level-1].FixedSprites.Atlas
			case "enemy-shots":
				atlas = bundle.Levels[level-1].Rules.EnemyShots
			case "common":
				atlas = bundle.Common
			case "guardian-parts":
				atlas = *bundle.Levels[level-1].GuardianParts
			case "guardians":
				atlas = bundle.Levels[level-1].Guardians.Atlas
			}
			for _, region := range atlas.Sprites {
				if region.Name == sprite.Sprite {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("level %d snapshot refers to missing %s/%s", level, sprite.Atlas, sprite.Sprite)
			}
		}
	}
}
