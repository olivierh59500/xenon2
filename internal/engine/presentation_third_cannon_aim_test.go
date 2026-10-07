package engine

import "testing"

func thirdAimCannonFixture(t *testing.T, stage int) (*World, *WorldActor) {
	t.Helper()
	w, err := NewWorld(playableOriginalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	actor := thirdRouteCannonBirth(t, w, 128, 1696)
	if stage == 1 {
		w.damageActor(actor, 24)
	}
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 1713-stage*60, 0, 2608, 2608
	w.Player.X, w.Player.Y = 152, 176
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.advanceThirdCannon(actor)
	return w, actor
}

func TestPresentationThirdCannonAimIgnoresStaleDrawingHistoryOptional(t *testing.T) {
	for stage := 0; stage <= 1; stage++ {
		w, actor := thirdAimCannonFixture(t, stage)
		if actor.PreviousY != -96 || actor.Y == actor.PreviousY || actor.Collision.Top <= 0 {
			t.Fatal("source constructor/update did not retain the observed drawing history")
		}
		before := forecastDigest(w)
		if !presentationShotOpportunityForMotion(w, MotionInput{}) {
			t.Fatalf("stage%d source weak point directly above the gun was missed: box %+v, position %.0f previous %.0f", stage, actor.Collision, actor.Y, actor.PreviousY)
		}
		if forecastDigest(w) != before {
			t.Fatal("aim prediction changed the real cannon, RNG, collision map or ship")
		}
		// The callback's current world anchor offers a real nine-pixel point
		// ray. Verify it independently through copied source updates.
		state, random := *actor.thirdCannon, w.RandomState()
		hit := false
		for future := 1; future <= 18; future++ {
			event := state.Advance(w.ScrollY-(future-1)*w.BaseScrollStep, w.MaximumScrollY, w.Level.FixedSprites.Third.Cannon, &random)
			y := w.Player.Y - 6 - 9*future
			hit = hit || event.Collision.Intersects(CollisionRect{Left: w.Player.X, Right: w.Player.X, Top: y, Bottom: y})
		}
		if !hit {
			t.Fatal("source callback offered no projectile point intersection")
		}
		w.Player.X = 130
		w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
		if presentationShotOpportunityForMotion(w, MotionInput{}) {
			t.Fatal("a gun outside both source cannon weak points was treated as aligned")
		}
		actor.thirdCannon.Removed = true
		if !actor.thirdCannon.CollisionAt(w.ScrollY).Empty() {
			t.Fatal("removed cannon retained its weak point")
		}
	}
}

// This arranges an original gun encounter and ordinary purchased upgrades; it
// verifies actual projectile callbacks, not a connected campaign victory.
func TestPresentationThirdCannonNormalShotsDefeatBothSourceStagesOptional(t *testing.T) {
	w, actor := thirdAimCannonFixture(t, 0)
	w.ScrollY, w.Player.X, w.Player.Y = 1656, 160, 176
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.cursor = EncounterCursor{MovingHighWater: w.ScrollY, FixedHighWater: w.ScrollY}
	w.Equipment.ApplyItem(ItemAutofire)
	w.Equipment.ApplyItem(ItemAutofire)
	w.Equipment.ApplyItem(ItemPowerup)
	w.Equipment.ApplyItem(ItemPowerup)
	w.advanceThirdCannon(actor)
	pilot := PresentationPilot{}
	score := w.Score
	upper := false
	for pass := 0; pass < 200; pass++ {
		for range 3 {
			w.AdvancePALTick()
		}
		input := Input{Fire: pilot.selectiveFireForMotion(w, MotionInput{})}
		if err := w.Step(input); err != nil {
			t.Fatal(err)
		}
		upper = upper || actor.thirdCannon.Stage == 1
		if !w.PlayerAlive || w.Equipment.Lives != 3 || w.Cheats.Enabled() {
			t.Fatalf("ordinary gun fixture lost its unmodified ship at pass%d: shield%d", pass+1, w.Equipment.Shield)
		}
		if !actor.Active {
			if !upper || !actor.thirdCannon.Removed || w.Score < score+1000 {
				t.Fatal("gun disappeared without both source damage callbacks and rewards")
			}
			t.Logf("Both 24-point cannon stages defeated by normal point shots in %d passes, shield %d", pass+1, w.Equipment.Shield)
			return
		}
	}
	t.Fatalf("ordinary selective fire did not defeat the source cannon: stage%d health%d", actor.thirdCannon.Stage, actor.Health)
}
