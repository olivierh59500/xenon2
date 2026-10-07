package app

import (
	"errors"
	"github.com/hajimehoshi/ebiten/v2"
	"os"
	"testing"
	"xenon2/internal/engine"
	"xenon2/internal/shopui"
)

func presentationFrontendGame(t *testing.T) *Game {
	t.Helper()
	root := os.Getenv("XENON2_RUNTIME_TEST_DIR")
	if root == "" {
		t.Skip("local exported resources not supplied")
	}
	bundle, err := LoadFS(os.DirFS(root))
	if err != nil {
		t.Fatal(err)
	}
	g, err := NewConfiguredGame(bundle, Config{Level: 1, StartScreen: PresentationScreen, Demo: true, HumanDemo: true, Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// This runs ordinary frontend controls from the real intro. Motion measurements
// establish activity, not human appearance or complete campaign success; video
// inspection remains separate from this bounded regression.
func TestPresentationPilotMovesAndSelectsFireThroughRealFrontendOptional(t *testing.T) {
	if os.Getenv("XENON2_HUMAN_PRESENTATION_CHECK") == "" {
		t.Skip("enable the real presentation pilot check explicitly")
	}
	g := presentationFrontendGame(t)
	var lastFrame uint64 = ^uint64(0)
	var lastPlayer engine.PlayerMotionState
	passes, moving, firing, resting := 0, 0, 0, 0
	quietRun, longestQuietRun := 0, 0
	minimumX, maximumX, minimumY, maximumY := ScreenWidth, 0, PlayfieldHeight, 0
	for update := 0; update < 60*160; update++ {
		advanceFrontend(t, g, inputFrame{})
		if g.Screen != LevelScreen || g.backdropOnly || g.View.Ready {
			continue
		}
		w := g.Driver.(*worldDriver).world
		if w.Frame == lastFrame {
			continue
		}
		if w.Cheats.Enabled() || w.Level.Number != 1 || w.GameOver || !g.DemoActive() {
			t.Fatal("presentation did not retain the ordinary first-stage rules")
		}
		if passes > 0 && (w.Player.X != lastPlayer.X || w.Player.Y != lastPlayer.Y) {
			moving++
		}
		if g.demo.controls.fire {
			firing++
			quietRun = 0
		} else {
			resting++
			quietRun++
			longestQuietRun = max(longestQuietRun, quietRun)
		}
		minimumX, maximumX = min(minimumX, w.Player.X), max(maximumX, w.Player.X)
		minimumY, maximumY = min(minimumY, w.Player.Y), max(maximumY, w.Player.Y)
		passes++
		lastFrame, lastPlayer = w.Frame, w.Player
	}
	if passes < 500 || moving < passes/5 || maximumX-minimumX < 80 || resting < passes/4 || firing == 0 || longestQuietRun < 12 {
		t.Fatalf("presentation activity insufficient: passes%d moving%d firing%d resting%d x%d..%d y%d..%d", passes, moving, firing, resting, minimumX, maximumX, minimumY, maximumY)
	}
	t.Logf("Ordinary presentation: %d passes, %d moving, %d firing, %d resting; longest quiet%d; x%d..%d y%d..%d", passes, moving, firing, resting, longestQuietRun, minimumX, maximumX, minimumY, maximumY)
}

func TestPresentationPilotKnownLeftRouteEarnsFirstShopFundsOptional(t *testing.T) {
	if os.Getenv("XENON2_HUMAN_PRESENTATION_CHECK") == "" {
		t.Skip("enable the real presentation route check explicitly")
	}
	g := presentationFrontendGame(t)
	left := false
	for update := 0; update < 60*240; update++ {
		advanceFrontend(t, g, inputFrame{})
		d, ok := g.Driver.(*worldDriver)
		if !ok || d.session == nil {
			continue
		}
		w := d.world
		if w.Cheats.Enabled() || d.diagnostic {
			t.Fatal("presentation route changed ordinary game rules")
		}
		left = left || w.Level.Number == 1 && w.ScrollY >= 3160 && w.ScrollY <= 3344 && w.Player.X < 140
		if g.Screen == ShopScreen && !g.shopFinal {
			if !left || w.FirstMiddle == nil || !w.FirstMiddle.Crossed || w.Equipment.Lives < 1 || w.Money < 50 {
				t.Fatal("presentation omitted the known left crossing or earned merchant funds")
			}
			t.Logf("Real intro/left route enters first merchant at%.2fs: ships%d shield%d score%d money%d", float64(update+1)/60, w.Equipment.Lives, w.Equipment.Shield, w.Score, w.Money)
			return
		}
	}
	w := g.Driver.(*worldDriver).world
	t.Fatalf("bounded presentation route did not reach its genuine first merchant: camera%d xy%d,%d lives%d money%d", w.ScrollY, w.Player.X, w.Player.Y, w.Equipment.Lives, w.Money)
}

func TestPresentationPilotCompletesFirstLevelFromDefaultIntroOptional(t *testing.T) {
	if os.Getenv("XENON2_HUMAN_PRESENTATION_CHECK") == "" {
		t.Skip("enable the complete first-level presentation check explicitly")
	}
	g := presentationFrontendGame(t)
	recording := &RecordingGame{Game: g, completeLevel: 1}
	middle, final, guardian := false, false, false
	var previousShop *shopui.State
	for update := 0; update < 60*1200; update++ {
		err := recording.Update()
		if err != nil && !errors.Is(err, ebiten.Termination) {
			t.Fatal(err)
		}
		d, ok := g.Driver.(*worldDriver)
		if !ok || d.session == nil {
			continue
		}
		w := d.world
		if w.Cheats.Enabled() || d.diagnostic {
			t.Fatal("presentation became assisted or diagnostic")
		}
		if w.Level.Number == 1 && w.FirstGuardian != nil && w.FirstGuardian.Defeated {
			guardian = true
		}
		if g.Screen == ShopScreen && g.shop != nil && g.shop != previousShop {
			previousShop = g.shop
			if g.shopFinal {
				final = true
				if !guardian || !w.ExitReady || w.PendingExitDrops != 0 {
					t.Fatal("final merchant bypassed guardian defeat or exit drops")
				}
			} else {
				middle = true
			}
			t.Logf("Merchant final%v at%.2fs ships%d shield%d cash%d", g.shopFinal, float64(update+1)/60, w.Equipment.Lives, w.Equipment.Shield, w.Money)
		}
		if w.Level.Number == 2 {
			if !middle || !final || !guardian || w.GameOver || !g.DemoActive() || !errors.Is(err, ebiten.Termination) {
				t.Fatal("presentation omitted a genuine first-level completion gate")
			}
			t.Logf("Complete first-level presentation at%.2fs: next ships%d shield%d", float64(update+1)/60, w.Equipment.Lives, w.Equipment.Shield)
			return
		}
		if w.GameOver && w.ContinueCredits == 0 {
			t.Fatalf("presentation exhausted recovery at frame%d camera%d: middle%v final%v guardian%v", w.Frame, w.ScrollY, middle, final, guardian)
		}
	}
	w := g.Driver.(*worldDriver).world
	t.Fatalf("bounded full first-level presentation not complete: frame%d camera%d xy%d,%d ships%d shield%d cash%d middle%v final%v guardian%v", w.Frame, w.ScrollY, w.Player.X, w.Player.Y, w.Equipment.Lives, w.Equipment.Shield, w.Money, middle, final, guardian)
}

func TestPresentationPilotCarriesFirstLevelIntoSecondMiddleShopOptional(t *testing.T) {
	if os.Getenv("XENON2_HUMAN_PRESENTATION_CHECK") == "" {
		t.Skip("enable the real carried presentation progression explicitly")
	}
	g := presentationFrontendGame(t)
	firstMiddle, firstFinal, firstGuardian := false, false, false
	var lastShop *shopui.State
	for update := 0; update < 60*900; update++ {
		advanceFrontend(t, g, inputFrame{})
		d, ok := g.Driver.(*worldDriver)
		if !ok || d.session == nil {
			continue
		}
		w := d.world
		if w.Cheats.Enabled() || d.diagnostic {
			t.Fatal("carried presentation became assisted")
		}
		if w.Level.Number == 1 && w.FirstGuardian != nil && w.FirstGuardian.Defeated {
			firstGuardian = true
		}
		if g.Screen == ShopScreen && g.shop != nil && g.shop != lastShop {
			lastShop = g.shop
			if w.Level.Number == 1 {
				if g.shopFinal {
					firstFinal = true
					if !firstGuardian || !w.ExitReady || w.PendingExitDrops != 0 {
						t.Fatal("first final merchant skipped source gates")
					}
				} else {
					firstMiddle = true
				}
			} else if w.Level.Number == 2 && !g.shopFinal {
				if !firstMiddle || !firstFinal || !firstGuardian || w.LevelFinished || w.PendingExitDrops != 0 || w.GameOver || w.Equipment.Lives < 1 {
					t.Fatal("second middle merchant omitted genuine carried completion gates")
				}
				t.Logf("Carried second middle merchant at%.2fs: ships%d shield%d credits%d cash%d", float64(update+1)/60, w.Equipment.Lives, w.Equipment.Shield, w.ContinueCredits, w.Money)
				return
			}
		}
		if w.GameOver && w.ContinueCredits == 0 {
			t.Fatal("ordinary recovery exhausted before the second middle merchant")
		}
	}
	t.Fatal("bounded carried presentation did not defeat the second arena and collect its merchant drops")
}
