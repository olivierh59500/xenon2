package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func TestPresentationTargetsOriginalFifthBarrierPostsAndOpensMapOptional(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 5))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	var parts [3]*WorldActor
	for _, actor := range w.Actors {
		if actor.fifthTile != nil && actor.fixedTileArt != nil && actor.fixedTileArt.Kind == 1 {
			parts[actor.fifthTile.Part] = actor
		}
	}
	if parts[0] == nil || parts[1] == nil || parts[2] == nil {
		t.Fatal("original encounter stream did not create the first barrier")
	}
	// Scope this to the original barrier and ordinary shot callbacks. The map,
	// health, equipment, collision prefixes and source damage stay unchanged.
	w.Level.Encounters = &visualassets.Encounters{}
	for range 6 {
		if err := w.Step(Input{Motion: MotionInput{Right: true}}); err != nil {
			t.Fatal(err)
		}
	}
	for range 6 {
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if w.Player.X != 184 || !w.PlayerAlive || w.Equipment.Shield != 39 {
		t.Fatal("ordinary controls did not align with the original clear post lane")
	}
	before := forecastDigest(w)
	for index, actor := range parts {
		if actor.Visible || actor.Collision.Empty() {
			t.Fatal("terrain-rendered barrier lost its actual collision rectangle")
		}
		bounds, target := presentationTargetBounds(w, actor)
		if target != (index != 1) || target && bounds != actor.Collision {
			t.Errorf("barrier part%d target%v: only the two damageable posts are aim targets", index, target)
		}
	}
	if !presentationShotOpportunity(w) {
		t.Error("aligned original post did not admit an ordinary forward shot")
	}
	if forecastDigest(w) != before {
		t.Fatal("post targeting or copied shot aim changed the live world")
	}
	post := parts[0]
	column, row := post.fifthTile.X/16, post.fifthTile.WorldY/16
	patch := post.fixedTileArt.Variants[0].Destroyed
	initial := post.Health
	damaged := false
	for pass := 0; pass < 80 && post.Active; pass++ {
		for range 3 {
			w.AdvancePALTick()
		}
		if err := w.Step(Input{Fire: pass%2 == 0}); err != nil {
			t.Fatal(err)
		}
		damaged = damaged || post.Health < initial
	}
	if !damaged || post.Active || w.Score != 200 || !w.PlayerAlive || w.Equipment.Shield != 39 || w.Equipment.Lives != 3 {
		t.Fatalf("ordinary post shots did not open the barrier: damaged%v active%v score%d alive%v shield%d lives%d", damaged, post.Active, w.Score, w.PlayerAlive, w.Equipment.Shield, w.Equipment.Lives)
	}
	for _, actor := range parts {
		if actor.Active {
			t.Fatal("destroyed post left a linked barrier actor active")
		}
	}
	for y := range patch.Rows {
		for x := range patch.Columns {
			cell := (row+y)*w.Level.Terrain.Columns + column + x
			if w.Level.Terrain.Map[cell] != patch.Tiles[y*patch.Columns+x] || w.Coverage.Map[cell] != patch.Tiles[y*patch.Columns+x] {
				t.Fatal("post destruction did not publish the original open map patch and coverage")
			}
		}
	}
}
