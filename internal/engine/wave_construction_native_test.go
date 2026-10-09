package engine

import (
	"os"
	"path/filepath"
	"testing"

	"xenon2/internal/visualassets"
)

func waveConstructorResidueFixture(slot int) ActorResidue {
	return ActorResidue{X: int16(11 + slot), Y: int16(23 + slot), XFraction: uint16(0x1234 + slot), YFraction: uint16(0x5678 + slot),
		Counter: int16(65 + slot), Direction: int16(81 + slot), HorizontalDriftRemainder: uint16(91 + slot),
		VerticalVelocity: int16(-19 + slot), VerticalFraction: uint16(71 + slot), Health: uint16(51 + slot), StrongHealth: true,
		PowerOrScore: uint16(101 + slot), WaveBonusToken: uint16(111 + slot), MountOffsetX: int16(-7 + slot), MountOffsetY: int16(13 + slot),
		MotionBudget: int16(9 + slot), EmitterClock: uint16(0xa55a + slot), OwnerSlot: 3, LeaderSlot: NoActorSlot, FollowingSlot: 2}
}

// Unlike the earlier representative formation fixture, this comparison runs
// every actual encounter through the production World constructor. Reused
// slots contain distinguishable values, so writes and retention are separate.
func TestAllMovingWaveConstructorsNativeTraceOptional(t *testing.T) {
	root := os.Getenv("XENON2_NATIVE_TRACE_DIR")
	if root == "" {
		t.Skip("set XENON2_NATIVE_TRACE_DIR to compare every original wave constructor")
	}
	read := func(name string) []byte {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(root), "imported", name+".decoded"))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	common, shop := read("XenonII"), read("05c400f8")
	catalogue, err := visualassets.DecodeShopCatalogue(shop)
	if err != nil {
		t.Fatal(err)
	}
	firstTerrain, err := visualassets.DecodeTerrain(read("000B00E5"))
	if err != nil {
		t.Fatal(err)
	}
	commonArt, err := visualassets.DecodeCommonActorArtWithEquipment(common, shop, catalogue, firstTerrain.Palette)
	if err != nil {
		t.Fatal(err)
	}
	var data [5]LevelData
	var sprites [5]map[int]string
	for level, name := range []string{"000B00E5", "00FA00FE", "02020113", "031F0159", "04820138"} {
		data[level] = originalWorldData(t, level+1)
		bank, err := visualassets.DecodeWaveActors(read(name), data[level].Terrain.Palette, data[level].Encounters, &commonArt)
		if err != nil {
			t.Fatal(err)
		}
		sprites[level] = bank.Atlas.SourceSpriteNames
	}
	var world *World
	var ordered [ActorPoolCapacity]*WorldActor
	var actors []*WorldActor
	cases, births := 0, 0
	nativeCombatRows(t, "all-wave-construction-trace.csv", func(v []int64) {
		level, record, reused, order := int(v[0]), int(v[1]), v[2] != 0, int(v[3])
		if order == 0 {
			world, err = NewWorld(data[level-1])
			if err != nil {
				t.Fatal(err)
			}
			if reused {
				for slot := world.Pool.FreeFirst(); slot != NoActorSlot; slot = world.Pool.Slot(slot).freeNext {
					world.Pool.Slot(slot).Residue = waveConstructorResidueFixture(slot)
				}
			}
			world.WaveBonuses.BeginPass(0)
			if err := world.spawnWave(data[level-1].Encounters.Moving[record]); err != nil {
				t.Fatal(err)
			}
			actors = world.orderedMovingActors(&ordered)
			cases++
		}
		if order >= len(actors) {
			t.Fatalf("level %d wave %d reused %v: missing original member %d", level, record, reused, order)
		}
		actor := actors[order]
		name := sprites[level-1][int(v[25])]
		if actor.Atlas == "common" {
			name = commonArt.SourceSpriteNames[int(v[25])]
		}
		collision := CollisionRect{Left: int(v[27]), Top: int(v[28]), Right: int(v[29]), Bottom: int(v[30])}
		if actor.Binding.Slot != int(v[4]) || actor.part.ResourceTag != int(v[5]) || int(actor.X) != int(v[6]) || int(actor.Y) != int(v[7]) || actor.motion.Remaining != int(v[10]) || actor.Health != int(v[15]) || actor.part.StrongHealth != (v[16] != 0) || actor.Score != int(v[17]) || actor.WaveToken != uint16(v[18]) || actor.fire.Accumulator != uint8(v[19]>>8) || actor.fire.Rate != uint8(v[19]) || actor.motion.Budget != int(v[20]) || actor.Sprite != name || actor.animationState.Remaining != int(v[26]) || actor.Collision != collision || world.Pool.Slot(actor.Binding.Slot).Linked != (v[31] != 0) {
			t.Fatalf("level %d wave %d reused %v member %d: initial semantic state differs from %v; actor %+v", level, record, reused, order, v, actor)
		}
		r := actor.Binding.Residue
		if r.X != int16(v[6]) || r.Y != int16(v[7]) || r.XFraction != uint16(v[8]) || r.YFraction != uint16(v[9]) || r.Counter != int16(v[10]) || r.Direction != int16(v[11]) || r.HorizontalDriftRemainder != uint16(v[12]) || r.Health != uint16(v[15]) || r.StrongHealth != (v[16] != 0) || r.PowerOrScore != uint16(v[17]) || r.WaveBonusToken != uint16(v[18]) || r.EmitterClock != uint16(v[19]) || r.MotionBudget != int16(v[20]) || r.OwnerSlot != int(v[23]) || r.FollowingSlot != int(v[24]) {
			t.Fatalf("level %d wave %d reused %v member %d: constructor physical state %+v differs from %v", level, record, reused, order, r, v)
		}
		// Unwritten cold-memory values are not a startup reference. Deliberately
		// seeded reused slots establish the retention of those spare words.
		if reused && (r.VerticalVelocity != int16(v[13]) || r.VerticalFraction != uint16(v[14]) || r.MountOffsetX != int16(v[21]) || r.MountOffsetY != int16(v[22])) {
			t.Fatalf("level %d wave %d member %d: retained constructor words %+v differ from %v", level, record, order, r, v)
		}
		births++
	})
	// One original record has count zero, so only its two state rows exist.
	if cases != 1200 || births != 3048 {
		t.Fatalf("incomplete real-wave coverage: %d constructor cases, %d births", cases, births)
	}
	t.Logf("Compared %d actual wave constructor cases and %d initial actor states", cases, births)
	var levels [5]int
	empty := 0
	nativeCombatRows(t, "all-wave-construction-state.csv", func(v []int64) {
		level, record, reused := int(v[0]), int(v[1]), v[2] != 0
		w, err := NewWorld(data[level-1])
		if err != nil {
			t.Fatal(err)
		}
		if reused {
			for slot := w.Pool.FreeFirst(); slot != NoActorSlot; slot = w.Pool.Slot(slot).freeNext {
				w.Pool.Slot(slot).Residue = waveConstructorResidueFixture(slot)
			}
		}
		w.WaveBonuses.BeginPass(0)
		before, firstSlot, firstID := len(w.Actors), w.Pool.FreeFirst(), w.nextActorID
		if err := w.spawnWave(data[level-1].Encounters.Moving[record]); err != nil {
			t.Fatal(err)
		}
		cache := w.WaveBonuses
		if len(w.Actors)-before != int(v[3]) || w.random.A != uint32(v[4]) || w.random.B != uint32(v[5]) || cache.NextID != uint16(v[6]) || cache.NormalID != uint16(v[7]) || cache.HeavyID != uint16(v[8]) || cache.Entries[0] != (WaveBonusEntry{ID: uint16(v[9]), Remaining: uint16(v[10])}) || cache.Entries[1] != (WaveBonusEntry{ID: uint16(v[11]), Remaining: uint16(v[12])}) {
			t.Fatalf("original wave state %v: actor count %d RNG %+v cache %+v", v, len(w.Actors)-before, w.random, cache)
		}
		if v[3] == 0 {
			empty++
			if w.Pool.FreeFirst() != firstSlot || w.nextActorID != firstID {
				t.Fatal("empty original wave allocated an actor")
			}
		}
		levels[level-1]++
	})
	if levels != [5]int{226, 174, 298, 206, 298} || empty != 2 {
		t.Fatalf("incomplete actual wave/RNG/cache coverage: levels %v empty %d", levels, empty)
	}
	t.Logf("Compared all 601 real waves with fresh/reused slots, including the empty record, RNG and shared bonus cache")
}
