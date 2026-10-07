package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

// Native common terrain creation at 0x345e keeps whole world coordinates and
// fractions; the compound cannon changes phase and health without moving them.
func TestThirdCompoundCannonRetainsWorldPositionAndPhysicalStateOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.MaximumScrollY = 2700, 4607
	slot := w.Pool.FreeFirst()
	w.Pool.Slot(slot).Residue = secondResidueFixture()
	w.spawnThirdFixed(visualassets.FixedEncounter{EnemyKind: 5, X: 104, Y: 2840})
	actor := w.Actors[0]
	if actor.thirdCannon == nil || actor.Binding.Slot != slot {
		t.Fatal("compound cannon did not reclaim the expected source slot")
	}
	want := secondResidueFixture()
	want.X, want.Y = 96, 2832
	want.Counter, want.VerticalVelocity, want.EmitterClock = 0, 0, 0
	want.Health = uint16(w.Level.FixedSprites.Third.Cannon.Health[0])
	check := func() {
		t.Helper()
		want.Counter, want.Health = int16(actor.thirdCannon.Phase), actor.thirdCannon.Health
		want.SetFireState(actor.thirdCannon.FireAccumulator, 0)
		if got := w.Pool.Slot(slot).Residue; got != want {
			t.Fatalf("compound cannon physical state: got %+v want %+v", got, want)
		}
	}
	check()
	for range 32 {
		w.ScrollY--
		w.advanceThirdCannon(actor)
		w.finishActorUpdate(actor)
		check()
	}
	w.damageThirdCannon(actor, actor.thirdCannon.Health)
	w.storeActorResidue(actor)
	if actor.thirdCannon.Stage != 1 || !actor.Active {
		t.Fatal("native first-body destruction did not retain the second stage")
	}
	check()
	w.advanceThirdCannon(actor)
	w.finishActorUpdate(actor)
	check()
	w.damageThirdCannon(actor, actor.thirdCannon.Health)
	if actor.Active || w.Pool.Slot(slot).ResourceTag != 4 {
		t.Fatal("native second-body destruction did not retire the cannon")
	}
	check()
	w.releaseDeadPoolEntries(ActorPoolMoving)
	if w.Pool.Slot(slot).allocated || w.Pool.FreeFirst() != slot {
		t.Fatal("destroyed compound cannon did not release its physical slot")
	}
}
