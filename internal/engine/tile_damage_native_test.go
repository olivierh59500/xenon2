package engine

import (
	"os"
	"path/filepath"
	"testing"

	"xenon2/internal/visualassets"
)

func nativeRuntimeTileCodes(t *testing.T, data [5]LevelData) [5]map[uint16]uint16 {
	t.Helper()
	var result [5]map[uint16]uint16
	for index, name := range []string{"000B00E5", "00FA00FE", "02020113", "031F0159", "04820138"} {
		raw, err := os.ReadFile(filepath.Join(filepath.Dir(os.Getenv("XENON2_NATIVE_TRACE_DIR")), "imported", name+".decoded"))
		if err != nil {
			t.Fatal(err)
		}
		_, extra, err := visualassets.DecodeFixedTiles(index+1, raw)
		if err != nil {
			t.Fatal(err)
		}
		base, err := visualassets.DecodeTerrain(raw)
		if err != nil {
			t.Fatal(err)
		}
		_, guardianTiles, err := visualassets.DecodeGuardianArt(index+1, raw, base.Palette)
		if err != nil {
			t.Fatal(err)
		}
		extra = append(extra, guardianTiles...)
		groups, _, err := visualassets.DecodeCompoundGuardianArt(index+1, raw, base.Palette)
		if err != nil {
			t.Fatal(err)
		}
		extra = append(extra, visualassets.GuardianGroupTileCodes(groups)...)
		sprites, err := visualassets.DecodeFixedSprites(index+1, raw, base.Palette)
		if err != nil {
			t.Fatal(err)
		}
		extra = append(extra, visualassets.ThirdFixedTileCodes(sprites.Third)...)
		if index == 3 {
			codes, err := visualassets.FourthCrawlerTileCodes(raw)
			if err != nil {
				t.Fatal(err)
			}
			extra = append(extra, codes...)
		}
		terrain, err := visualassets.DecodeTerrainWithTiles(raw, extra)
		if err != nil {
			t.Fatal(err)
		}
		if len(terrain.Tiles) != len(data[index].Terrain.Tiles) {
			t.Fatal("original tile-code recovery differs from the production atlas")
		}
		result[index] = map[uint16]uint16{0: 0}
		for code, id := range terrain.SourceTileIDs {
			result[index][id] = code
		}
	}
	return result
}

func terrainDamageMapHash(t *testing.T, w *World, codes map[uint16]uint16) uint32 {
	t.Helper()
	hash := uint32(2166136261)
	for _, tile := range w.Level.Terrain.Map {
		code, ok := codes[tile]
		if !ok {
			t.Fatalf("terrain restoration requested unknown runtime tile %d", tile)
		}
		tile = code
		hash = (hash ^ uint32(tile>>8)) * 16777619
		hash = (hash ^ uint32(tile&255)) * 16777619
	}
	return hash
}

func newTerrainDamageReferenceWorld(t *testing.T, data LevelData, record, profile, amount, frame int) (*World, int, int) {
	t.Helper()
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	first := w.Pool.FreeFirst()
	for slot := first; slot != NoActorSlot; slot = w.Pool.Slot(slot).freeNext {
		w.Pool.Slot(slot).Residue = waveConstructorResidueFixture(slot)
	}
	clear(w.Level.Terrain.Map) // Isolate constructor/restoration writes in both fixtures.
	w.WaveBonuses.BeginPass(0)
	r := data.Encounters.Fixed[record]
	w.ScrollY, w.MaximumScrollY, w.ScrollDelta = r.Y-108, 4607, 0
	if !w.spawnFixedTile(r) {
		t.Fatal("original cannon record did not select a terrain constructor")
	}
	actor := w.Actors[0]
	target := actor.Binding.Slot
	for index := 0; profile != 0 && w.Pool.FreeFirst() != NoActorSlot; index++ {
		if profile == 1 {
			w.spawnEnemyShot(80+index%20*8, 48+index%12*8, EnemyShot{Direction: uint8(index & 7), Speed: 4 + index%5})
		} else if _, err := w.reserveWorldActor(200, ActorPoolMoving, true); err != nil {
			t.Fatal(err)
		}
	}
	w.Score, w.SoundRequests = 0, [4]string{}
	if frame != 0 {
		w.advanceFixedTile(actor)
		w.finishActorUpdate(actor)
	}
	w.damageActor(actor, uint16(amount))
	if w.poolError != nil {
		t.Fatal(w.poolError)
	}
	return w, first, target
}

func TestEveryTerrainCannonDamageAndRestorationNativeTraceOptional(t *testing.T) {
	data, _, _ := nativeWaveReferenceData(t)
	codes := nativeRuntimeTileCodes(t, data)
	// Full moving lists can reclaim the hit cannon before its restoration
	// copy. Compare that map against the same original callback with free
	// capacity, preserving its intended tiles instead of corrupted animation data.
	referenceMaps := make(map[[5]int]uint32)
	nativeCombatRows(t, "tile-damage.csv", func(v []int64) {
		if len(v) != 61 {
			t.Fatalf("terrain damage trace has %d columns, want 61", len(v))
		}
		key := [5]int{int(v[0]), int(v[1]), int(v[2]), int(v[3]), int(v[58])}
		if old, exists := referenceMaps[key]; exists && old != uint32(v[59]) {
			t.Fatalf("terrain case %v has inconsistent original map hashes", key)
		}
		referenceMaps[key] = uint32(v[59])
	})
	var w *World
	previous := [5]int{-1, -1, -1, -1, -1}
	seen := make(map[[5]int]bool)
	nextSlot, cases, rows, correctedMaps := ActorPoolCapacity, 0, 0, 0
	var levels [5]int
	nativeCombatRows(t, "tile-damage.csv", func(v []int64) {
		key := [5]int{int(v[0]), int(v[1]), int(v[2]), int(v[3]), int(v[58])}
		if key != previous {
			if nextSlot != ActorPoolCapacity || seen[key] || key[0] < 1 || key[0] > 5 || key[1] < 0 || key[1] >= len(data[key[0]-1].Encounters.Fixed) || key[2] < 0 || key[2] > 2 || key[3] != 1 && key[3] != 127 || key[4] < 0 || key[4] > 1 {
				t.Fatalf("invalid or incomplete terrain damage case %v after %v", key, previous)
			}
			seen[key] = true
			var first, target int
			w, first, target = newTerrainDamageReferenceWorld(t, data[key[0]-1], key[1], key[2], key[3], key[4])
			if target != int(v[4]) || first != int(v[6]) || w.Pool.FreeFirst() != int(v[7]) || w.ScrollY != int(v[60]) {
				t.Fatalf("terrain case%v target/free/scroll differ: target%d free%d scroll%d original%v", key, target, w.Pool.FreeFirst(), w.ScrollY, v)
			}
			wantMap := uint32(v[59])
			if key[2] == 2 && key[3] == 127 {
				baseline := key
				baseline[2] = 0
				var ok bool
				wantMap, ok = referenceMaps[baseline]
				if !ok {
					t.Fatalf("terrain case %v lacks an unsaturated original restoration", key)
				}
				if wantMap != uint32(v[59]) {
					correctedMaps++
				}
			}
			gotMap := terrainDamageMapHash(t, w, codes[key[0]-1])
			if gotMap != wantMap || w.random != (RandomState{A: uint32(v[34]), B: uint32(v[35])}) || w.Score != int(v[36]) || w.SoundRequests[1] != nativeRewardSound(v[37]) || w.SoundRequests[2] != nativeRewardSound(v[38]) || w.WaveBonuses.NextID != uint16(v[39]) || w.WaveBonuses.NormalID != uint16(v[40]) || w.WaveBonuses.HeavyID != uint16(v[41]) {
				t.Fatalf("terrain case%v map%08x RNG%+v score%d sound%v cache%+v, expected map%08x (raw original %08x) states%v", key, gotMap, w.random, w.Score, w.SoundRequests, w.WaveBonuses, wantMap, uint32(v[59]), v[34:42])
			}
			for index, bucket := range w.WaveBonuses.Entries {
				if bucket != (WaveBonusEntry{ID: uint16(v[42+index*2]), Remaining: uint16(v[43+index*2])}) {
					t.Fatalf("terrain case %v original reward bucket %d differs", key, index)
				}
			}
			previous, nextSlot = key, first
			cases++
			levels[key[0]-1]++
		}
		if int(v[5]) != nextSlot {
			t.Fatalf("terrain case%v skipped slot%d for%d", key, nextSlot, v[5])
		}
		nextSlot++
		compareDamageNativePoolRow(t, key, w, v)
		rows++
	})
	if cases != 852 || rows != 130176 || nextSlot != ActorPoolCapacity || levels != [5]int{384, 132, 120, 84, 132} || correctedMaps != 118 {
		t.Fatalf("incomplete terrain callback coverage: cases%d rows%d levels%v corrected maps%d", cases, rows, levels, correctedMaps)
	}
	t.Logf("Compared %d terrain-cannon callbacks and %d physical-slot states; %d maps use the original unsaturated restoration to avoid its reclaimed-slot corruption", cases, rows, correctedMaps)
}
