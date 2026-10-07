package engine

import (
	"os"
	"path/filepath"
	"testing"

	"xenon2/internal/visualassets"
)

func secondResidueFixture() ActorResidue {
	return ActorResidue{XFraction: 0x1234, YFraction: 0x5678, Counter: 71, Direction: 81, VerticalFraction: 91, Health: 51, StrongHealth: true, PowerOrScore: 101, WaveBonusToken: 111, EmitterClock: 0xa55a}
}

func TestSecondSpecializedConstructorResidueNativeOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("local second-level constructors not supplied")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", "00FA00FE.decoded"))
	if err != nil {
		t.Fatal(err)
	}
	terrain, err := visualassets.DecodeTerrain(raw)
	if err != nil {
		t.Fatal(err)
	}
	bank, err := visualassets.DecodeFixedSprites(2, raw, terrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	cases := 0
	nativeCombatRows(t, "second-specialized-residue-trace.csv", func(v []int64) {
		w, err := NewWorld(originalWorldData(t, 2))
		if err != nil {
			t.Fatal(err)
		}
		w.ScrollY = 800
		for index := w.Pool.FreeFirst(); index != NoActorSlot; index = w.Pool.Slot(index).freeNext {
			w.Pool.Slot(index).Residue = secondResidueFixture()
		}
		var actor *WorldActor
		switch v[0] {
		case 0:
			w.spawnFixedHatch(visualassets.FixedEncounter{EnemyKind: 2, Variant: int(v[1]), X: 108, Y: 1008})
			actor = w.Actors[0]
		case 1:
			w.spawnFixedPod(visualassets.FixedEncounter{EnemyKind: 4 + int(v[1]), X: 108, Y: 1008})
			actor = w.Actors[0]
		case 2:
			w.spawnHatchCreatures(116, 216)
			actor = w.Actors[7-int(v[2])]
		case 3:
			w.spawnPodCreature(116, 216, int(v[1]))
			actor = w.Actors[0]
		}
		r := actor.Binding.Residue
		if actor.part.ResourceTag != int(v[3]) || r.X != int16(v[4]) || r.Y != int16(v[5]) || r.XFraction != uint16(v[6]) || r.YFraction != uint16(v[7]) || r.Counter != int16(v[8]) || r.Direction != int16(v[9]) || r.VerticalFraction != uint16(v[10]) || r.Health != uint16(v[11]) || r.StrongHealth != (v[12] != 0) || r.PowerOrScore != uint16(v[13]) || r.WaveBonusToken != uint16(v[14]) || r.EmitterClock != uint16(v[15]) || w.random.A != uint32(v[18]) || w.random.B != uint32(v[19]) {
			t.Fatalf("second constructor %v: actor=%+v residue=%+v random=%+v", v, actor, r, w.random)
		}
		if v[0] >= 2 && actor.Sprite != bank.Atlas.SourceSpriteNames[int(v[16])] {
			t.Fatalf("second constructor art %v: %s", v, actor.Sprite)
		}
		// Generic end-of-phase bookkeeping must preserve the same constructor
		// values, including the fractions unused by whole-pixel callbacks.
		w.storeActorResidue(actor)
		if actor.Binding.Residue != r {
			t.Fatal("specialized constructor state was overwritten by generic storage")
		}
		cases++
	})
	if cases != 14 {
		t.Fatalf("second constructor coverage:%d", cases)
	}
}

func TestSecondSpecializedResidueStoresLiveCounters(t *testing.T) {
	w := testWorld(t)
	for _, actor := range []*WorldActor{
		{Active: true, ActorList: "scenery", fixedHatch: &FixedHatchState{X: 100, WorldY: 2200, Phase: 8}, part: &visualassets.ActorPart{ResourceTag: 240}},
		{Active: true, ActorList: "scenery", fixedPod: &FixedPodState{X: 110, WorldY: 2201, Phase: 7, Repeats: 0}, part: &visualassets.ActorPart{ResourceTag: 248}},
		{Active: true, ActorList: "moving", Health: 1, Score: 20, hatchCreature: &HatchCreatureState{X: 50, Y: 60, Timer: 37, Direction: 5}, part: &visualassets.ActorPart{ResourceTag: 264}},
		{Active: true, ActorList: "moving", Health: 2, Score: 200, podCreature: &PodCreatureState{X: 51, Y: 61, Phase: 1}, part: &visualassets.ActorPart{ResourceTag: 228}},
	} {
		if err := w.bindWorldActor(actor); err != nil {
			t.Fatal(err)
		}
		actor.Binding.Residue = secondResidueFixture()
		w.storeActorResidue(actor)
		r := actor.Binding.Residue
		if r.X == 0 || r.Y == 0 || r.Counter == 71 || r.XFraction != 0x1234 || r.YFraction != 0x5678 || r.EmitterClock != 0xa55a {
			t.Fatalf("specialized live state lost named fields:%+v", r)
		}
	}
}

func TestWaveAnimationModesStoreSourcePathAndFireState(t *testing.T) {
	w := testWorld(t)
	for _, mode := range []string{"path", "path-heading-frames", "path-entry-edge-frames", "follow-leader"} {
		actor := &WorldActor{Active: true, part: &visualassets.ActorPart{ResourceTag: 208, MotionMode: mode}, motion: PathMotionState{X: 0x12345678, Y: 0x56781234, AngleFixed: 0x00301234, Remaining: -8, Budget: 5}, fire: EnemyFireState{Accumulator: 27, Rate: 8}}
		if err := w.bindWorldActor(actor); err != nil {
			t.Fatal(err)
		}
		w.storeActorResidue(actor)
		r := actor.Binding.Residue
		if r.XFraction != 0x5678 || r.YFraction != 0x1234 || r.Counter != -8 || r.MotionBudget != 5 || r.Direction != 0x1234 || r.HorizontalDriftRemainder != 0x30 || r.FireAccumulator() != 27 || r.FireRate() != 8 {
			t.Fatalf("wave %s did not retain source path/fire state:%+v", mode, r)
		}
	}
}
