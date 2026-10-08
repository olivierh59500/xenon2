package engine

import "testing"

func TestPresentationTargetsOriginalFifthTerrainTurretsOptional(t *testing.T) {
	for _, kind := range []int{3, 4, 9} {
		w, err := NewWorld(playableOriginalWorldData(t, 5))
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, record := range w.Level.Encounters.Fixed {
			if record.EnemyKind != kind {
				continue
			}
			w.ScrollY, w.MaximumScrollY = record.Y-8-64, record.TriggerY
			w.spawnFixed(record)
			found = true
			break
		}
		if !found || len(w.Actors) != 1 {
			t.Fatalf("source turret kind%d missing", kind)
		}
		a := w.Actors[0]
		w.advanceFifthTile(a)
		if a.Visible || a.fifthTile == nil || a.Collision.Empty() || a.Health <= 1 {
			t.Fatal("original turret fixture lost its terrain rendering or health")
		}
		before := forecastIsolationDigest(w)
		bounds, ok := presentationTargetBounds(w, a)
		if !ok || bounds != a.Collision {
			t.Fatalf("source terrain turret kind%d was invisible to targeting", kind)
		}
		if forecastIsolationDigest(w) != before {
			t.Fatal("recognizing native collider changed its world")
		}
		health := a.Health
		shot := &WorldSmallShot{Active: true, Shot: SmallShot{X: (bounds.Left + bounds.Right) / 2, Y: bounds.Top + 9, VelocityY: -9, Damage: 1}}
		w.advanceSmallShot(shot)
		if shot.Active || a.Health != health-1 || !a.Visible {
			t.Fatalf("native ordinary shot did not damage recognized kind%d: HP%d->%d", kind, health, a.Health)
		}
		w.advanceFifthTile(a)
		if a.Visible {
			t.Fatal("native flash did not return to terrain rendering")
		}
		if _, ok := presentationTargetBounds(w, a); !ok {
			t.Fatal("target vanished after its native damage flash")
		}
	}
}
