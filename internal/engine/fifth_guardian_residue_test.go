package engine

import (
	"strconv"
	"testing"
	"xenon2/internal/visualassets"
)

func releasedOriginalCannonMount(t *testing.T, w *World) (int, ActorResidue) {
	t.Helper()
	w.MaterializationFrames = 0
	w.Equipment.ApplyItem(ItemCannon)
	if err := w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false)); err != nil {
		t.Fatal(err)
	}
	mount := w.Weapons.mounts[1]
	if mount.Item != ItemCannon || mount.Binding.EntityID == 0 {
		t.Fatal("original cannon mount did not allocate")
	}
	residue := w.Pool.Slot(mount.Binding.Slot).Residue
	if residue.EmitterClock != 0xffff {
		t.Fatal("original cannon did not leave its native emitter word")
	}
	if _, err := (ShopRules{Level: 5}).Sell(&w.Equipment, &w.Money, SaleMount0); err != nil {
		t.Fatal(err)
	}
	if err := w.Weapons.SynchronizeEquipment(w.weaponContext(Input{}, false)); err != nil {
		t.Fatal(err)
	}
	w.releaseDeadPoolEntries(ActorPoolProjectile)
	w.releaseDeadPoolEntries(ActorPoolEquipment)
	if w.Pool.FreeFirst() != mount.Binding.Slot {
		t.Fatal("sold cannon did not return its physical mount slot")
	}
	return mount.Binding.Slot, residue
}

func TestOriginalFifthMiddleContactRetainsReusedStrengthOptional(t *testing.T) {
	for _, reused := range []bool{false, true} {
		for _, index := range []int{1, 5} {
			t.Run(strconv.FormatBool(reused)+"/"+strconv.Itoa(index), func(t *testing.T) {
				w := fifthResourceWorld(t)
				w.Ready, w.MaterializationFrames = false, 0
				w.Level.Encounters = &visualassets.Encounters{}
				w.ScrollY, w.ScrollDelta = 2240, 0
				if reused {
					var precursor visualassets.FixedEncounter
					for _, record := range w.Level.FixedSprites.Kinds {
						if record.Kind == 8 {
							precursor = visualassets.FixedEncounter{EnemyKind: 8, X: 280, Y: 2800}
							break
						}
					}
					if precursor.EnemyKind != 8 {
						t.Fatal("original strong oscillator is missing")
					}
					reclaimOriginalStrongFixedSlots(t, w, precursor, 10)
					w.ScrollY, w.ScrollDelta = 2240, 0
				}
				if err := w.activateFifthGuardian(visualassets.FixedEncounter{Y: 2336}, false); err != nil {
					t.Fatal(err)
				}
				w.advanceFifthGuardian(false)
				target := w.fifthMiddleActors[index]
				w.Player = PlayerMotionState{X: (target.Collision.Left + target.Collision.Right) / 2, Y: (target.Collision.Top + target.Collision.Bottom) / 2}
				w.PreviousPlayer = w.Player
				if err := w.Step(Input{}); err != nil {
					t.Fatal(err)
				}
				want := 31
				if reused {
					want = 23
				}
				if !w.PlayerAlive || w.Equipment.Shield != want || target.part.StrongHealth != reused || w.Pool.Slot(target.Binding.Slot).Residue.StrongHealth != reused {
					t.Fatalf("middle physical contact strength: shield%d want%d descriptor%v slot%v", w.Equipment.Shield, want, target.part.StrongHealth, w.Pool.Slot(target.Binding.Slot).Residue.StrongHealth)
				}
			})
		}
	}
}

// Both source guardian constructors leave 0x5e/0x5f intact. Middle body updates
// both bytes, while the final body's mouth timer uses only the high byte.
func TestOriginalFifthBodyInheritsEmitterFromReleasedCannonOptional(t *testing.T) {
	for _, final := range []bool{false, true} {
		t.Run(strconv.FormatBool(final), func(t *testing.T) {
			w := fifthResourceWorld(t)
			slot, residue := releasedOriginalCannonMount(t, w)
			var offsets [ActorPoolCapacity][2]int16
			for index := range offsets {
				r := w.Pool.Slot(index).Residue
				offsets[index] = [2]int16{r.MountOffsetX, r.MountOffsetY}
			}
			if err := w.activateFifthGuardian(visualassets.FixedEncounter{Y: 2336}, final); err != nil {
				t.Fatal(err)
			}
			body := w.fifthMiddleActors[0]
			state := w.FifthMiddle
			if final {
				body = w.fifthFinalActors[0]
				state = nil
			}
			if body.Binding.Slot != slot {
				t.Fatal("body did not reclaim the sold cannon mount")
			}
			actors := w.fifthMiddleActors[:]
			if final {
				actors = w.fifthFinalActors[:]
			}
			for _, component := range actors {
				r := w.Pool.Slot(component.Binding.Slot).Residue
				if r.OwnerSlot != slot || r.Counter != 0 || r.Direction != 4 || r.WaveBonusToken != 0 || [2]int16{r.MountOffsetX, r.MountOffsetY} != offsets[component.Binding.Slot] {
					t.Fatal("guardian constructor lost its physical owner, phase, direction or retained spare offsets")
				}
			}
			part := w.fifthPartState(body)
			if part.FireAccumulator != residue.FireAccumulator() || part.SecondaryAccumulator != residue.FireRate() {
				t.Fatalf("body birth lost emitter %04x: fire%02x secondary%02x", residue.EmitterClock, part.FireAccumulator, part.SecondaryAccumulator)
			}
			before := w.RandomState()
			w.Player.X, w.Player.Y = 300, 176
			w.ScrollDelta = 0
			if final {
				expected := *w.FifthFinal
				random := before
				expected.Advance(w.fifthFinalArt, w.ScrollY, 0, w.BaseScrollStep, w.MaximumScrollY, int(w.Frame), 300, 176, &random)
				w.advanceFifthGuardian(true)
				if w.FifthFinal.Parts[0] != expected.Parts[0] || w.RandomState() != random {
					t.Fatal("final inherited emitter did not follow its actual first callback")
				}
			} else {
				expected := *state
				random := before
				events := expected.Advance(w.fifthMiddleArt, w.ScrollY, 0, w.MaximumScrollY, 300, 176, &random)
				w.advanceFifthGuardian(false)
				if w.FifthMiddle.Parts[0] != expected.Parts[0] || w.RandomState() != random || len(events.Shots) != 2 {
					t.Fatal("middle inherited secondary carry did not emit the source side pair")
				}
			}
			physical := w.Pool.Slot(slot).Residue
			if final {
				wantClock := 2
				if w.Frame&1 == 0 {
					wantClock = -2
				}
				if part.Clock != wantClock || part.FireAccumulator >= 64 || part.SecondaryAccumulator != 255 || w.RandomState() == before {
					t.Fatal("native inherited mouth carry did not start its signed clock and random reset")
				}
			} else {
				if part.Clock != 1 || part.MoveRemaining != 3 || part.FireAccumulator >= 64 || part.SecondaryAccumulator >= 64 || w.RandomState() == before {
					t.Fatal("native inherited body carries did not start the clock and reset both bytes")
				}
			}
			if physical.Counter != int16(part.Clock) || physical.EmitterClock != uint16(part.FireAccumulator)<<8|uint16(part.SecondaryAccumulator) {
				t.Fatal("live body phase/emitter was not published in its physical slot")
			}
			if physical.XFraction != residue.XFraction || physical.YFraction != residue.YFraction || physical.PowerOrScore != residue.PowerOrScore {
				t.Fatal("whole-pixel body callback erased untouched physical words")
			}
		})
	}
}
