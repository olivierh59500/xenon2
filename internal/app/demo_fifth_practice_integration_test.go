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
	if w.Equipment.Mounts[0].Item != engine.ItemLaser || w.Equipment.Mounts[1].Item != engine.ItemNone || w.Equipment.Rear.Tier != 2 {
		t.Fatal("fourth final shop did not earn the left-laser profile")
	}
	minimum := 39
	for update := 0; update < 60*150; update++ {
		advanceFrontend(t, g, inputFrame{})
		d := g.Driver.(*worldDriver)
		w = d.world
		if !g.DemoActive() || d.diagnostic || w.Cheats.Enabled() || !w.PlayerAlive || w.GameOver || w.Equipment.Lives != 3 || w.ContinueCredits != 2 || w.Level.Number != 5 || w.Rewind.Timer != 0 {
			t.Fatalf("ordinary fifth route lost its carried reserve: F%d C%d HP%d", w.Frame, w.ScrollY, w.Equipment.Shield)
		}
		minimum = min(minimum, w.Equipment.Shield)
		if w.Checkpoint.ScrollY == 3424 {
			if w.Frame != 1185 || w.ScrollY != 3423 || minimum != 15 || w.Equipment.Shield != 39 || w.Money != 300 || w.Score != 226650 || w.RandomState() != (engine.RandomState{A: 3819972131, B: 4135342466}) {
				t.Fatalf("source fifth checkpoint differs: F%d C%d HP%d min%d cash%d score%d RNG%+v", w.Frame, w.ScrollY, w.Equipment.Shield, minimum, w.Money, w.Score, w.RandomState())
			}
			t.Logf("Complete intro reaches genuine fifth checkpoint3424 at frame%d: shield39 minimum%d all three ships and two continues", w.Frame, minimum)
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
		if !g.DemoActive() || d.diagnostic || w.Cheats.Enabled() || !w.PlayerAlive || w.GameOver || w.Equipment.Lives != 3 || w.ContinueCredits != 2 || w.Level.Number != 5 || w.Rewind.Timer != 0 {
			t.Fatalf("terrain-defense route changed its earned reserve: F%d C%d HP%d", w.Frame, w.ScrollY, w.Equipment.Shield)
		}
		minimum = min(minimum, w.Equipment.Shield)
		if w.Checkpoint.ScrollY == 2880 {
			if w.Frame != 1729 || w.ScrollY != 2879 || w.Equipment.Shield != 31 || minimum != 31 || w.Money != 300 || w.Score != 233850 || w.RandomState() != (engine.RandomState{A: 2474659900, B: 134657990}) {
				t.Fatalf("native terrain checkpoint differs: F%d C%d HP%d min%d cash%d score%d RNG%+v", w.Frame, w.ScrollY, w.Equipment.Shield, minimum, w.Money, w.Score, w.RandomState())
			}
			t.Logf("Complete intro reaches genuine fifth checkpoint2880 at frame%d with31shield, all three ships and two continues", w.Frame)
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
	minimum := 31
	for update := 0; update < 60*75; update++ {
		advanceFrontend(t, g, inputFrame{})
		d := g.Driver.(*worldDriver)
		w := d.world
		if !g.DemoActive() || d.diagnostic || w.Cheats.Enabled() || !w.PlayerAlive || w.GameOver || w.Equipment.Lives != 3 || w.ContinueCredits != 2 || w.Level.Number != 5 || w.Rewind.Timer != 0 {
			t.Fatalf("middle approach changed its carried reserve: F%d C%d HP%d", w.Frame, w.ScrollY, w.Equipment.Shield)
		}
		minimum = min(minimum, w.Equipment.Shield)
		if w.Checkpoint.ScrollY == 2368 {
			if w.FifthMiddle == nil || w.FifthMiddle.Defeated || w.Frame != 2241 || w.ScrollY != 2367 || w.Equipment.Shield != 31 || minimum != 31 || w.Money != 300 || w.Score != 238350 || w.RandomState() != (engine.RandomState{A: 648954619, B: 661526534}) {
				t.Fatalf("genuine middle admission differs: F%d C%d HP%d min%d cash%d score%d RNG%+v", w.Frame, w.ScrollY, w.Equipment.Shield, minimum, w.Money, w.Score, w.RandomState())
			}
			t.Logf("Complete intro reaches genuine fifth middle at frame%d checkpoint2368, shield31, all three ships and two continues", w.Frame)
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
	minimum := 31
	for update := 0; update < 60*300; update++ {
		advanceFrontend(t, g, inputFrame{})
		d := g.Driver.(*worldDriver)
		w := d.world
		if !g.DemoActive() || d.diagnostic || w.Cheats.Enabled() || !w.PlayerAlive || w.GameOver || w.Equipment.Lives != 3 || w.ContinueCredits != 2 || w.Level.Number != 5 {
			t.Fatalf("fifth middle victory changed earned reserves: F%d HP%d", w.Frame, w.Equipment.Shield)
		}
		minimum = min(minimum, w.Equipment.Shield)
		if w.FifthMiddle != nil && w.FifthMiddle.Defeated && !defeated {
			if w.Frame != 3193 || w.ScrollY != 2313 || w.Equipment.Shield != 3 || w.PendingExitDrops != 10 || w.Score != 248650 || w.LevelFinished {
				t.Fatal("core victory omitted native damage, ten coins or same-stage status")
			}
			defeated = true
		}
		if g.Screen == ShopScreen && !merchant {
			if !defeated || g.shopFinal || w.Frame != 3238 || w.ScrollY != 2268 || w.Equipment.Shield != 3 || w.Money != 1050 || w.PendingExitDrops != 0 || !w.ShopReady || w.ExitReady || w.LevelFinished {
				t.Fatal("real middle merchant omitted native reward drain")
			}
			merchant = true
		}
		if merchant && g.Screen == LevelScreen && !w.Ready && !g.backdropOnly && (g.fade == nil || g.fade.Done) {
			if w.Equipment.Shield != 39 || w.Money != 550 || minimum != 3 || w.Equipment.Side.Item != engine.ItemSideShot || w.Equipment.Side.Tier != 1 || w.Equipment.Rear.Item != engine.ItemNone || w.Equipment.Mounts[0].Item != engine.ItemLaser || w.Equipment.Mounts[0].Tier != 2 || w.Equipment.Primary.Tier != 2 || !w.Equipment.Protection || w.Equipment.FirePeriod != 8 || !w.FifthMiddle.Defeated || w.LevelFinished {
				t.Fatalf("native fifth repair/same-stage return differs: HP%d min%d cash%d", w.Equipment.Shield, minimum, w.Money)
			}
			t.Logf("Current intro defeats fifth middle at3193, collects all750 exit cash by3238, sells Rear/Homing for5500, buys repair/Side/Protection/two power-ups for6000, then returns with three ships, two continues and550 cash; RNG%+v equipment%+v", w.RandomState(), w.Equipment)
			return g
		}
	}
	t.Fatal("bounded earned fifth middle did not return from its genuine merchant")
	return nil
}

func TestPresentationPilotCrossesFifthSecondHalfFromRealMerchantOptional(t *testing.T) {
	if os.Getenv("XENON2_HUMAN_PRESENTATION_CHECK") == "" {
		t.Skip("enable the earned fifth second-half route explicitly")
	}
	verifyPresentationFifthSecondHalfCheckpoint(t)
}

func verifyPresentationFifthSecondHalfCheckpoint(t *testing.T) *Game {
	t.Helper()
	g := verifyPresentationFifthMiddleMerchantReturn(t)
	minimum := 39
	first := false
	for update := 0; update < 60*150; update++ {
		advanceFrontend(t, g, inputFrame{})
		d := g.Driver.(*worldDriver)
		w := d.world
		if !g.DemoActive() || d.diagnostic || w.Cheats.Enabled() || !w.PlayerAlive || w.GameOver || w.Equipment.Lives != 3 || w.ContinueCredits != 2 || w.Level.Number != 5 || w.Rewind.Timer != 0 || !w.FifthMiddle.Defeated {
			t.Fatalf("second-half route changed earned reserves: F%d C%d HP%d", w.Frame, w.ScrollY, w.Equipment.Shield)
		}
		minimum = min(minimum, w.Equipment.Shield)
		if w.Checkpoint.ScrollY == 2032 && !first {
			if w.Frame != 3475 || w.ScrollY != 2031 || w.Equipment.Shield != 39 || w.Score != 249450 || w.RandomState() != (engine.RandomState{A: 3988359816, B: 4115468022}) {
				t.Fatal("first second-half checkpoint differs from native replay")
			}
			first = true
		}
		if w.Checkpoint.ScrollY == 1008 {
			if !first || w.Frame != 4499 || w.ScrollY != 1007 || w.Equipment.Shield != 39 || minimum != 39 || w.Money != 550 || w.Score != 254450 || w.RandomState() != (engine.RandomState{A: 944472629, B: 2223340960}) {
				t.Fatalf("native final-forest checkpoint differs: F%d C%d HP%d min%d score%d RNG%+v", w.Frame, w.ScrollY, w.Equipment.Shield, minimum, w.Score, w.RandomState())
			}
			t.Logf("Complete intro and genuine fifth merchant reach checkpoint1008 at frame%d with all39shield, three ships and two continues", w.Frame)
			return g
		}
	}
	t.Fatal("bounded second-half route never reached the original final-forest checkpoint")
	return nil
}

func TestPresentationPilotReachesFifthFinalFromDefaultIntroOptional(t *testing.T) {
	if os.Getenv("XENON2_HUMAN_PRESENTATION_CHECK") == "" {
		t.Skip("enable the complete earned final approach explicitly")
	}
	verifyPresentationFifthFinalAdmission(t)
}

func verifyPresentationFifthFinalAdmission(t *testing.T) *Game {
	t.Helper()
	g := verifyPresentationFifthSecondHalfCheckpoint(t)
	for update := 0; update < 60*100; update++ {
		advanceFrontend(t, g, inputFrame{})
		d := g.Driver.(*worldDriver)
		w := d.world
		if !g.DemoActive() || d.diagnostic || w.Cheats.Enabled() || !w.PlayerAlive || w.GameOver || w.Equipment.Lives != 3 || w.ContinueCredits != 2 || w.Level.Number != 5 || w.Rewind.Timer != 0 || w.Equipment.Shield != 39 {
			t.Fatalf("final approach lost carried reserves: F%d C%d HP%d", w.Frame, w.ScrollY, w.Equipment.Shield)
		}
		if w.FifthFinal != nil {
			if w.Frame != 5507 || w.ScrollY != 415 || w.Checkpoint.ScrollY != 416 || w.FifthFinal.OuterRemaining != 18 || w.FifthFinal.CoreHealth != 20 || w.FifthFinal.Defeated || w.Score != 280050 || w.Money != 850 || w.RandomState() != (engine.RandomState{A: 1906578826, B: 683871768}) {
				t.Fatalf("original final constructor differs: F%d C%d HP%d score%d RNG%+v", w.Frame, w.ScrollY, w.Equipment.Shield, w.Score, w.RandomState())
			}
			t.Logf("Complete intro reaches genuine fifth final at frame5507 with all39shield, all18defenses and20corehealth, three ships and two continues")
			return g
		}
	}
	t.Fatal("bounded earned final approach never admitted the original guardian")
	return nil
}
