package app

import (
	"os"
	"testing"

	"xenon2/internal/engine"
)

func TestPresentationPilotCrossesFifthLauncherCorridorFromDefaultIntroOptional(t *testing.T) {
	if os.Getenv("XENON2_HUMAN_PRESENTATION_CHECK") == "" {
		t.Skip("enable the complete intro and fifth launcher route explicitly")
	}
	verifyPresentationFifthLauncherCheckpoint(t)
}

func verifyPresentationFifthLauncherCheckpoint(t *testing.T) *Game {
	t.Helper()
	g := verifyPresentationFirstFourLevelsFromDefaultIntro(t)
	w := g.Driver.(*worldDriver).world
	if w.Equipment.Mounts[0].Item != engine.ItemLaser || w.Equipment.Mounts[1].Item != engine.ItemNone || w.Equipment.Rear.Tier != 1 {
		t.Fatal("fourth final shop did not earn the left-laser profile")
	}
	minimum := 39
	for update := 0; update < 60*150; update++ {
		advanceFrontend(t, g, inputFrame{})
		d := g.Driver.(*worldDriver)
		w = d.world
		if !g.DemoActive() || d.diagnostic || w.Cheats.Enabled() || !w.PlayerAlive || w.GameOver || w.Equipment.Lives != 1 || w.ContinueCredits != 2 || w.Level.Number != 5 || w.Rewind.Timer != 0 {
			t.Fatalf("ordinary fifth route lost its carried reserve: F%d C%d HP%d", w.Frame, w.ScrollY, w.Equipment.Shield)
		}
		minimum = min(minimum, w.Equipment.Shield)
		if w.Checkpoint.ScrollY == 3424 {
			if w.Frame != 1185 || w.ScrollY != 3423 || minimum != 23 || w.Equipment.Shield != 39 || w.Money != 150 || w.Score != 197280 || w.RandomState() != (engine.RandomState{A: 678879552, B: 2328483032}) {
				t.Fatalf("source fifth checkpoint differs: F%d C%d HP%d min%d cash%d score%d RNG%+v", w.Frame, w.ScrollY, w.Equipment.Shield, minimum, w.Money, w.Score, w.RandomState())
			}
			t.Logf("Complete intro reaches genuine fifth checkpoint3424 at frame%d: shield39 minimum%d same ship and two continues", w.Frame, minimum)
			return g
		}
	}
	t.Fatal("bounded complete-intro fifth route never reached the original checkpoint")
	return nil
}

func TestPresentationPilotCrossesFifthTerrainDefensesFromDefaultIntroOptional(t *testing.T) {
	if os.Getenv("XENON2_HUMAN_PRESENTATION_CHECK") == "" {
		t.Skip("enable the carried fifth terrain-defense route explicitly")
	}
	verifyPresentationFifthTerrainCheckpoint(t)
}

func verifyPresentationFifthTerrainCheckpoint(t *testing.T) *Game {
	t.Helper()
	g := verifyPresentationFifthLauncherCheckpoint(t)
	minimum := 39
	for update := 0; update < 60*75; update++ {
		advanceFrontend(t, g, inputFrame{})
		d := g.Driver.(*worldDriver)
		w := d.world
		if !g.DemoActive() || d.diagnostic || w.Cheats.Enabled() || !w.PlayerAlive || w.GameOver || w.Equipment.Lives != 1 || w.ContinueCredits != 2 || w.Level.Number != 5 || w.Rewind.Timer != 0 {
			t.Fatalf("terrain-defense route changed its earned reserve: F%d C%d HP%d", w.Frame, w.ScrollY, w.Equipment.Shield)
		}
		minimum = min(minimum, w.Equipment.Shield)
		if w.Checkpoint.ScrollY == 2880 {
			if w.Frame != 1729 || w.ScrollY != 2879 || w.Equipment.Shield != 23 || minimum != 23 || w.Money != 150 || w.Score != 203330 || w.RandomState() != (engine.RandomState{A: 3281101092, B: 3188931690}) {
				t.Fatalf("native terrain checkpoint differs: F%d C%d HP%d min%d cash%d score%d RNG%+v", w.Frame, w.ScrollY, w.Equipment.Shield, minimum, w.Money, w.Score, w.RandomState())
			}
			t.Logf("Complete intro reaches genuine fifth checkpoint2880 at frame%d with23shield, same ship and two continues", w.Frame)
			return g
		}
	}
	t.Fatal("bounded fifth terrain route did not reach its original checkpoint")
	return nil
}

func TestPresentationPilotReachesFifthMiddleGuardianFromDefaultIntroOptional(t *testing.T) {
	if os.Getenv("XENON2_HUMAN_PRESENTATION_CHECK") == "" {
		t.Skip("enable the complete fifth guardian approach explicitly")
	}
	verifyPresentationFifthMiddleAdmission(t)
}

func verifyPresentationFifthMiddleAdmission(t *testing.T) *Game {
	t.Helper()
	g := verifyPresentationFifthTerrainCheckpoint(t)
	minimum := 23
	for update := 0; update < 60*75; update++ {
		advanceFrontend(t, g, inputFrame{})
		d := g.Driver.(*worldDriver)
		w := d.world
		if !g.DemoActive() || d.diagnostic || w.Cheats.Enabled() || !w.PlayerAlive || w.GameOver || w.Equipment.Lives != 1 || w.ContinueCredits != 2 || w.Level.Number != 5 || w.Rewind.Timer != 0 {
			t.Fatalf("middle approach changed its carried reserve: F%d C%d HP%d", w.Frame, w.ScrollY, w.Equipment.Shield)
		}
		minimum = min(minimum, w.Equipment.Shield)
		if w.Checkpoint.ScrollY == 2368 {
			if w.FifthMiddle == nil || w.FifthMiddle.Defeated || w.Frame != 2241 || w.ScrollY != 2367 || w.Equipment.Shield != 23 || minimum != 23 || w.Money != 200 || w.Score != 206630 || w.RandomState() != (engine.RandomState{A: 168595658, B: 2245750478}) {
				t.Fatalf("genuine middle admission differs: F%d C%d HP%d min%d cash%d score%d RNG%+v", w.Frame, w.ScrollY, w.Equipment.Shield, minimum, w.Money, w.Score, w.RandomState())
			}
			t.Logf("Complete intro reaches genuine fifth middle at frame%d checkpoint2368, shield23, same ship and two continues", w.Frame)
			return g
		}
	}
	t.Fatal("bounded fifth approach never admitted the original middle guardian")
	return nil
}

func TestPresentationPilotDefeatsFifthMiddleAndRepairsThroughRealMerchantOptional(t *testing.T) {
	if os.Getenv("XENON2_HUMAN_PRESENTATION_CHECK") == "" {
		t.Skip("enable the complete earned fifth middle victory explicitly")
	}
	verifyPresentationFifthMiddleMerchantReturn(t)
}

func verifyPresentationFifthMiddleMerchantReturn(t *testing.T) *Game {
	t.Helper()
	g := verifyPresentationFifthMiddleAdmission(t)
	defeated, merchant := false, false
	minimum := 23
	for update := 0; update < 60*300; update++ {
		advanceFrontend(t, g, inputFrame{})
		d := g.Driver.(*worldDriver)
		w := d.world
		if !g.DemoActive() || d.diagnostic || w.Cheats.Enabled() || !w.PlayerAlive || w.GameOver || w.Equipment.Lives != 1 || w.ContinueCredits != 2 || w.Level.Number != 5 {
			t.Fatalf("fifth middle victory changed earned reserves: F%d HP%d", w.Frame, w.Equipment.Shield)
		}
		minimum = min(minimum, w.Equipment.Shield)
		if w.FifthMiddle != nil && w.FifthMiddle.Defeated && !defeated {
			if w.Frame != 3087 || w.ScrollY != 2314 || w.Equipment.Shield != 7 || w.PendingExitDrops != 10 || w.Score != 215930 || w.LevelFinished {
				t.Fatal("core victory omitted native damage, ten coins or same-stage status")
			}
			defeated = true
		}
		if g.Screen == ShopScreen && !merchant {
			if !defeated || g.shopFinal || w.Frame != 3195 || w.ScrollY != 2206 || w.Equipment.Shield != 7 || w.Money != 500 || w.PendingExitDrops != 0 || !w.ShopReady || w.ExitReady || w.LevelFinished {
				t.Fatal("real middle merchant omitted native reward drain")
			}
			merchant = true
		}
		if merchant && g.Screen == LevelScreen && !w.Ready && !g.backdropOnly && (g.fade == nil || g.fade.Done) {
			if w.Equipment.Shield != 39 || w.Money != 0 || minimum != 7 || !w.FifthMiddle.Defeated || w.LevelFinished {
				t.Fatalf("native fifth repair/same-stage return differs: HP%d min%d cash%d", w.Equipment.Shield, minimum, w.Money)
			}
			t.Logf("Complete intro defeats fifth middle at3087, drains ten coins at3195, buys native500cost full repair and returns to fifth stage with same ship and two continues")
			return g
		}
	}
	t.Fatal("bounded earned fifth middle did not return from its genuine merchant")
	return nil
}
