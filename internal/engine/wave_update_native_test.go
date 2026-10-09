package engine

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func newWaveUpdateReferenceWorld(t *testing.T, data LevelData, record int) (*World, [ActorPoolCapacity]*WorldActor) {
	t.Helper()
	w, err := NewWorld(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, actor := range w.Actors {
		actor.Active = false // The native fixture isolates unrelated stage callbacks.
	}
	first := w.Pool.FreeFirst()
	for slot := first; slot != NoActorSlot; slot = w.Pool.Slot(slot).freeNext {
		w.Pool.Slot(slot).Residue = waveConstructorResidueFixture(slot)
	}
	w.WaveBonuses.BeginPass(0)
	if err := w.spawnWave(data.Encounters.Moving[record]); err != nil {
		t.Fatal(err)
	}
	var born [ActorPoolCapacity]*WorldActor
	for _, actor := range w.Actors {
		if actor.Active && actor.Binding.Slot >= first {
			born[actor.Binding.Slot] = actor
		}
	}
	return w, born
}

func advanceWaveUpdateReferenceWorld(t *testing.T, w *World) {
	t.Helper()
	// Fire/RNG callbacks run normally. Emitted bullets are observed separately
	// and cleared before the next moving pass, keeping allocation available.
	for _, shot := range w.Projectiles {
		shot.Active = false
		w.releaseWorldActor(shot.Binding)
	}
	w.Projectiles = w.Projectiles[:0]
	w.Frame++
	if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
		t.Fatal(err)
	}
}

func TestAllMovingWaveUpdatesNativeTraceOptional(t *testing.T) {
	data, sprites, commonSprites := nativeWaveReferenceData(t)
	var w *World
	var born [ActorPoolCapacity]*WorldActor
	lastLevel, lastRecord, lastFrame := -1, -1, -1
	cases, passes, states := 0, 0, 0
	nativeCombatRows(t, "all-wave-updates-trace.csv", func(v []int64) {
		level, record, frame, slot := int(v[0]), int(v[1]), int(v[2]), int(v[3])
		if level != lastLevel || record != lastRecord {
			w, born = newWaveUpdateReferenceWorld(t, data[level-1], record)
			lastLevel, lastRecord, lastFrame = level, record, -1
			cases++
		}
		if frame != lastFrame {
			if frame != lastFrame+1 {
				t.Fatalf("nonconsecutive original wave update: %v", v[:4])
			}
			advanceWaveUpdateReferenceWorld(t, w)
			lastFrame = frame
			passes++
		}
		a := born[slot]
		if a == nil {
			t.Fatalf("missing level %d wave %d original slot %d", level, record, slot)
		}
		active := v[4] != 0 && v[4] != 4
		if a.Active != active {
			t.Fatalf("level %d wave %d pass %d slot %d: active %v, original tag %d", level, record, frame, slot, a.Active, v[4])
		}
		states++
		if !active {
			return // Freed slots and projectile pressure have separate comparisons.
		}
		r := a.Binding.Residue
		x := int32(r.X)<<16 | int32(r.XFraction)
		y := int32(r.Y)<<16 | int32(r.YFraction)
		angle := uint32(uint16(r.Direction))<<16 | uint32(r.HorizontalDriftRemainder)
		name := sprites[level-1][int(v[23])]
		if a.Atlas == "common" {
			name = commonSprites[int(v[23])]
		}
		collision := CollisionRect{Left: int(v[25]), Top: int(v[26]), Right: int(v[27]), Bottom: int(v[28])}
		if x != int32(v[5]) || y != int32(v[6]) || angle != uint32(v[7]) || r.Counter != int16(v[10]) || r.MotionBudget != int16(v[11]) || r.EmitterClock != uint16(v[12]) || r.Health != uint16(v[13]) || r.StrongHealth != (v[14] != 0) || r.PowerOrScore != uint16(v[15]) || r.WaveBonusToken != uint16(v[16]) || r.MountOffsetX != int16(v[17]) || r.MountOffsetY != int16(v[18]) || r.VerticalVelocity != int16(v[19]) || r.VerticalFraction != uint16(v[20]) || r.OwnerSlot != int(v[21]) || r.FollowingSlot != int(v[22]) || a.Sprite != name || a.animationState.Remaining != int(v[24]) || a.Collision != collision || w.Pool.Slot(slot).Linked != (v[29] != 0) {
			t.Fatalf("level %d wave %d pass %d slot %d: residue %+v sprite %s remaining %d collision %+v; original %v (%s)", level, record, frame, slot, r, a.Sprite, a.animationState.Remaining, a.Collision, v, name)
		}
	})
	if cases != 600 || passes != 38400 || states != 97536 {
		t.Fatalf("incomplete wave callback coverage: %d nonempty waves, %d moving passes, %d actor states", cases, passes, states)
	}
	t.Logf("Compared %d actual waves over %d original moving-list passes and %d actor states", cases, passes, states)
}

func TestAllMovingWaveFireAndSharedStateNativeTraceOptional(t *testing.T) {
	data, _, _ := nativeWaveReferenceData(t)
	var image [5]uint32
	for level, name := range []string{"000B00E5", "00FA00FE", "02020113", "031F0159", "04820138"} {
		raw, err := os.ReadFile(filepath.Join(filepath.Dir(os.Getenv("XENON2_NATIVE_TRACE_DIR")), "imported", name+".decoded"))
		if err != nil {
			t.Fatal(err)
		}
		image[level] = binary.BigEndian.Uint32(raw[0x28:])
	}
	type frameKey struct{ level, record, frame int }
	emissions := make(map[frameKey][][5]int)
	shots := 0
	nativeCombatRows(t, "all-wave-updates-shots.csv", func(v []int64) {
		key := frameKey{int(v[0]), int(v[1]), int(v[2])}
		if int(v[3]) != len(emissions[key]) || uint32(v[8]) != image[key.level-1] {
			t.Fatalf("original point-shot order/image changed: %v", v)
		}
		emissions[key] = append(emissions[key], [5]int{int(v[4]), int(v[5]), int(v[6]), int(v[7]), int(v[8])})
		shots++
	})
	var w *World
	var born [ActorPoolCapacity]*WorldActor
	var levels [5]int
	cases, passes, observed := 0, 0, 0
	nativeCombatRows(t, "all-wave-updates-state.csv", func(v []int64) {
		level, record, frame := int(v[0]), int(v[1]), int(v[2])
		if frame == 0 {
			w, born = newWaveUpdateReferenceWorld(t, data[level-1], record)
			cases++
			levels[level-1]++
		}
		advanceWaveUpdateReferenceWorld(t, w)
		live := 0
		for _, actor := range born {
			if actor != nil && actor.Active {
				live++
			}
		}
		cache := w.WaveBonuses
		if live != int(v[3]) || len(w.Projectiles) != int(v[5]) || w.random.A != uint32(v[6]) || w.random.B != uint32(v[7]) || cache.Entries[0] != (WaveBonusEntry{ID: uint16(v[8]), Remaining: uint16(v[9])}) || cache.Entries[1] != (WaveBonusEntry{ID: uint16(v[10]), Remaining: uint16(v[11])}) {
			t.Fatalf("original wave update state %v: live %d shots %d RNG %+v cache %+v", v, live, len(w.Projectiles), w.random, cache)
		}
		want := emissions[frameKey{level, record, frame}]
		if len(want) != len(w.Projectiles) {
			t.Fatalf("incomplete shot reference for level %d wave %d pass %d", level, record, frame)
		}
		for i, event := range want {
			// Production insertion prepends each shot; native factory calls are
			// recorded in creation order, including multiple shots in one pass.
			shot := w.Projectiles[len(want)-1-i]
			if int(shot.X) != event[0] || int(shot.Y) != event[1] || int(shot.Motion.Direction) != event[2] || shot.Motion.Speed != event[3] || shot.Sprite != data[level-1].Rules.DefaultEnemyShot {
				t.Fatalf("level %d wave %d pass %d emission %d: shot %+v, original %v", level, record, frame, i, shot, event)
			}
			observed++
		}
		passes++
	})
	if cases != 601 || passes != 38464 || shots != 880 || observed != shots || levels != [5]int{113, 87, 149, 103, 149} {
		t.Fatalf("incomplete wave fire/shared-state coverage: %d cases, %d passes, %d/%d shots, levels %v", cases, passes, observed, shots, levels)
	}
	t.Logf("Compared %d original point-shot emissions and %d shared RNG/bonus/liveness states across all %d real waves", shots, passes, cases)
}
