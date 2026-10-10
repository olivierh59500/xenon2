package engine

import "testing"

func newFifthGuardianDamageReferenceWorld(t *testing.T, data LevelData, record, part, profile, amount int) (*World, int, int) {
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
	if r.EnemyKind == 6 {
		w.ScrollY = 416
	}
	w.spawnFixed(r)
	actors := w.fifthMiddleActors[:]
	if r.EnemyKind == 6 {
		actors = w.fifthFinalActors[:]
	}
	actor := actors[part]
	if actor == nil || actor.fifthIndex == 0 {
		t.Fatal("actual fifth guardian component was not constructed")
	}
	target := actor.Binding.Slot
	for index := 0; profile != 0 && w.Pool.FreeFirst() != NoActorSlot; index++ {
		if profile == 1 {
			w.spawnEnemyShot(80+index%20*8, 48+index%12*8, EnemyShot{Direction: uint8(index & 7), Speed: 4 + index%5})
		} else if _, err := w.reserveWorldActor(200, ActorPoolMoving, true); err != nil {
			t.Fatal(err)
		}
	}
	w.Score, w.SoundRequests, w.ImmediateSoundRequests = 0, [4]string{}, [4]string{}
	w.damageActor(actor, uint16(amount))
	if w.poolError != nil {
		t.Fatal(w.poolError)
	}
	return w, first, target
}

func TestEveryFifthGuardianDamageCallbackUnderPressureNativeTraceOptional(t *testing.T) {
	data, _, _ := nativeWaveReferenceData(t)
	codes := nativeRuntimeTileCodes(t, data)
	var w *World
	previous := [4]int{-1, -1, -1, -1}
	seen := make(map[[4]int]bool)
	nextSlot, cases, rows := ActorPoolCapacity, 0, 0
	var families [2]int
	nativeCombatRows(t, "fifth-guardian-pool-damage.csv", func(v []int64) {
		if len(v) != 64 {
			t.Fatalf("fifth guardian damage trace has %d columns, want 64", len(v))
		}
		key := [4]int{int(v[1]), int(v[58]), int(v[2]), int(v[3])}
		if key != previous {
			if nextSlot != ActorPoolCapacity || seen[key] || v[0] != 5 || key[0] < 0 || key[0] >= len(data[4].Encounters.Fixed) || key[2] < 0 || key[2] > 2 || key[3] != 1 && key[3] != 255 {
				t.Fatalf("invalid or incomplete fifth guardian damage case %v after %v", key, previous)
			}
			kind := data[4].Encounters.Fixed[key[0]].EnemyKind
			if kind != 5 && kind != 6 || kind == 5 && (key[1] < 1 || key[1] > 5) || kind == 6 && (key[1] < 3 || key[1] > 21) {
				t.Fatalf("unexpected original guardian component in %v", key)
			}
			seen[key] = true
			var first, target int
			w, first, target = newFifthGuardianDamageReferenceWorld(t, data[4], key[0], key[1], key[2], key[3])
			if first != int(v[6]) || target != int(v[4]) || w.Pool.FreeFirst() != int(v[7]) || w.ScrollY != int(v[60]) {
				t.Fatalf("fifth guardian damage %v target/free/scroll differs: target %d free %d scroll %d original %v", key, target, w.Pool.FreeFirst(), w.ScrollY, v)
			}
			outer, victory := 0, w.LevelFinished
			if kind == 6 {
				outer = w.FifthFinal.OuterRemaining
			}
			immediate := ""
			if v[63] >= 0 {
				immediate = nativeRewardSound(v[63])
			}
			if gotMap := terrainDamageMapHash(t, w, codes[4]); gotMap != uint32(v[59]) || w.random != (RandomState{A: uint32(v[34]), B: uint32(v[35])}) || w.Score != int(v[36]) || w.SoundRequests[1] != nativeRewardSound(v[37]) || w.SoundRequests[2] != nativeRewardSound(v[38]) || outer != int(v[61]) || victory != (v[62] != 0) || w.ImmediateSoundRequests[0] != immediate {
				t.Fatalf("fifth guardian damage %v map %08x RNG %+v score %d sound %v immediate %v outer %d victory %v, original map %08x metadata %v", key, gotMap, w.random, w.Score, w.SoundRequests, w.ImmediateSoundRequests, outer, victory, uint32(v[59]), v[34:])
			}
			if w.WaveBonuses.NextID != uint16(v[39]) || w.WaveBonuses.NormalID != uint16(v[40]) || w.WaveBonuses.HeavyID != uint16(v[41]) {
				t.Fatalf("fifth guardian damage %v changed original reward cache", key)
			}
			for index, bucket := range w.WaveBonuses.Entries {
				if bucket != (WaveBonusEntry{ID: uint16(v[42+index*2]), Remaining: uint16(v[43+index*2])}) {
					t.Fatalf("fifth guardian damage %v changed reward bucket %d", key, index)
				}
			}
			previous, nextSlot = key, first
			cases++
			families[kind-5]++
		}
		if int(v[5]) != nextSlot {
			t.Fatalf("fifth guardian damage %v skipped slot %d for %d", key, nextSlot, v[5])
		}
		nextSlot++
		compareDamageNativePoolRow(t, key, w, v)
		rows++
	})
	if cases != 144 || rows != 22176 || nextSlot != ActorPoolCapacity || families != [2]int{30, 114} {
		t.Fatalf("incomplete fifth guardian damage coverage: cases %d rows %d families %v", cases, rows, families)
	}
	t.Logf("Compared %d complete fifth guardian damage callbacks and %d physical-slot states", cases, rows)
}
