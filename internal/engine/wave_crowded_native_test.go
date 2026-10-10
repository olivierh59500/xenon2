package engine

import "testing"

// The source fixture uses actual projectile factories to fill the shared pool.
// Only the two explicitly dead projectile tags are artificial capacity cases.
func newCrowdedWaveReferenceWorld(t *testing.T, data LevelData, profile int) (*World, int) {
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
	count := ActorPoolCapacity - first
	if profile == 0 {
		count -= 2
	}
	for index := range count {
		w.spawnEnemyShot(80+index%20*8, 48+index%12*8, EnemyShot{Direction: uint8(index & 7), Speed: 4 + index%5})
		if w.poolError != nil {
			t.Fatal(w.poolError)
		}
	}
	if profile == 2 {
		for _, shot := range w.Projectiles {
			if shot.Binding.Slot == first+2 || shot.Binding.Slot == first+10 {
				shot.Active = false
				w.retireWorldActor(shot.Binding)
			}
		}
	}
	return w, first
}

func TestAllMovingWaveCrowdedPoolNativeTraceOptional(t *testing.T) {
	data, _, _ := nativeWaveReferenceData(t)
	var w *World
	first := 0
	previous := [4]int{-1, -1, -1, -1}
	nextSlot := ActorPoolCapacity
	seen := make(map[[3]int]bool)
	var levels [5]int
	cases, boundaries, slots := 0, 0, 0
	boolean := func(value bool) int64 {
		if value {
			return 1
		}
		return 0
	}
	nativeCombatRows(t, "crowded-wave-pool.csv", func(v []int64) {
		key := [4]int{int(v[0]), int(v[1]), int(v[2]), int(v[3])}
		if key != previous {
			if nextSlot != ActorPoolCapacity {
				t.Fatalf("original pressure boundary %v ended before slot %d", previous, nextSlot)
			}
			if key[3] == 0 {
				identity := [3]int{key[0], key[1], key[2]}
				if key[0] < 1 || key[0] > 5 || key[1] < 0 || key[1] >= len(data[key[0]-1].Encounters.Moving) || key[2] < 0 || key[2] > 2 || seen[identity] || previous[3] == 0 {
					t.Fatalf("invalid, duplicated or unfinished original capacity case %v", key)
				}
				seen[identity] = true
				w, first = newCrowdedWaveReferenceWorld(t, data[key[0]-1], key[2])
				cases++
				levels[key[0]-1]++
			} else if previous != ([4]int{key[0], key[1], key[2], 0}) {
				t.Fatalf("original pressure boundary is not consecutive: %v after %v", key, previous)
			} else if err := w.spawnWave(data[key[0]-1].Encounters.Moving[key[1]]); err != nil {
				t.Fatal(err)
			}
			if first != int(v[5]) || w.Pool.FreeFirst() != int(v[6]) {
				t.Fatalf("pressure %v: original first/free %d/%d, Go %d/%d", key, v[5], v[6], first, w.Pool.FreeFirst())
			}
			cache := w.WaveBonuses
			if w.random.A != uint32(v[33]) || w.random.B != uint32(v[34]) || cache.Entries[0] != (WaveBonusEntry{ID: uint16(v[35]), Remaining: uint16(v[36])}) || cache.Entries[1] != (WaveBonusEntry{ID: uint16(v[37]), Remaining: uint16(v[38])}) {
				t.Fatalf("pressure %v: RNG %+v/cache %+v differs from original %v", key, w.random, cache, v[33:])
			}
			previous = key
			nextSlot = first
			boundaries++
		}
		index := int(v[4])
		if index != nextSlot {
			t.Fatalf("original pressure %v skipped slot %d for %d", key, nextSlot, index)
		}
		nextSlot++
		slot := w.Pool.Slot(index)
		if int(slot.list) != int(v[7]) || slot.ResourceTag != int16(v[10]) || slot.Linked != (v[11] != 0) || slot.AuxiliaryFlags != ([2]bool{v[12] != 0, v[13] != 0}) {
			t.Fatalf("pressure %v slot%d: list%d tag%d linked%v flags%v, original %v", key, index, slot.list, slot.ResourceTag, slot.Linked, slot.AuxiliaryFlags, v[7:14])
		}
		if slot.list != ActorPoolNone && (slot.previous != int(v[8]) || slot.next != int(v[9])) {
			t.Fatalf("pressure %v slot%d: physical links %d/%d, original %d/%d", key, index, slot.previous, slot.next, v[8], v[9])
		}
		if slot.list == ActorPoolNone && slot.freeNext != int(v[8]) {
			t.Fatalf("pressure %v slot%d: free successor %d, original %d", key, index, slot.freeNext, v[8])
		}
		r := slot.Residue
		got := [19]int64{int64(r.X), int64(r.Y), int64(r.XFraction), int64(r.YFraction), int64(r.Counter), int64(r.Direction), int64(r.HorizontalDriftRemainder), int64(r.VerticalVelocity), int64(r.VerticalFraction), int64(r.Health), boolean(r.StrongHealth), int64(r.PowerOrScore), int64(r.WaveBonusToken), int64(r.MotionBudget), int64(r.EmitterClock), int64(r.MountOffsetX), int64(r.MountOffsetY), int64(r.OwnerSlot), int64(r.FollowingSlot)}
		var want [19]int64
		copy(want[:], v[14:33])
		want[10] = boolean(v[24] != 0)
		if got != want {
			t.Fatalf("pressure %v slot%d tag%d: immediate fields %v; original %v", key, index, slot.ResourceTag, got, want)
		}
		slots++
	})
	if cases != 1803 || boundaries != 3606 || slots != 551664 || nextSlot != ActorPoolCapacity || previous[3] != 1 || levels != [5]int{339, 261, 447, 309, 447} {
		t.Fatalf("incomplete crowded wave coverage: cases%d boundaries%d levels%v slots%d", cases, boundaries, levels, slots)
	}
	t.Logf("Compared %d actual wave/capacity cases, %d before/after boundaries and %d physical slots", cases, boundaries, slots)
}
