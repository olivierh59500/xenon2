package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

// This isolates the captured stream0 members. It does not reconstruct all
// combat deaths in the campaign or claim that Right is collision-free.
func originalSecondThreeMemberContactFixture(t testing.TB, callbacks int) (*World, [3]*WorldActor) {
	t.Helper()
	w := originalSecondClosedLeftNodeFixture(t)
	for _, a := range w.Actors {
		if a.secondSegment != nil {
			a.Active = false
		}
	}
	w.ScrollDelta = 0
	w.secondScheduler.DefenseFlags = 1
	launch := w.secondWaveArt.Launches[10]
	var result [3]*WorldActor
	for i, partIndex := range []int{0, 1, 4} {
		part := &w.secondWaveArt.Components[partIndex]
		state, err := NewSecondDefenseSegment(launch, *part, 0, 2528, 1)
		if err != nil {
			t.Fatal(err)
		}
		event := SecondDefenseSegmentEvents{}
		gates, random := w.secondGateCounters, w.RandomState()
		for range callbacks {
			event, err = state.Advance(&launch.Path, &w.Level.Paths.SineTable, 0, &gates, &random)
			if err != nil {
				t.Fatal(err)
			}
		}
		x, y := int(state.Motion.X>>16), int(state.Motion.Y>>16)
		sprite := part.HeadingFrames[event.HeadingFrame]
		a := &WorldActor{Active: true, Visible: true, ActorList: "moving", Atlas: "guardian-parts", Health: part.Health, Score: part.Score, Sprite: sprite, X: float64(x), Y: float64(y), PreviousX: float64(x), PreviousY: float64(y), secondSegment: &state, secondPart: part, path: &launch.Path, part: &visualassets.ActorPart{ResourceTag: part.ResourceTag, DamageMode: "second-defense-segment"}, Collision: ActorCollisionRect(w.movingSpriteBoxes[sprite], x, y)}
		if err := w.bindWorldActor(a); err != nil {
			t.Fatal(err)
		}
		w.Actors = append([]*WorldActor{a}, w.Actors...)
		result[i] = a
	}
	w.Frame = 2138
	return w, result
}

func TestSecondTwoOverlapsUseOneSourceContactCallback(t *testing.T) {
	w, a := originalSecondThreeMemberContactFixture(t, 45)
	w.Player = PlayerMotionState{X: 141, Y: 51, Inertia: 2, SpeedTier: 2}
	w.Rewind = NewTerrainRewind(2528, 141, 51)
	if a[1].Collision != (CollisionRect{Left: 139, Top: 33, Right: 150, Bottom: 44}) || a[0].Collision != (CollisionRect{Left: 147, Top: 39, Right: 158, Bottom: 50}) {
		t.Fatal("original body/head callbacks do not match actual row0")
	}
	prefix := thirdMiddlePlayerBounds(w, w.Player)
	if prefix != (CollisionRect{Left: 132, Top: 41, Right: 149, Bottom: 61}) {
		t.Fatalf("original right-bank prefix differs: %+v", prefix)
	}
	count := 0
	for _, actor := range a {
		if prefix.Intersects(actor.Collision) {
			count++
		}
	}
	if count != 2 {
		t.Fatal("fixture no longer has the captured two simultaneous overlaps")
	}
	beforeRandom := w.RandomState()
	beforeScore := w.Score
	expectedRandom := beforeRandom
	expectedRandom.Next()
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if a[1].Score != 50 || w.Equipment.Shield != 31 || !w.PlayerAlive || a[1].Active || !a[0].Active || !a[2].Active || w.secondScheduler.DefenseFlags != 1 || w.Score != beforeScore+50 || w.RandomState() != expectedRandom {
		t.Fatalf("first-contact callback differs: shield%d body%v head%v other%v flags%d score%d random%+v expected%+v", w.Equipment.Shield, a[1].Active, a[0].Active, a[2].Active, w.secondScheduler.DefenseFlags, w.Score, w.RandomState(), expectedRandom)
	}
	foundHead := false
	for _, id := range w.Pool.EntityIDs(ActorPoolMoving, nil) {
		if id == a[1].ID {
			t.Fatal("contacted body survived the moving-list removal phase")
		}
		foundHead = foundHead || id == a[0].ID
	}
	if !foundHead || w.Pool.Slot(a[0].Binding.Slot).ResourceTag != 276 {
		t.Fatal("second overlapping head did not retain its native moving-list binding")
	}
}
