package engine

import "testing"

// This isolated source scene removes no future encounter and preserves native
// cannon/formation HP. Earlier middle-arena completion and the incoming pose
// are explicit fixtures; this is not a carried campaign claim.
func path55PreparationScene(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(playableOriginalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	w.ThirdMiddle = &ThirdGuardianState{Defeated: true}
	w.Ready, w.MaterializationFrames = false, 0
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 673, 0, 689, 689
	w.Player.X, w.Player.Y = 169, 107
	w.Equipment.SpeedTier = 2
	for _, item := range []Item{ItemPowerup, ItemCannon, ItemRearShot, ItemAutofire, ItemAutofire} {
		w.Equipment.ApplyItem(item)
	}
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.cursor = EncounterCursor{MovingHighWater: 674, FixedHighWater: 673}
	for _, record := range w.Level.Encounters.Fixed {
		if record.EnemyKind == 5 && record.X-8 == 128 && record.Y-8 == 640 {
			w.spawnFixed(record)
		}
	}
	var cannon *WorldActor
	for _, actor := range w.Actors {
		if actor.thirdCannon != nil && actor.thirdCannon.X == 128 && actor.thirdCannon.WorldY == 640 {
			cannon = actor
		}
	}
	if cannon == nil {
		t.Fatal("original last cannon factory is missing")
	}
	for pass := 0; pass < 80 && cannon.Active; pass++ {
		bounds := cannon.thirdCannon.CollisionAt(w.ScrollY)
		if bounds.Top >= 0 && bounds.Top < 192 {
			fifthBoundaryShot(t, w, bounds.Left+1, bounds.Top+1)
		}
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if cannon.Active || !cannon.thirdCannon.Removed || w.Score != 1000 {
		t.Fatalf("native bullets failed to destroy both original cannon stages: stage%d health%d score%d camera%d", cannon.thirdCannon.Stage, cannon.thirdCannon.Health, w.Score, w.ScrollY)
	}
	// Arrange the known incoming approach pose before camera608. The real
	// source destruction above supplies its changed terrain; no map is cleared.
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 608, 624, 624
	w.Player.X, w.Player.Y, w.Player.Inertia = 140, 126, 0
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.cursor = EncounterCursor{MovingHighWater: 609, FixedHighWater: 608}
	if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("original approach fixture is not clear after real cannon destruction")
	}
	return w
}

func TestOriginalPath55PreparationSingleCandidate(t *testing.T) {
	start := path55PreparationScene(t)
	var copy WorldForecast
	if err := copy.Load(start); err != nil {
		t.Fatal(err)
	}
	var health [2]int
	for index, fixture := range []struct {
		name    string
		world   *World
		prepare bool
	}{{"current", start, false}, {"prepare160100", copy.State(), true}} {
		t.Run(fixture.name, func(t *testing.T) {
			w := fixture.world
			p := PresentationPilot{PALRefreshes: 3}
			born, maxParts, clearFrame := false, 0, uint64(0)
			lives := w.Equipment.Lives
			for pass := 0; pass < 160 && w.PlayerAlive; pass++ {
				for range 3 {
					w.AdvancePALTick()
				}
				input := Input{}
				if fixture.prepare {
					input = p.NormalInput(w)
				} else {
					input = p.normalInputWithoutPath55Preparation(w)
				}
				active := 0
				for _, actor := range w.Actors {
					if actor.Active && actor.path != nil && actor.path.ID == 55 && actor.part.ResourceTag == 228 {
						active++
					}
				}
				born, maxParts = born || active > 0, max(maxParts, active)
				before := w.Equipment.Shield
				if err := w.Step(input); err != nil {
					t.Fatal(err)
				}
				if w.Equipment.Shield < before {
					t.Logf("damage frame%d camera%d xy%d,%d HP%d->%d rewind%d input%+v", w.Frame, w.ScrollY, w.Player.X, w.Player.Y, before, w.Equipment.Shield, w.Rewind.Timer, input)
				}
				if born && active == 0 {
					if clearFrame == 0 {
						clearFrame = w.Frame
					}
					if w.Frame >= clearFrame+32 {
						break
					}
				}
			}
			t.Logf("end camera%d xy%d,%d HP%d alive%v born%v maxParts%d rewind%d", w.ScrollY, w.Player.X, w.Player.Y, w.Equipment.Shield, w.PlayerAlive, born, maxParts, w.Rewind.Timer)
			if !born || maxParts != 7 {
				t.Fatal("source original seven-part formation did not enter intact")
			}
			if !w.PlayerAlive || w.Equipment.Lives != lives || clearFrame == 0 || w.Frame < clearFrame+32 {
				t.Fatal("source comparison failed to survive thirty-two passes after formation clearance")
			}
			health[index] = w.Equipment.Shield
		})
	}
	if health[1] <= health[0] {
		t.Fatalf("single preparation candidate did not preserve more shield: current%d prepared%d", health[0], health[1])
	}
}

func TestThirdPath55PreparationRequiresOriginalDestroyedPatchAndPendingWave(t *testing.T) {
	if _, ok := thirdPath55Preparation(nil); ok {
		t.Fatal("nil world started preparation")
	}
	w := path55PreparationScene(t)
	before := forecastIsolationDigest(w)
	if _, ok := thirdPath55Preparation(w); !ok {
		t.Fatal("original pending formation did not admit preparation")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("preparation changed source state")
	}
	for _, name := range []string{"before-cannon", "missing-patch", "expired-wave"} {
		t.Run(name, func(t *testing.T) {
			var forecast WorldForecast
			if err := forecast.Load(w); err != nil {
				t.Fatal(err)
			}
			v := forecast.State()
			switch name {
			case "before-cannon":
				v.cursor.FixedHighWater = 737
			case "missing-patch":
				v.Coverage.Map[40*20+8] ^= 1
			case "expired-wave":
				v.cursor.MovingHighWater = 576
			}
			before := forecastIsolationDigest(v)
			if _, ok := thirdPath55Preparation(v); ok {
				t.Fatal("ineligible source state started preparation")
			}
			if forecastIsolationDigest(v) != before {
				t.Fatal("scope rejection changed source state")
			}
		})
	}
}
