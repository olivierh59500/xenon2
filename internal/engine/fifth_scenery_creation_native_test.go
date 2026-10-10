package engine

import "testing"

func newCrowdedFifthSceneryWorld(t *testing.T, data LevelData, profile int) (*World, int) {
	t.Helper()
	if profile < 3 {
		return newCrowdedWaveReferenceWorld(t, data, profile)
	}
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	first := w.Pool.FreeFirst()
	for slot := first; slot != NoActorSlot; slot = w.Pool.Slot(slot).freeNext {
		w.Pool.Slot(slot).Residue = waveConstructorResidueFixture(slot)
	}
	w.WaveBonuses.BeginPass(0)
	fillWaveDamageMovingTail(t, w)
	return w, first
}

func TestEveryFifthSceneryConstructorUnderPressureNativeTraceOptional(t *testing.T) {
	compareFifthSceneryCreationNativeTrace(t, "fifth-scenery-creation.csv", false)
}

func TestEveryFifthSceneryCrowdedLifecycleNativeTraceOptional(t *testing.T) {
	compareFifthSceneryCreationNativeTrace(t, "fifth-scenery-lifecycle.csv", true)
}

func compareFifthSceneryCreationNativeTrace(t *testing.T, name string, lifecycle bool) {
	t.Helper()
	data, _, _ := nativeWaveReferenceData(t)
	codes := nativeRuntimeTileCodes(t, data)
	var w *World
	first := 0
	frame, phase := 0, -1
	previous := [4]int{-1, -1, -1, -1}
	seen := make(map[[2]int]bool)
	nextSlot, cases, boundaries, rows := ActorPoolCapacity, 0, 0, 0
	var families [4]int
	columns := 43
	if lifecycle {
		columns = 45
	}
	nativeCombatRows(t, name, func(v []int64) {
		if len(v) != columns {
			t.Fatalf("fifth scenery creation trace has %d columns, want %d", len(v), columns)
		}
		key := [4]int{int(v[0]), int(v[1]), int(v[2]), int(v[3])}
		if key != previous {
			if nextSlot != ActorPoolCapacity || key[0] != 5 || key[1] < 0 || key[1] >= len(data[4].Encounters.Fixed) || key[2] < 0 || key[2] > 3 || key[3] < 0 || !lifecycle && key[3] > 1 || lifecycle && key[3] != 0 && key[3] != 1 && key[3] != 2 && key[3] != 3 && key[3] != 64 && key[3] != 65 {
				t.Fatalf("invalid or incomplete fifth scenery creation boundary %v after %v", key, previous)
			}
			r := data[4].Encounters.Fixed[key[1]]
			if key[3] == 0 {
				identity := [2]int{key[1], key[2]}
				if seen[identity] || previous[3] == 0 {
					t.Fatalf("duplicated or unfinished fifth scenery creation case %v", key)
				}
				seen[identity] = true
				w, first = newCrowdedFifthSceneryWorld(t, data[4], key[2])
				w.Ready, w.MaterializationFrames, w.InvulnerableFrames = false, 0, 0
				w.playerCollision = CollisionRect{Left: 1000, Top: 1000, Right: 1020, Bottom: 1020}
				frame, phase = 0, -1
				clear(w.Level.Terrain.Map)
				w.ScrollY, w.MaximumScrollY, w.ScrollDelta = r.Y-108, 4607, 0
				family := -1
				for index, kind := range []int{1, 3, 4, 9} {
					if kind == r.EnemyKind {
						family = index
					}
				}
				if family < 0 {
					t.Fatalf("unexpected scenery family %d", r.EnemyKind)
				}
				for _, kind := range w.Level.FixedTiles.Kinds {
					if kind.Kind != r.EnemyKind {
						continue
					}
					for _, change := range kind.InitialChanges {
						position := ((r.Y-8)/16)*w.Level.Terrain.Columns + (r.X-8)/16 + change.ColumnOffset
						w.setSecondMapCell(position%w.Level.Terrain.Columns, position/w.Level.Terrain.Columns, change.Before)
					}
				}
				cases++
				families[family]++
			} else if key[3] == 1 {
				if previous != ([4]int{key[0], key[1], key[2], 0}) || !w.spawnFifthTile(r) || w.poolError != nil {
					t.Fatalf("fifth scenery constructor %v failed after %v: %v", key, previous, w.poolError)
				}
			} else {
				wantPrevious := key[3] - 1
				if key[3] == 64 {
					wantPrevious = 3
				}
				if previous != ([4]int{key[0], key[1], key[2], wantPrevious}) || int(v[43])*2+int(v[44])+2 != key[3] {
					t.Fatalf("nonconsecutive scenery lifecycle boundary %v after %v", key, previous)
				}
				for frame < int(v[43]) || phase < int(v[44]) {
					phase++
					if phase > 1 {
						frame, phase = frame+1, 0
					}
					w.Frame, w.Player.X, w.Player.Y = uint64(frame+1), 128+(frame*3)%64, 176
					var err error
					if phase == 0 {
						err = w.advanceActorPhase(ActorPoolMoving, Input{})
					} else {
						err = w.advancePooledProjectiles(Input{})
					}
					if err != nil || w.poolError != nil {
						t.Fatalf("scenery lifecycle %v failed: %v/%v", key, err, w.poolError)
					}
				}
			}
			if first != int(v[5]) || w.Pool.FreeFirst() != int(v[6]) || w.ScrollY != int(v[40]) || terrainDamageMapHash(t, w, codes[4]) != uint32(v[39]) || w.random != (RandomState{A: uint32(v[33]), B: uint32(v[34])}) || w.fifthDestroyedTurrets[0] != (v[41] != 0) || w.fifthDestroyedTurrets[1] != (v[42] != 0) {
				t.Fatalf("fifth scenery creation %v free %d RNG %+v map %08x persistent %v differs from source %v", key, w.Pool.FreeFirst(), w.random, terrainDamageMapHash(t, w, codes[4]), w.fifthDestroyedTurrets, v)
			}
			for index := range 2 {
				if w.WaveBonuses.Entries[index] != (WaveBonusEntry{ID: uint16(v[35+index*2]), Remaining: uint16(v[36+index*2])}) {
					t.Fatalf("fifth scenery creation %v changed original reward bucket %d", key, index)
				}
			}
			previous, nextSlot = key, first
			boundaries++
		}
		if int(v[4]) != nextSlot {
			t.Fatalf("fifth scenery creation %v skipped slot %d for %d", key, nextSlot, v[4])
		}
		nextSlot++
		// Normalize the case-column offsets to the shared physical-state schema.
		var physical [34]int64
		physical[5] = v[4]
		copy(physical[8:], v[7:33])
		compareDamageNativePoolRow(t, key, w, physical[:])
		rows++
	})
	wantBoundaries, wantRows, final := 176, 27104, 1
	if lifecycle {
		wantBoundaries, wantRows, final = 528, 81312, 65
	}
	if cases != 88 || boundaries != wantBoundaries || rows != wantRows || families != [4]int{20, 40, 20, 8} || nextSlot != ActorPoolCapacity || previous[3] != final {
		t.Fatalf("incomplete fifth scenery creation coverage: cases %d boundaries %d rows %d families %v", cases, boundaries, rows, families)
	}
	t.Logf("Compared %d crowded fifth scenery constructor cases, %d boundaries and %d physical-slot states", cases, boundaries, rows)
}
