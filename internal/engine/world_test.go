package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func testWorld(t *testing.T) *World {
	t.Helper()
	p := visualassets.Path{ID: 1, Commands: []visualassets.PathCommand{
		{Kind: "origin", X: 20, Y: -10}, {Kind: "curve", Heading: 64, Duration: 100}, {Kind: "end"},
	}}
	paths := &visualassets.Paths{Paths: []visualassets.Path{p}}
	paths.SineTable[64] = 64
	a := visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "enemy", Duration: 0}}}
	w, err := NewWorld(LevelData{Number: 1,
		Terrain: &visualassets.Terrain{Columns: 20, Rows: 300, TileSize: 16, Map: make([]uint16, 6000)}, Paths: paths,
		Actors:     &visualassets.Actors{Kinds: []visualassets.WaveActor{{Kind: 1, Parts: []visualassets.ActorPart{{Animation: a, MotionMode: "path"}}}}},
		Encounters: &visualassets.Encounters{Moving: []visualassets.Wave{{TriggerY: 4608, EnemyKind: 1, PathID: 1, Count: 2, Spacing: 132, MotionBudget: 7}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestWorldActivationMovesOnFollowingPass(t *testing.T) {
	w := testWorld(t)
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if len(w.Actors) != 2 || w.Actors[0].X != 52 || w.Actors[1].X != 20 || w.Actors[0].Y != -10 {
		t.Fatalf("spawn order or untouched initial positions differ: %+v", w.Actors)
	}
	if w.ScrollY != 4607 {
		t.Fatalf("scroll after first pass=%d", w.ScrollY)
	}
	if w.Actors[0].Visible || w.RenderScrollY != 4608 {
		t.Fatal("late spawn and next-pass scroll must not change the displayed pass")
	}
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.Actors[0].Y != -3 || w.Actors[1].Y != -3 || w.Actors[0].PreviousY != -10 {
		t.Fatalf("spawned actor did not advance exactly seven path substeps")
	}
	if !w.Actors[0].Visible || w.RenderScrollY != 4607 {
		t.Fatal("existing actors must draw against this pass's terrain position")
	}
}

func TestWorldDrawCadenceCannotAlterSimulation(t *testing.T) {
	first, second := testWorld(t), testWorld(t)
	for range 25 {
		if err := first.Step(Input{Motion: MotionInput{Right: true}}); err != nil {
			t.Fatal(err)
		}
	}
	c := NewFrameClock(25, 60)
	for range 60 {
		for steps := c.Advance(); steps > 0; steps-- {
			if err := second.Step(Input{Motion: MotionInput{Right: true}}); err != nil {
				t.Fatal(err)
			}
		}
	}
	if first.Player != second.Player || first.ScrollY != second.ScrollY || first.Frame != second.Frame || first.random != second.random {
		t.Fatal("display cadence changed simulation state")
	}
}
