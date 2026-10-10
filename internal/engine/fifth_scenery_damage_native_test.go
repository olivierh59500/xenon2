package engine

import "testing"

func newFifthSceneryDamageReferenceWorld(t *testing.T, data LevelData, record, part, profile, amount, frame int) (*World, int, int) {
	t.Helper()
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	first := w.Pool.FreeFirst()
	for slot := first; slot != NoActorSlot; slot = w.Pool.Slot(slot).freeNext {
		w.Pool.Slot(slot).Residue = waveConstructorResidueFixture(slot)
	}
	clear(w.Level.Terrain.Map)
	w.WaveBonuses.BeginPass(0)
	r := data.Encounters.Fixed[record]
	w.ScrollY, w.MaximumScrollY, w.ScrollDelta = r.Y-108, 4607, 0
	for _, kind := range w.Level.FixedTiles.Kinds {
		if kind.Kind == r.EnemyKind {
			for _, change := range kind.InitialChanges {
				position := ((r.Y-8)/16)*w.Level.Terrain.Columns + (r.X-8)/16 + change.ColumnOffset
				w.setSecondMapCell(position%w.Level.Terrain.Columns, position/w.Level.Terrain.Columns, change.Before)
			}
		}
	}
	if !w.spawnFifthTile(r) {
		t.Fatal("actual record did not select a fifth-stage scenery constructor")
	}
	target := first + part
	actor := w.poolActors[target]
	if actor == nil || actor.fifthTile == nil {
		t.Fatal("actual scenery part was not allocated")
	}
	for index := 0; profile != 0 && w.Pool.FreeFirst() != NoActorSlot; index++ {
		if profile == 1 {
			w.spawnEnemyShot(80+index%20*8, 48+index%12*8, EnemyShot{Direction: uint8(index & 7), Speed: 4 + index%5})
		} else if _, err := w.reserveWorldActor(200, ActorPoolMoving, true); err != nil {
			t.Fatal(err)
		}
	}
	w.Score, w.SoundRequests = 0, [4]string{}
	if frame != 0 {
		w.advanceFifthTile(actor)
		w.finishActorUpdate(actor)
	}
	w.damageActor(actor, uint16(amount))
	if w.poolError != nil {
		t.Fatal(w.poolError)
	}
	return w, first, target
}

func TestEveryFifthSceneryPartDamageAndRestorationNativeTraceOptional(t *testing.T) {
	data, _, _ := nativeWaveReferenceData(t)
	codes := nativeRuntimeTileCodes(t, data)
	// The original follows the hit post's next/previous pointers after an
	// explosion can move it to a different list. In that case it marks a list
	// header dead. Preserve the intended barrier cohort instead; its normal
	// source callback supplies the independent expected retirement tags.
	baselineTags := make(map[[4]int][3]int64)
	nativeCombatRows(t, "fifth-scenery-damage.csv", func(v []int64) {
		if len(v) != 66 {
			t.Fatalf("fifth scenery trace has %d columns, want 66", len(v))
		}
		if v[2] == 0 && int(v[5]-v[6]) < 3 {
			key := [4]int{int(v[1]), int(v[61]), int(v[3]), int(v[58])}
			tags := baselineTags[key]
			tags[int(v[5]-v[6])] = v[11]
			baselineTags[key] = tags
		}
	})
	var w *World
	previous := [5]int{-1, -1, -1, -1, -1}
	seen := make(map[[5]int]bool)
	nextSlot, cases, rows, corruptedHeaders, correctedRows := ActorPoolCapacity, 0, 0, 0, 0
	var families [4]int
	nativeCombatRows(t, "fifth-scenery-damage.csv", func(v []int64) {
		if len(v) != 66 {
			t.Fatalf("fifth scenery trace has %d columns, want 66", len(v))
		}
		key := [5]int{int(v[1]), int(v[61]), int(v[2]), int(v[3]), int(v[58])}
		if key != previous {
			if nextSlot != ActorPoolCapacity || seen[key] || v[0] != 5 || key[0] < 0 || key[0] >= len(data[4].Encounters.Fixed) || key[2] < 0 || key[2] > 2 || key[3] != 1 && key[3] != 127 || key[4] < 0 || key[4] > 1 {
				t.Fatalf("invalid or incomplete fifth scenery case %v after %v", key, previous)
			}
			r := data[4].Encounters.Fixed[key[0]]
			family := -1
			for index, kind := range []int{1, 3, 4, 9} {
				if r.EnemyKind == kind {
					family = index
				}
			}
			if family < 0 || key[1] < 0 || key[1] >= 3 || r.EnemyKind != 1 && key[1] != 0 {
				t.Fatalf("unexpected original scenery family or part in %v", key)
			}
			seen[key] = true
			corrupt := r.EnemyKind == 1 && key[1] != 1 && key[2] == 2 && key[3] == 127
			wantHeaders := [2]int64{}
			if corrupt {
				if key[1] == 0 {
					wantHeaders[1] = 4
				} else {
					wantHeaders[0] = 4
				}
				corruptedHeaders++
			}
			if [2]int64{v[64], v[65]} != wantHeaders {
				t.Fatalf("unclassified original list-header state in %v: %v", key, v[64:66])
			}
			var first, target int
			w, first, target = newFifthSceneryDamageReferenceWorld(t, data[4], key[0], key[1], key[2], key[3], key[4])
			if first != int(v[6]) || target != int(v[4]) || w.Pool.FreeFirst() != int(v[7]) || w.ScrollY != int(v[60]) {
				t.Fatalf("fifth scenery case %v target/free/scroll differs: target %d free %d scroll %d original %v", key, target, w.Pool.FreeFirst(), w.ScrollY, v)
			}
			if gotMap := terrainDamageMapHash(t, w, codes[4]); gotMap != uint32(v[59]) || w.random != (RandomState{A: uint32(v[34]), B: uint32(v[35])}) || w.Score != int(v[36]) || w.SoundRequests[1] != nativeRewardSound(v[37]) || w.SoundRequests[2] != nativeRewardSound(v[38]) || w.fifthDestroyedTurrets[0] != (v[62] != 0) || w.fifthDestroyedTurrets[1] != (v[63] != 0) {
				t.Fatalf("fifth scenery case %v map %08x RNG %+v score %d sound %v persistent %v, original map %08x states %v", key, gotMap, w.random, w.Score, w.SoundRequests, w.fifthDestroyedTurrets, uint32(v[59]), v[34:39])
			}
			if w.WaveBonuses.NextID != uint16(v[39]) || w.WaveBonuses.NormalID != uint16(v[40]) || w.WaveBonuses.HeavyID != uint16(v[41]) {
				t.Fatalf("fifth scenery case %v changed original reward cache", key)
			}
			for index, bucket := range w.WaveBonuses.Entries {
				if bucket != (WaveBonusEntry{ID: uint16(v[42+index*2]), Remaining: uint16(v[43+index*2])}) {
					t.Fatalf("fifth scenery case %v changed reward bucket %d", key, index)
				}
			}
			previous, nextSlot = key, first
			cases++
			families[family]++
		}
		if int(v[5]) != nextSlot {
			t.Fatalf("fifth scenery case %v skipped slot %d for %d", key, nextSlot, v[5])
		}
		nextSlot++
		if key[1] == 0 && key[2] == 2 && key[3] == 127 && data[4].Encounters.Fixed[key[0]].EnemyKind == 1 && v[5] > v[6] && v[5] < v[6]+3 {
			baseline := [4]int{key[0], key[1], key[3], key[4]}
			tags, ok := baselineTags[baseline]
			member := w.poolActors[int(v[5])]
			if !ok || tags[int(v[5]-v[6])] != 4 || member == nil || v[11] != int64(member.part.ResourceTag) {
				t.Fatalf("missing independent intact-cohort restoration for %v", key)
			}
			v[11] = tags[int(v[5]-v[6])]
			correctedRows++
		}
		compareDamageNativePoolRow(t, key, w, v)
		rows++
	})
	if cases != 384 || rows != 59136 || nextSlot != ActorPoolCapacity || families != [4]int{180, 120, 60, 24} || corruptedHeaders != 20 || correctedRows != 20 {
		t.Fatalf("incomplete fifth scenery damage comparison: cases %d rows %d families %v corrupt headers %d corrected rows %d", cases, rows, families, corruptedHeaders, correctedRows)
	}
	t.Logf("Compared %d fifth scenery damage cases and %d physical-slot states; %d source header-corruption cases retain a valid barrier cohort, changing %d intended retirement tags", cases, rows, corruptedHeaders, correctedRows)
}
