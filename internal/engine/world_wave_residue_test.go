package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func TestWaveBirthPublishesSlotStateBeforeFirstUpdate(t *testing.T) {
	w := testWorld(t)
	w.Level.Rules = &visualassets.LevelRules{OrdinaryHealthMultiplier: 1, StrongHealthMultiplier: 2, DefaultEnemyShot: "shot"}
	for slot := w.Pool.FreeFirst(); slot != NoActorSlot; slot = w.Pool.Slot(slot).freeNext {
		w.Pool.Slot(slot).Residue = waveConstructorResidueFixture(slot)
	}
	w.kinds[2] = &visualassets.WaveActor{Kind: 2, Parts: []visualassets.ActorPart{
		{ResourceTag: 208, Score: 50, StrongHealth: true, Linked: true, MotionMode: "path-heading-frames", HeadingFrames: []string{"heading"}, Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "initial", Duration: 3}}}},
		{ResourceTag: 212, Score: 16, StrongHealth: true, Linked: true, MotionMode: "follow-leader", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "body", Duration: 5}}}},
	}}
	w.movingSpriteBoxes["initial"] = visualassets.CollisionBox{Width: 5, Height: 5}
	w.movingSpriteBoxes["heading"] = visualassets.CollisionBox{Width: 11, Height: 11}
	w.WaveBonuses.BeginPass(0)
	first := w.Pool.FreeFirst()
	if err := w.spawnWave(visualassets.Wave{EnemyKind: 2, PathID: 1, Count: 2, Spacing: 132, MotionBudget: 7}); err != nil {
		t.Fatal(err)
	}
	if len(w.Actors) != 4 || w.WaveBonuses.Entries[0] != (WaveBonusEntry{ID: 2, Remaining: 2}) {
		t.Fatal("compound wave did not register two independent members")
	}
	for _, actor := range w.Actors {
		slot := actor.Binding.Slot
		want := waveConstructorResidueFixture(slot)
		want.X, want.Y = int16(actor.X), int16(actor.Y)
		want.XFraction, want.YFraction = 0, 0
		want.Counter, want.Direction, want.HorizontalDriftRemainder = int16(actor.motion.Remaining), 0, 0
		want.Health, want.PowerOrScore, want.StrongHealth = 2, uint16(actor.Score), true
		want.WaveBonusToken, want.MotionBudget = 2, 7
		want.SetFireState(actor.fire.Accumulator, 0)
		want.OwnerSlot = first + ((slot-first)/2)*2
		want.FollowingSlot = NoActorSlot
		if slot%2 == first%2 {
			want.FollowingSlot = slot + 1
			if actor.Sprite != "initial" || actor.Collision.Right != int(actor.X)+4 {
				t.Fatal("birth used a heading frame or its collision before the first callback")
			}
		}
		if actor.Visible || actor.Binding.Residue != want || w.Pool.Slot(slot).Residue != want {
			t.Fatalf("birth slot %d: visible %v, binding %+v pool %+v, want %+v", slot, actor.Visible, actor.Binding.Residue, w.Pool.Slot(slot).Residue, want)
		}
	}
	// Reuse a head before it ever receives a movement callback. Ordinary
	// bullets must inherit the constructor's cleared coordinate fractions.
	head := w.Actors[0]
	head.Active = false
	w.releaseWorldActor(head.Binding)
	w.spawnEnemyShot(100, 90, EnemyShot{Direction: 1, Speed: 6})
	shot := w.Projectiles[0]
	if shot.Binding.Slot != head.Binding.Slot || uint16(shot.Motion.X) != 0 || uint16(shot.Motion.Y) != 0 {
		t.Fatal("immediate wave-slot reuse retained pre-construction fractions")
	}
	if err := w.advanceEnemyShot(shot); err != nil {
		t.Fatal(err)
	}
	if shot.Motion.X != 100<<16+11585*24 || shot.Motion.Y != 90<<16-11585*24+int32(w.ScrollDelta)<<16 {
		t.Fatal("the replacement bullet lost its original fixed-point trajectory")
	}
}

func TestCarrierBirthPublishesRewardAndClearsItsBonusToken(t *testing.T) {
	w := testWorld(t)
	part := visualassets.ActorPart{ResourceTag: 100, MotionMode: "path", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "carrier"}}}}
	w.kinds[0] = &visualassets.WaveActor{Kind: 0, Parts: []visualassets.ActorPart{part}, MotionBudgetOverride: 3, CarriedRewardFromMotionBudget: true}
	slot := w.Pool.FreeFirst()
	w.Pool.Slot(slot).Residue = waveConstructorResidueFixture(slot)
	w.WaveBonuses.BeginPass(0)
	if err := w.spawnWave(visualassets.Wave{EnemyKind: 0, PathID: 1, Count: 1, MotionBudget: 14}); err != nil {
		t.Fatal(err)
	}
	actor := w.Actors[0]
	r := w.Pool.Slot(slot).Residue
	if actor.CarriedReward != 14 || r.VerticalFraction != 14 || actor.motion.Budget != 3 || r.MotionBudget != 3 || actor.WaveToken != 0 || r.WaveBonusToken != 0 || w.WaveBonuses.Entries[0] != (WaveBonusEntry{ID: 1, Remaining: 1}) {
		t.Fatalf("carrier constructor lost its original wrapper fields: actor reward %d, residue %+v cache %+v", actor.CarriedReward, r, w.WaveBonuses)
	}
}
