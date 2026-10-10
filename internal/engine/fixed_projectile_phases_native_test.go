package engine

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"xenon2/internal/visualassets"
)

func nativeFixedProjectileImages(t *testing.T, data [5]LevelData) [5]map[int]string {
	t.Helper()
	var result [5]map[int]string
	for index, name := range []string{"000B00E5", "00FA00FE", "02020113", "031F0159", "04820138"} {
		raw, err := os.ReadFile(filepath.Join(filepath.Dir(os.Getenv("XENON2_NATIVE_TRACE_DIR")), "imported", name+".decoded"))
		if err != nil {
			t.Fatal(err)
		}
		bank, err := visualassets.DecodeFixedSprites(index+1, raw, data[index].Terrain.Palette)
		if err != nil {
			t.Fatal(err)
		}
		result[index] = bank.Atlas.SourceSpriteNames
		result[index][int(binary.BigEndian.Uint32(raw[0x28:]))] = data[index].Rules.DefaultEnemyShot
		guardians, _, err := visualassets.DecodeGuardianArt(index+1, raw, data[index].Terrain.Palette)
		if err != nil {
			t.Fatal(err)
		}
		if guardians != nil {
			for address, name := range guardians.Atlas.SourceSpriteNames {
				result[index][address] = name
			}
		}
	}
	return result
}

func newFixedProjectilePhaseReferenceWorld(t *testing.T, data LevelData, record, reused int) (*World, int) {
	t.Helper()
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	first := w.Pool.FreeFirst()
	for slot := first; slot != NoActorSlot; slot = w.Pool.Slot(slot).freeNext {
		if reused != 0 {
			w.Pool.Slot(slot).Residue = waveConstructorResidueFixture(slot)
		}
	}
	w.WaveBonuses.BeginPass(0)
	r := data.Encounters.Fixed[record]
	w.ScrollY, w.MaximumScrollY, w.ScrollDelta = r.Y-108, 4607, 0
	w.Player.X, w.Player.Y = 160, 96
	w.Ready, w.MaterializationFrames, w.InvulnerableFrames = false, 0, 0
	// Player updates and contacts are isolated in both recordings, while
	// ordinary aiming still receives the real changing ship coordinates.
	w.playerCollision = CollisionRect{Left: 1000, Top: 1000, Right: 1020, Bottom: 1020}
	w.spawnFixed(r)
	if len(w.Actors) == 0 || w.Actors[0].fixedKind == nil {
		t.Fatal("actual record did not construct an ordinary fixed sprite")
	}
	return w, first
}

func advanceFixedProjectileReferencePhase(t *testing.T, w *World, frame, phase, origin int) {
	t.Helper()
	if phase == 0 {
		w.Frame = uint64(frame + 1)
		w.ScrollY, w.ScrollDelta = origin-(frame+3)/4, 0
		if frame%4 == 0 {
			w.ScrollDelta = 1
		}
		w.Player.X, w.Player.Y = 128+(frame*3)%64, 96
		w.SoundRequests = [4]string{}
		if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
			t.Fatal(err)
		}
	} else if err := w.advancePooledProjectiles(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.poolError != nil {
		t.Fatal(w.poolError)
	}
}

func TestEveryFixedSpriteAndProjectilePhaseNativeTraceOptional(t *testing.T) {
	data, _, common := nativeWaveReferenceData(t)
	images := nativeFixedProjectileImages(t, data)
	previous := [5]int{-1, -1, -1, -1, -1}
	seen := make(map[[5]int]bool)
	var w *World
	first, origin, currentFrame, currentPhase := 0, 0, 0, -1
	nextSlot, cases, rows := ActorPoolCapacity, 0, 0
	var levels [5]int
	nativeCombatRows(t, "fixed-projectile-phases.csv", func(v []int64) {
		if len(v) != 70 {
			t.Fatalf("fixed/projectile trace has %d columns, want 70", len(v))
		}
		key := [5]int{int(v[0]), int(v[1]), int(v[2]), int(v[58]), int(v[59])}
		if key != previous {
			if nextSlot != ActorPoolCapacity || seen[key] || key[0] < 1 || key[0] > 5 || key[1] < 0 || key[1] >= len(data[key[0]-1].Encounters.Fixed) || key[2] < 0 || key[2] > 1 || key[3] < 0 || key[3] >= 160 || key[4] < 0 || key[4] > 1 {
				t.Fatalf("invalid or incomplete fixed/projectile phase %v after %v", key, previous)
			}
			seen[key] = true
			if key[0] != previous[0] || key[1] != previous[1] || key[2] != previous[2] {
				if key[3] != 0 || key[4] != 0 {
					t.Fatal("original fixed/projectile case lacks its first moving phase")
				}
				w, first = newFixedProjectilePhaseReferenceWorld(t, data[key[0]-1], key[1], key[2])
				origin = data[key[0]-1].Encounters.Fixed[key[1]].Y - 108
				currentFrame, currentPhase = 0, -1
			}
			for currentFrame < key[3] || currentPhase < key[4] {
				currentPhase++
				if currentPhase > 1 {
					currentPhase = 0
					currentFrame++
				}
				advanceFixedProjectileReferencePhase(t, w, currentFrame, currentPhase, origin)
			}
			if first != int(v[6]) || w.Pool.FreeFirst() != int(v[7]) || w.ScrollY != int(v[60]) || w.ScrollDelta != int(v[61]) || w.Player.X != int(v[62]) || w.Player.Y != int(v[63]) || w.random != (RandomState{A: uint32(v[34]), B: uint32(v[35])}) || w.Score != int(v[36]) || w.SoundRequests[1] != nativeRewardSound(v[37]) || w.SoundRequests[2] != nativeRewardSound(v[38]) {
				t.Fatalf("fixed/projectile phase %v: free %d scroll %d RNG %+v score %d sounds %v, original %v", key, w.Pool.FreeFirst(), w.ScrollY, w.random, w.Score, w.SoundRequests, v)
			}
			if w.WaveBonuses.NextID != uint16(v[39]) || w.WaveBonuses.NormalID != uint16(v[40]) || w.WaveBonuses.HeavyID != uint16(v[41]) {
				t.Fatalf("fixed/projectile phase %v changed original reward cache", key)
			}
			for index, bucket := range w.WaveBonuses.Entries {
				if bucket != (WaveBonusEntry{ID: uint16(v[42+index*2]), Remaining: uint16(v[43+index*2])}) {
					t.Fatalf("fixed/projectile phase %v changed reward bucket %d", key, index)
				}
			}
			previous, nextSlot = key, first
			cases++
			levels[key[0]-1]++
		}
		if int(v[5]) != nextSlot {
			t.Fatalf("fixed/projectile phase %v skipped slot %d for %d", key, nextSlot, v[5])
		}
		nextSlot++
		compareDamageNativePoolRow(t, key, w, v)
		index := int(v[5])
		slot := w.Pool.Slot(index)
		if slot.allocated && slot.ResourceTag != 4 {
			want := images[key[0]-1][int(v[64])]
			if name, ok := common[int(v[64])]; ok {
				want = name
			}
			if actor := w.poolActors[index]; actor != nil && actor.Active {
				remaining := actor.animationState.Remaining
				if actor.fixedAiming != nil {
					remaining = actor.fixedAiming.Animation.Remaining
				}
				if actor.Sprite != want || remaining != int(v[65]) {
					t.Fatalf("fixed/projectile phase %v slot %d image %s/%d, original %s/%d", key, index, actor.Sprite, remaining, want, v[65])
				}
				if actor.ActorList == "moving" {
					if v[66] == 1000 && v[68] == 1000 {
						if !actor.Collision.Empty() {
							t.Fatalf("fixed/projectile phase %v slot %d exposed collision before its original callback", key, index)
						}
					} else if actor.Collision != (CollisionRect{Left: int(v[66]), Top: int(v[67]), Right: int(v[68]), Bottom: int(v[69])}) {
						t.Fatalf("fixed/projectile phase %v slot %d collision %+v, original %v", key, index, actor.Collision, v[66:70])
					}
				}
			} else if shot := w.poolProjectiles[index]; shot != nil && shot.Active && shot.Sprite != want {
				t.Fatalf("fixed/projectile phase %v slot %d shot image %s, original %s", key, index, shot.Sprite, want)
			}
		}
		rows++
	})
	if cases != 1720 || rows != 263040 || nextSlot != ActorPoolCapacity || levels != [5]int{440, 160, 760, 0, 360} {
		t.Fatalf("incomplete fixed/projectile phases: cases %d rows %d levels %v", cases, rows, levels)
	}
	t.Logf("Compared %d mixed fixed/projectile phase boundaries and %d physical-slot states", cases, rows)
}
