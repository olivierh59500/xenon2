package engine

import (
	"fmt"
	"testing"
)

func newRewardConstructionReferenceWorld(t *testing.T, data LevelData, kind, profile int) (*World, []int) {
	t.Helper()
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	first := w.Pool.FreeFirst()
	for slot := first; slot != NoActorSlot; slot = w.Pool.Slot(slot).freeNext {
		w.Pool.Slot(slot).Residue = waveConstructorResidueFixture(slot)
	}
	w.random = RandomState{A: 0x630c1592 + uint32(kind*7919), B: 0x35358979 - uint32(kind*173)}
	fill := 0
	if profile != 0 {
		fill = ActorPoolCapacity - first
		if profile == 1 {
			fill -= 2
		}
	}
	for index := range fill {
		w.spawnEnemyShot(80+index%20*8, 48+index%12*8, EnemyShot{Direction: uint8(index & 7), Speed: 4 + index%5})
	}
	w.PlayerAlive = false // Isolate motion and constructor state from collection.
	w.SoundRequests = [4]string{}
	switch {
	case kind < 2:
		w.spawnWaveCash(96, 88, kind == 1)
	case kind <= 20:
		w.spawnPickup(kind-2, 96, 88)
	case kind <= 25:
		w.spawnExitCash([5]int{1, 3, 5, 9, 10}[kind-21])
	default:
		name := "explosion-small"
		if kind == 27 {
			name = "explosion-large"
		}
		w.spawnSecondNamedExplosion(96, 88, name)
	}
	if w.poolError != nil {
		t.Fatal(w.poolError)
	}
	var born []int
	for slot := w.Pool.First(ActorPoolProjectile); slot != NoActorSlot; slot = w.Pool.Next(slot) {
		if slot >= first && w.Pool.Slot(slot).ResourceTag != 20 {
			born = append(born, slot)
		}
	}
	return w, born
}

func nativeRewardSound(code int64) string {
	if code == 0 {
		return ""
	}
	family := "synthesized"
	if code >= 128 {
		family = "sampled"
		code -= 128
	}
	return fmt.Sprintf("%s-effect-%02d", family, code)
}

func TestRewardAndExplosionConstructorsNativeTraceOptional(t *testing.T) {
	data, _, images := nativeWaveReferenceData(t)
	var w *World
	var born []int
	previous := [3]int{-1, -1, -1}
	cases, boundaries, rows, next := 0, 0, 0, 0
	nativeCombatRows(t, "reward-construction.csv", func(v []int64) {
		key := [3]int{int(v[0]), int(v[1]), int(v[2])}
		if key != previous {
			if next != len(born) {
				t.Fatalf("incomplete original reward boundary %v", previous)
			}
			if key[2] == 0 {
				w, born = newRewardConstructionReferenceWorld(t, data[0], key[0], key[1])
				cases++
			} else {
				if key != ([3]int{previous[0], previous[1], previous[2] + 1}) {
					t.Fatalf("nonconsecutive original reward boundary %v after %v", key, previous)
				}
				for _, slot := range born {
					if item := w.poolCollectibles[slot]; item != nil && item.Active {
						w.advanceCollectible(item)
						w.finishCollectibleUpdate(item)
					} else if actor := w.poolActors[slot]; actor != nil && actor.Active {
						if err := w.advanceTransientActor(actor); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
			previous, next = key, 0
			boundaries++
		}
		order, index := int(v[3]), int(v[4])
		if order != next || order >= len(born) || born[order] != index {
			t.Fatalf("reward %v order%d: native slot%d, Go slots%v", key, order, index, born)
		}
		next++
		slot := w.Pool.Slot(index)
		if w.Pool.FreeFirst() != int(v[6]) || slot.previous != int(v[7]) || slot.next != int(v[8]) || slot.ResourceTag != int16(v[9]) || slot.Linked != (v[10] != 0) || slot.AuxiliaryFlags != ([2]bool{v[11] != 0, v[12] != 0}) {
			t.Fatalf("reward %v slot%d: free/list/tag/flags differ: Go%+v, original%v", key, index, slot, v[6:13])
		}
		r := slot.Residue
		got := [19]int64{int64(r.X), int64(r.Y), int64(r.XFraction), int64(r.YFraction), int64(r.Counter), int64(r.Direction), int64(r.HorizontalDriftRemainder), int64(r.VerticalVelocity), int64(r.VerticalFraction), int64(r.Health), int64(boolCount(r.StrongHealth)), int64(r.PowerOrScore), int64(r.WaveBonusToken), int64(r.MotionBudget), int64(r.EmitterClock), int64(r.MountOffsetX), int64(r.MountOffsetY), int64(r.OwnerSlot), int64(r.FollowingSlot)}
		var want [19]int64
		copy(want[:], v[13:32])
		want[10] = int64(boolCount(v[23] != 0))
		if got != want {
			t.Fatalf("reward %v slot%d tag%d: fields%v, original%v", key, index, slot.ResourceTag, got, want)
		}
		image, remaining := "", 0
		if item := w.poolCollectibles[index]; item != nil {
			image, remaining = item.Sprite, item.animationState.Remaining
		} else if actor := w.poolActors[index]; actor != nil {
			image, remaining = actor.Sprite, actor.animationState.Remaining
		}
		if image != images[int(v[32])] || remaining != int(v[33]) || w.random != (RandomState{A: uint32(v[34]), B: uint32(v[35])}) || w.PendingExitDrops != int(v[36]) || w.SoundRequests[1] != nativeRewardSound(v[37]) || w.SoundRequests[2] != nativeRewardSound(v[38]) {
			t.Fatalf("reward %v slot%d: image%s remaining%d RNG%+v pending%d sounds%v; original%v", key, index, image, remaining, w.random, w.PendingExitDrops, w.SoundRequests, v[32:])
		}
		rows++
	})
	if cases != 84 || boundaries != 1092 || rows != 3081 || next != len(born) || previous != [3]int{27, 2, 12} {
		t.Fatalf("incomplete original reward comparison: cases%d boundaries%d rows%d last%v", cases, boundaries, rows, previous)
	}
	t.Logf("Compared %d reward/effect constructor cases, %d boundaries and %d native image/state rows", cases, boundaries, rows)
}
