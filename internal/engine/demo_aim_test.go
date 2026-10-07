package engine

import "testing"

// The aimed shot must originate after ordinary player motion, as the source
// equipment phase runs after movement and moving-actor updates.
func TestPresentationAimUsesNextGunPositionForActualPathHitOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	var waveIndex int = -1
	for index, wave := range w.Level.Encounters.Moving {
		if wave.PathID == 23 && wave.EnemyKind == 1 {
			waveIndex = index
			break
		}
	}
	if waveIndex < 0 {
		t.Fatal("original straight path encounter is missing")
	}
	wave := w.Level.Encounters.Moving[waveIndex]
	wave.Count = 1
	w.Ready, w.MaterializationFrames = false, 0
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = wave.TriggerY, wave.TriggerY, wave.TriggerY
	w.Level.Encounters.Moving, w.Level.Encounters.Fixed = nil, nil
	w.Player.X = 300
	if err := w.spawnWave(wave); err != nil {
		t.Fatal(err)
	}
	actor := w.Actors[0]
	for range 6 {
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	const flight = 6
	predicted, supported := demoActorPrediction(w, actor, flight, w.ScrollY)
	if !supported || !predicted.Active || predicted.Bounds.Top < 4 {
		t.Fatal("original actor did not establish a visible target")
	}
	w.Player.X, w.Player.Y, w.Player.Inertia = predicted.Bounds.Left-3, predicted.Bounds.Top+6+9*flight, 0
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.MaterializationFrames = 0
	motion := MotionInput{Right: true}
	if !presentationShotOpportunityForMotion(w, motion) {
		t.Fatal("next gun position was not recognized as a real intercept")
	}
	before := actor.Health
	if err := w.Step(Input{Motion: motion, Fire: true}); err != nil {
		t.Fatal(err)
	}
	for pass := 1; pass < flight; pass++ {
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if actor.Health != before-1 || !w.PlayerAlive {
		t.Fatalf("predicted shot did not reach its original target through World.Step: health%d->%d alive%v", before, actor.Health, w.PlayerAlive)
	}
}

func TestPresentationBonusInterceptCopiesItsMotion(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y = 160, 160
	bonus := &WorldCollectible{Active: true, X: 112, Y: 120, Motion: CashMotion{X: 112, Y: 120, Mode: 7}}
	before, player, random := bonus.Motion, w.Player, w.RandomState()
	x, y := presentationBonusIntercept(w, bonus)
	if x <= int(bonus.X) || y >= int(bonus.Y) || bonus.Motion != before || w.Player != player || w.RandomState() != random {
		t.Fatal("bonus intercept did not follow its moving reward without changing live state")
	}
	bonus.Motion.Mode = 0
	bonus.Y = 180
	if x, y := presentationBonusIntercept(w, bonus); x != -1 || y != -1 {
		t.Fatal("expired falling reward was treated as a future intercept")
	}
}

func TestPresentationPreparesOriginalWaveWithoutCreatingOrFiringOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	w.Player.X, w.Player.Y = 160, 120
	var goal presentationGoal
	for _, wave := range w.Level.Encounters.Moving {
		w.ScrollY = wave.TriggerY + 24
		w.cursor = RestartEncounterCursor(w.ScrollY)
		goal = presentationPrepareWave(w)
		if goal.arrival != 0 {
			break
		}
	}
	if goal.arrival == 0 {
		t.Fatal("original upcoming formations did not produce a preparatory position")
	}
	actors, frame, player, random, pool := len(w.Actors), w.Frame, w.Player, w.RandomState(), *w.Pool
	if !goal.valid(w) || presentationShotOpportunity(w) {
		t.Fatal("preparing an unvisited encounter caused premature firing")
	}
	if len(w.Actors) != actors || w.Frame != frame || w.Player != player || w.RandomState() != random || *w.Pool != pool {
		t.Fatal("preparing an encounter changed the live game")
	}
	w.ScrollY = goal.arrival
	if goal.valid(w) {
		t.Fatal("preparatory goal survived its ordinary encounter admission")
	}
}
