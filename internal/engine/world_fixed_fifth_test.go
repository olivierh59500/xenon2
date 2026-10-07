package engine

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"xenon2/internal/visualassets"
)

func TestFifthFixedTileNativeDamageOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("local fifth fixed-tile reference not supplied")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "04820138.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	art, _, err := visualassets.DecodeFixedTiles(5, raw)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(root, "fixed-fifth-tiles-damage.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	radialFile, err := os.Open(filepath.Join(root, "fixed-fifth-radial-damage.csv"))
	if err != nil {
		t.Fatal(err)
	}
	radialRows, err := csv.NewReader(radialFile).ReadAll()
	radialFile.Close()
	if err != nil {
		t.Fatal(err)
	}
	rows = append(rows, radialRows[1:]...)
	for _, row := range rows[1:] {
		n := func(i int) int {
			value, err := strconv.Atoi(row[i])
			if err != nil {
				t.Fatal(err)
			}
			return value
		}
		w := testWorld(t)
		w.Level.Number, w.Level.FixedTiles, w.ScrollY = 5, art, 900
		clip := visualassets.NamedActorAnimation{Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "explosion", Duration: 2}}}}
		w.commonAnimations["explosion-small"], w.commonAnimations["explosion-large"] = clip, clip
		position := (1000/16)*20 + 96/16
		w.setSecondMapCell(position%20-1, position/20, 0x867d)
		w.setSecondMapCell(position%20+2, position/20, 0x8687)
		kind := 1
		if n(0) == 1 {
			kind = 3
		} else if n(0) == 2 {
			kind = 4
			w.setSecondMapCell(position%20-1, position/20, 0x8669)
			w.setSecondMapCell(position%20+2, position/20, 0x8673)
		}
		w.spawnFixed(visualassets.FixedEncounter{EnemyKind: kind, X: 104, Y: 1008, Variant: n(1)})
		var actor *WorldActor
		var group []*WorldActor
		for _, a := range w.Actors {
			if a.fifthTile != nil {
				group = append(group, a)
			}
		}
		if len(group) == 0 {
			t.Fatal("missing fifth tile controller")
		}
		actor = group[0]
		if kind == 1 && n(1) == 1 {
			actor = group[2]
		}
		w.damageActor(actor, uint16(n(2)))
		if actor.Health != n(3) || w.Score != n(4) {
			t.Fatalf("damage state health%d score%d native%v", actor.Health, w.Score, row)
		}
		explosions := 0
		for _, a := range w.Actors {
			if a.ActorList == "transient" {
				explosions++
			}
		}
		if explosions != n(5) {
			t.Fatalf("explosions%d native%v", explosions, row)
		}
		for i, a := range group {
			tag := a.part.ResourceTag
			if !a.Active {
				tag = 4
			}
			if tag != n(6+i) {
				t.Fatalf("part%d tag%d native%v", i, tag, row)
			}
		}
		positions := []int{position - 1, position + 19, position, position + 1, position + 20, position + 21, position + 2, position + 22}
		for i, p := range positions {
			if int(w.Level.Terrain.Map[p]) != n(9+i) {
				t.Fatalf("map%d Go%x native%x row%v", p, w.Level.Terrain.Map[p], n(9+i), row)
			}
		}
		if n(5) == 0 && (!actor.Visible || !actor.Flash || actor.Patch == nil) {
			t.Fatal("nonlethal hit lost its one-pass source tile flash")
		}
	}
	if len(rows)-1 != 66 {
		t.Fatal("incomplete damage comparisons")
	}
	t.Logf("Compared %d original fifth tile damage and neighbour-map changes.", len(rows)-1)
}

func TestFifthFixedFamiliesUseOriginalResourcesOptional(t *testing.T) {
	data := originalWorldData(t, 5)
	for _, kind := range []int{1, 3, 4} {
		w, err := NewWorld(data)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, record := range data.Encounters.Fixed {
			if record.EnemyKind == kind {
				w.ScrollY = record.TriggerY
				w.MaximumScrollY = 4607
				w.cursor = RestartEncounterCursor(w.ScrollY)
				w.spawnFixed(record)
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing original fixed family%d", kind)
		}
		var actors []*WorldActor
		for _, actor := range w.Actors {
			if actor.fifthTile != nil {
				actors = append(actors, actor)
			}
		}
		want := 1
		if kind == 1 {
			want = 3
		}
		if len(actors) != want {
			t.Fatalf("kind%d pieces%d want%d", kind, len(actors), want)
		}
		for _, actor := range actors {
			if actor.Binding.EntityID == 0 {
				t.Fatal("tile actor bypassed shared allocation")
			}
		}
		w.InvulnerableFrames = 10000
		for range 100 {
			if err := w.Step(Input{}); err != nil {
				t.Fatal(err)
			}
		}
		if kind == 3 || kind == 4 {
			if len(w.Projectiles) == 0 {
				t.Fatal("source aiming turret never emitted")
			}
			for _, shot := range w.Projectiles {
				if shot.Sprite == "" || shot.Binding.EntityID == 0 {
					t.Fatal("turret shot lacks original art or source allocation")
				}
			}
		}
	}
}
