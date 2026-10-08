package engine

import "testing"

func thirdFinalEntryPreparationScene(t *testing.T) *World {
	t.Helper()
	// The shared source fixture really destroys both 24 HP last-cannon stages.
	w := path55PreparationScene(t)
	// Earlier formation clearance and this incoming pose are explicit fixtures.
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 240, 0, 256, 256
	w.Player.X, w.Player.Y, w.Player.Inertia = 32, 171, 0
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.cursor = EncounterCursor{MovingHighWater: 241, FixedHighWater: 240}
	if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("original incoming left pose is covered")
	}
	return w
}

func TestOriginalThirdFinalPreparedRouteRetainsShieldThroughFirstFan(t *testing.T) {
	w := thirdFinalEntryPreparationScene(t)
	p := PresentationPilot{PALRefreshes: 3}
	if _, ok := p.thirdFinalEntryPreparation(w); !ok {
		t.Fatal("original checkpoint entry route not admitted")
	}
	before := forecastIsolationDigest(w)
	if _, ok := p.thirdFinalEntryPreparation(w); !ok || before != forecastIsolationDigest(w) {
		t.Fatal("entry route changed live game")
	}
	shield, lives := w.Equipment.Shield, w.Equipment.Lives
	launch, fan := false, false
	for pass := 0; pass < 160; pass++ {
		for range 3 {
			w.AdvancePALTick()
		}
		input := p.NormalInput(w)
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
		launch = launch || w.ThirdFinal.LaunchCount > 0
		for _, shot := range w.Projectiles {
			fan = fan || shot.Atlas == "guardian-parts" && shot.Motion.Speed == 12
		}
		if !w.PlayerAlive || w.Equipment.Lives != lives || w.Equipment.Shield != shield || w.Rewind.Timer != 0 {
			t.Fatalf("source entry safety lost at%d frame%d camera%d HP%d rewind%d", pass, w.Frame, w.ScrollY, w.Equipment.Shield, w.Rewind.Timer)
		}
	}
	if !launch || !fan {
		t.Fatal("source launch and native first fan missing")
	}
}

func TestThirdFinalEntryPreparationRequiresOriginalGates(t *testing.T) {
	var p PresentationPilot
	if _, ok := p.thirdFinalEntryPreparation(nil); ok {
		t.Fatal("nil world admitted")
	}
	w := thirdFinalEntryPreparationScene(t)
	for _, mode := range []string{"missing-patch", "unvisited-formation", "outside"} {
		t.Run(mode, func(t *testing.T) {
			var f WorldForecast
			if err := f.Load(w); err != nil {
				t.Fatal(err)
			}
			v := f.State()
			switch mode {
			case "missing-patch":
				v.Coverage.Map[40*20+8] ^= 1
			case "unvisited-formation":
				v.cursor.MovingHighWater = 577
			case "outside":
				v.ScrollY = 241
			}
			before := forecastIsolationDigest(v)
			var pilot PresentationPilot
			if _, ok := pilot.thirdFinalEntryPreparation(v); ok {
				t.Fatal("ineligible source state admitted")
			}
			if forecastIsolationDigest(v) != before {
				t.Fatal("scope rejection changed source state")
			}
		})
	}
}
