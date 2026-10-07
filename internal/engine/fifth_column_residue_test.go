package engine

import (
	"strconv"
	"testing"
)

// Column creation at 0x56144/0x5679e writes X/Y, length0 and signed speed.
// Its updater 0x56940 changes length/Y but leaves unrelated physical data intact.
func TestOriginalFifthColumnPublishesLengthAndRetainsSlotStateOptional(t *testing.T) {
	for _, direction := range []int{-1, 1} {
		t.Run(strconv.Itoa(direction), func(t *testing.T) {
			w := fifthResourceWorld(t)
			w.MaterializationFrames = 0
			w.Player.X, w.Player.Y = 300, 176
			w.updatePlayerCollision()
			slot := w.Pool.FreeFirst()
			want := secondResidueFixture()
			w.Pool.Slot(slot).Residue = want
			speed := direction * w.fifthMiddleArt.MotionParameters["body_laser_speed"]
			w.spawnFifthColumn(FifthGuardianLaser{X: 150, Y: 40, Speed: speed})
			actor := w.Actors[len(w.Actors)-1]
			if actor.fifthColumn == nil || actor.Binding.Slot != slot {
				t.Fatal("original column did not reclaim its expected projectile slot")
			}
			want.X, want.Y, want.Counter, want.Direction = 150, 40, 0, int16(speed)
			check := func() {
				t.Helper()
				if got := w.Pool.Slot(slot).Residue; got != want {
					t.Fatalf("column physical state: got %+v want %+v", got, want)
				}
			}
			check()
			for pass := 0; actor.Active && pass < 40; pass++ {
				w.ScrollDelta = 1
				if pass%5 == 0 {
					w.ScrollDelta = -1
				}
				if err := w.advancePooledProjectiles(Input{}); err != nil {
					t.Fatal(err)
				}
				want.Y, want.Counter = int16(actor.fifthColumn.Y), int16(actor.fifthColumn.Length)
				check()
			}
			if actor.Active || w.Pool.Slot(slot).ResourceTag != 4 {
				t.Fatal("full column did not expire at its native screen boundary")
			}
			if err := w.advancePooledProjectiles(Input{}); err != nil {
				t.Fatal(err)
			}
			if w.Pool.Slot(slot).allocated || w.Pool.FreeFirst() != slot {
				t.Fatal("expired column did not return its physical slot")
			}
			check()
			w.Equipment.ApplyItem(ItemFlamer)
			if err := w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false)); err != nil {
				t.Fatal(err)
			}
			mount := w.Weapons.mounts[0]
			if mount.Binding.Slot != slot || mount.FlamerSound.Counter != 48 || !mount.FlamerSound.Started {
				t.Fatal("next flamer owner lost the source column's retained length")
			}
			if err := w.Weapons.AdvanceEquipment(w.weaponContext(Input{}, false)); err != nil {
				t.Fatal(err)
			}
			if !w.StopEffectsRequested {
				t.Fatal("inherited nonzero column length did not trigger native flamer release cleanup")
			}
		})
	}
}
