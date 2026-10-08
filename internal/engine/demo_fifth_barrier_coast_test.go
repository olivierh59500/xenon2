package engine

import "testing"

func fifthNarrowPostScene(t testing.TB) (*World, *WorldActor) {
	t.Helper()
	w, err := NewWorld(playableOriginalWorldData(t, 5))
	if err != nil {
		t.Fatal(err)
	}
	w.Equipment.ApplyItem(ItemSpeedup)
	w.Equipment.ApplyItem(ItemSpeedup)
	w.Ready, w.MaterializationFrames = false, 0
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 4106, 4112, 4112
	w.Player.X, w.Player.Y, w.Player.SpeedTier = 75, 176, 2
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.cursor = RestartEncounterCursor(w.ScrollY)
	for _, record := range w.Level.Encounters.Fixed {
		if record.EnemyKind == 1 && record.TriggerY == 4112 {
			w.spawnFixed(record)
			break
		}
	}
	if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
		t.Fatal(err)
	}
	post := fifthBarrierTarget(w)
	if post == nil || post.fifthTile.Part != 0 || post.Health != 10 || post.Collision.Left != 83 || post.Collision.Right != 93 || w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatalf("original narrow post or clear approach missing: post%+v", post)
	}
	return w, post
}

func TestFifthNarrowPostAdmitsOneImpulseAndReleaseOptional(t *testing.T) {
	w, post := fifthNarrowPostScene(t)
	before := forecastIsolationDigest(w)
	p := DemoPilot{}
	input, handled := p.FifthBarrierInput(w)
	if !handled || !input.Motion.Right || input.Motion.Left || forecastIsolationDigest(w) != before {
		t.Fatalf("narrow post rejected the safe right impulse: %+v handled%v", input, handled)
	}
	if err := w.Step(input); err != nil {
		t.Fatal(err)
	}
	next, handled := p.FifthBarrierInput(w)
	if !handled || next.Motion.Right || next.Motion.Left {
		t.Fatalf("post alignment did not release the one-pass impulse: %+v", next)
	}
	if err := w.Step(next); err != nil {
		t.Fatal(err)
	}
	if w.Player.X < post.Collision.Left || w.Player.X > post.Collision.Right || w.Player.Inertia != 0 || !w.PlayerAlive || w.Equipment.Shield != 39 || w.Rewind.Timer != 0 {
		t.Fatalf("ordinary impulse/release did not stop in the original firing lane: P%+v HP%d rewind%d", w.Player, w.Equipment.Shield, w.Rewind.Timer)
	}
}
