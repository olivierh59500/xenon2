package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func TestWaveEntryFromRightChoosesOriginalAnimationAndCollision(t *testing.T) {
	w := testWorld(t)
	w.paths[2] = &visualassets.Path{ID: 2, Commands: []visualassets.PathCommand{{Kind: "origin", X: 352, Y: 120}, {Kind: "curve", Heading: 128, Duration: 100}, {Kind: "end"}}}
	w.Level.Paths.SineTable[192] = -63
	clip := func(name string) visualassets.ActorAnimation {
		return visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: name, Duration: 2}}}
	}
	w.kinds[2] = &visualassets.WaveActor{Kind: 2, Parts: []visualassets.ActorPart{{ResourceTag: 232, MotionMode: "path-entry-edge-frames", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "initial"}}}, EntryAnimations: []visualassets.ActorAnimation{clip("left"), clip("top"), clip("right"), clip("bottom")}}}}
	w.movingSpriteBoxes["right"] = visualassets.CollisionBox{X: 1, Y: -5, Width: 14, Height: 11}
	w.movingSpriteBoxes["bottom"] = visualassets.CollisionBox{X: -5, Y: 2, Width: 10, Height: 13}
	if err := w.spawnWave(visualassets.Wave{EnemyKind: 2, PathID: 2, Count: 1, MotionBudget: 9}); err != nil {
		t.Fatal(err)
	}
	if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
		t.Fatal(err)
	}
	a := w.Actors[0]
	if a.Sprite != "right" || a.animationState.Remaining != 2 || a.Collision != (CollisionRect{Left: 344, Top: 115, Right: 357, Bottom: 125}) {
		t.Fatalf("right-edge entry selected another orientation or collision: position (%v,%v), sprite %s collision %+v", a.X, a.Y, a.Sprite, a.Collision)
	}
}

func TestWaveCurvePublishesSharedWordsAfterInitialPause(t *testing.T) {
	w := testWorld(t)
	w.paths[2] = &visualassets.Path{ID: 2, Commands: []visualassets.PathCommand{{Kind: "origin", X: 20, Y: 20}, {Kind: "pause", Duration: 20}, {Kind: "curve", Heading: 64, AngularVelocity: 32, AngularAcceleration: 3, Duration: 30}, {Kind: "end"}}}
	slot := w.Pool.FreeFirst()
	w.Pool.Slot(slot).Residue = waveConstructorResidueFixture(slot)
	if err := w.spawnWave(visualassets.Wave{EnemyKind: 1, PathID: 2, Count: 1, MotionBudget: 7}); err != nil {
		t.Fatal(err)
	}
	want := waveConstructorResidueFixture(slot)
	for pass := 0; pass < 2; pass++ {
		if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
			t.Fatal(err)
		}
		r := w.Pool.Slot(slot).Residue
		if r.MountOffsetX != want.MountOffsetX || r.MountOffsetY != want.MountOffsetY {
			t.Fatal("an initial pause overwrote untouched shared curve words")
		}
	}
	if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
		t.Fatal(err)
	}
	r := w.Pool.Slot(slot).Residue
	if r.MountOffsetX != 8195 || r.MountOffsetY != 3 {
		t.Fatalf("first curve did not publish its updated velocity/acceleration words: %+v", r)
	}
	w.releaseWorldActor(w.Actors[0].Binding)
	if w.Pool.Slot(slot).Residue != r {
		t.Fatal("releasing the curve owner erased fields retained for the next owner")
	}
}
