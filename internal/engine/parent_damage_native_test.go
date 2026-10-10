package engine

import "testing"

// Tail entries are explicit inert capacity holders. The hit target always comes
// from the real wave constructor and retains its production damage callback.
func newWaveDamagePartReferenceWorld(t *testing.T, data LevelData, record, profile, amount, bodyOrder int) (*World, int, int) {
	t.Helper()
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	first := w.Pool.FreeFirst()
	for slot := first; slot != NoActorSlot; slot = w.Pool.Slot(slot).freeNext {
		w.Pool.Slot(slot).Residue = waveConstructorResidueFixture(slot)
	}
	w.WaveBonuses.BeginPass(0)
	if err := w.spawnWave(data.Encounters.Moving[record]); err != nil {
		t.Fatal(err)
	}
	var target *WorldActor
	for slot := w.Pool.First(ActorPoolMoving); slot != NoActorSlot; slot = w.Pool.Next(slot) {
		if slot >= first && (bodyOrder < 0 || w.poolActors[slot].part.DamageMode == "group") {
			if bodyOrder <= 0 {
				target = w.poolActors[slot]
				break
			}
			bodyOrder--
		}
	}
	if target == nil {
		t.Fatal("nonempty original wave did not provide a target")
	}
	targetSlot := target.Binding.Slot
	for index := 0; profile != 0 && w.Pool.FreeFirst() != NoActorSlot; index++ {
		if profile == 1 {
			w.spawnEnemyShot(80+index%20*8, 48+index%12*8, EnemyShot{Direction: uint8(index & 7), Speed: 4 + index%5})
		} else if _, err := w.reserveWorldActor(200, ActorPoolMoving, true); err != nil {
			t.Fatal(err)
		}
		if w.poolError != nil {
			t.Fatal(w.poolError)
		}
	}
	w.Score, w.SoundRequests = 0, [4]string{}
	w.damageActor(target, uint16(amount))
	if w.poolError != nil {
		t.Fatal(w.poolError)
	}
	return w, first, targetSlot
}

func TestActualWaveParentDamageUnderPressureNativeTraceOptional(t *testing.T) {
	compareWaveDamageNativeTrace(t, "parent-death.csv", false, 3600, 550746, [5]int{672, 522, 894, 618, 894})
}

func TestEveryCompoundWavePartDamageUnderPressureNativeTraceOptional(t *testing.T) {
	compareWaveDamageNativeTrace(t, "compound-part-damage.csv", true, 2502, 383682, [5]int{240, 0, 1386, 876, 0})
}

func compareWaveDamageNativeTrace(t *testing.T, name string, compound bool, wantCases, wantRows int, wantLevels [5]int) {
	t.Helper()
	data, _, _ := nativeWaveReferenceData(t)
	var w *World
	first, target := 0, 0
	previous := [5]int{-1, -1, -1, -1, -1}
	nextSlot := ActorPoolCapacity
	seen := make(map[[5]int]bool)
	var levels [5]int
	cases, rows := 0, 0
	nativeCombatRows(t, name, func(v []int64) {
		order := -1
		if compound {
			order = int(v[58])
		}
		key := [5]int{int(v[0]), int(v[1]), int(v[2]), int(v[3]), order}
		if key != previous {
			if nextSlot != ActorPoolCapacity || key[0] < 1 || key[0] > 5 || key[1] < 0 || key[1] >= len(data[key[0]-1].Encounters.Moving) || key[2] < 0 || key[2] > 2 || key[3] != 1 && key[3] != 127 || compound && order < 0 || seen[key] {
				t.Fatalf("invalid, duplicated or incomplete original damage case %v after %v", key, previous)
			}
			seen[key] = true
			w, first, target = newWaveDamagePartReferenceWorld(t, data[key[0]-1], key[1], key[2], key[3], key[4])
			if target != int(v[4]) || first != int(v[6]) || w.Pool.FreeFirst() != int(v[7]) {
				t.Fatalf("damage %v: target/first/free %d/%d/%d, original %d/%d/%d", key, target, first, w.Pool.FreeFirst(), v[4], v[6], v[7])
			}
			if w.random != (RandomState{A: uint32(v[34]), B: uint32(v[35])}) || w.Score != int(v[36]) || w.SoundRequests[1] != nativeRewardSound(v[37]) || w.SoundRequests[2] != nativeRewardSound(v[38]) || w.WaveBonuses.NextID != uint16(v[39]) || w.WaveBonuses.NormalID != uint16(v[40]) || w.WaveBonuses.HeavyID != uint16(v[41]) {
				t.Fatalf("damage %v: RNG%+v score%d sound%v cache%+v, original%v", key, w.random, w.Score, w.SoundRequests, w.WaveBonuses, v[34:])
			}
			for index, bucket := range w.WaveBonuses.Entries {
				if bucket != (WaveBonusEntry{ID: uint16(v[42+index*2]), Remaining: uint16(v[43+index*2])}) {
					t.Fatalf("damage %v: original bucket%d differs: %+v vs %v", key, index, bucket, v[42+index*2:44+index*2])
				}
			}
			previous, nextSlot = key, first
			cases++
			levels[key[0]-1]++
		}
		index := int(v[5])
		if index != nextSlot {
			t.Fatalf("damage %v skipped original physical slot%d for%d", key, nextSlot, index)
		}
		nextSlot++
		compareDamageNativePoolRow(t, key, w, v)
		rows++
	})
	if cases != wantCases || rows != wantRows || nextSlot != ActorPoolCapacity || levels != wantLevels {
		t.Fatalf("incomplete actual-wave damage coverage: cases%d rows%d levels%v", cases, rows, levels)
	}
	t.Logf("Compared %d actual-wave damage/capacity cases and %d physical-slot states", cases, rows)
}

func compareDamageNativePoolRow(t *testing.T, key any, w *World, v []int64) {
	t.Helper()
	index := int(v[5])
	slot := w.Pool.Slot(index)
	if slot.ResourceTag != int16(v[11]) || int(slot.list) != int(v[8]) || slot.Linked != (v[12] != 0) || slot.SkipDeathEffect != (v[12]&128 != 0) || slot.AuxiliaryFlags != ([2]bool{v[13] != 0, v[14] != 0}) || slot.list != ActorPoolNone && (slot.previous != int(v[9]) || slot.next != int(v[10])) || slot.list == ActorPoolNone && slot.freeNext != int(v[9]) {
		t.Fatalf("damage %v slot%d: physical type/list/flags %+v, original%v", key, index, slot, v[8:15])
	}
	r := slot.Residue
	got := [19]int64{int64(r.X), int64(r.Y), int64(r.XFraction), int64(r.YFraction), int64(r.Counter), int64(r.Direction), int64(r.HorizontalDriftRemainder), int64(r.VerticalVelocity), int64(r.VerticalFraction), int64(r.Health), int64(boolCount(r.StrongHealth)), int64(r.PowerOrScore), int64(r.WaveBonusToken), int64(r.MotionBudget), int64(r.EmitterClock), int64(r.MountOffsetX), int64(r.MountOffsetY), int64(r.OwnerSlot), int64(r.FollowingSlot)}
	var want [19]int64
	copy(want[:], v[15:34])
	want[10] = int64(boolCount(v[25] != 0))
	if got != want {
		t.Fatalf("damage %v slot%d tag%d: fields%v; original%v", key, index, slot.ResourceTag, got, want)
	}
}
