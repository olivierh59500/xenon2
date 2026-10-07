package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestRenderTerrainPreservesPreActorMapAndLiveCollisionState(t *testing.T) {
	w := testWorld(t)
	w.Level.Terrain.Map[12] = 7
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	w.setSecondMapCell(12, 0, 9)
	if w.RenderTerrainMap[12] != 7 || w.Level.Terrain.Map[12] != 9 {
		t.Fatal("actor mutation changed the already drawn map or missed live gameplay")
	}
	buffer := &w.RenderTerrainMap[0]
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.RenderTerrainMap[12] != 9 || &w.RenderTerrainMap[0] != buffer {
		t.Fatal("next presentation did not capture the mutation in its reusable map")
	}
}

func TestCheckpointRenderTerrainStartsAtRetainedMap(t *testing.T) {
	w := testWorld(t)
	w.Level.Terrain.Map[22] = 19
	w.RestartCheckpoint()
	if w.RenderTerrainMap[22] != 19 || &w.RenderTerrainMap[0] == &w.Level.Terrain.Map[0] {
		t.Fatal("checkpoint presentation omitted retained terrain or shared its live buffer")
	}
}

func TestSceneryAnimationChangesLiveMapAfterRenderCapture(t *testing.T) {
	w := testWorld(t)
	variant := &visualassets.FixedTileVariant{Frames: []visualassets.TilePatch{{Columns: 1, Rows: 1, Tiles: []uint16{2}}, {Columns: 1, Rows: 1, Tiles: []uint16{8}}}}
	actor := &WorldActor{Active: true, ActorList: "scenery", fixedHatch: &FixedHatchState{X: 0, WorldY: 4608, Phase: 1}, fixedTileArt: &visualassets.FixedTileKind{}, fixedTileVariant: variant, part: &visualassets.ActorPart{ResourceTag: 240}}
	if err := w.bindWorldActor(actor); err != nil {
		t.Fatal(err)
	}
	w.Actors = append(w.Actors, actor)
	w.Level.Terrain.Map[288*20] = 2
	if err := w.Step(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.Level.Terrain.Map[288*20] != 8 || w.RenderTerrainMap[288*20] != 2 {
		t.Fatal("scenery callback changed its already drawn presentation map")
	}
	if w.ActorRenderTerrainMap[288*20] != 8 {
		t.Fatal("actor materialization mask omitted the preceding updater's terrain mutation")
	}
	if w.ActorRenderScrollY != w.RenderScrollY || w.ScrollY == w.ActorRenderScrollY {
		t.Fatal("actor clipping camera advanced with the next-pass scroll")
	}
}

func TestLateEncounterMapWritesWaitUntilNextActorMask(t *testing.T) {
	w := testWorld(t)
	w.Level.Terrain.Map[18] = 4
	w.captureActorRenderTerrain()
	w.setSecondMapCell(18, 0, 6)
	if w.ActorRenderTerrainMap[18] != 4 || w.Level.Terrain.Map[18] != 6 {
		t.Fatal("late encounter constructor altered the completed draw mask")
	}
}
