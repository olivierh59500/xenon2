package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func TestThirdCannonContactRetainsSourceSlotStrengthOptional(t *testing.T) {
	for _, reused := range []bool{false, true} {
		name := "fresh"
		if reused {
			name = "reused-sweeper"
		}
		t.Run(name, func(t *testing.T) {
			data := originalWorldData(t, 3)
			var sweeper, cannon visualassets.FixedEncounter
			for _, record := range data.Encounters.Fixed {
				if record.EnemyKind == 2 && sweeper.EnemyKind == 0 {
					sweeper = record
				}
				if record.EnemyKind == 5 && cannon.EnemyKind == 0 {
					cannon = record
				}
			}
			if sweeper.EnemyKind != 2 || cannon.EnemyKind != 5 {
				t.Fatal("original sweeper and compound cannon records are required")
			}
			w, err := NewWorld(data)
			if err != nil {
				t.Fatal(err)
			}
			w.Ready, w.MaterializationFrames, w.InvulnerableFrames = false, 0, 0
			w.Level.Encounters = &visualassets.Encounters{}
			w.MaximumScrollY, w.VisitedScrollY = 4607, 4607
			slot := w.Pool.FreeFirst()
			if reused {
				w.ScrollY, w.ScrollDelta = sweeper.Y-8-120, 0
				w.spawnFixed(sweeper)
				var actor *WorldActor
				for _, candidate := range w.Actors {
					if candidate.fixedKind != nil && candidate.fixedKind.Kind == 2 {
						actor = candidate
						break
					}
				}
				if actor == nil || !actor.part.StrongHealth || actor.part.ResourceTag != 208 {
					t.Fatal("source horizontal sweeper must initialize a strong moving slot")
				}
				slot = actor.Binding.Slot
				w.Player.X, w.Player.Y = int(actor.X)-16, int(actor.Y)-6
				if err := w.Step(Input{}); err != nil {
					t.Fatal(err)
				}
				if actor.Active || w.Equipment.Shield != 23 || w.Pool.Slot(slot).allocated || w.Pool.FreeFirst() != slot {
					t.Fatal("real sweeper contact must destroy it and reclaim its physical slot")
				}
			}
			if w.Pool.Slot(slot).Residue.StrongHealth != reused {
				t.Fatal("source allocator must retain the preceding slot strength")
			}
			w.ScrollY, w.ScrollDelta = cannon.Y-8+64-176, 0
			w.Player.X, w.Player.Y = 280, 176
			w.spawnFixed(cannon)
			var actor *WorldActor
			for _, candidate := range w.Actors {
				if candidate.thirdCannon != nil {
					actor = candidate
					break
				}
			}
			if actor == nil || actor.Binding.Slot != slot || actor.part.ResourceTag != 248 || actor.Health != data.FixedSprites.Third.Cannon.Health[0] {
				t.Fatal("source compound cannon must reuse the selected slot with its own health and tag")
			}
			// The native 0x561a6 constructor and its common 0x345e helper
			// leave the strength byte unchanged; publishing residue must too.
			if actor.Binding.Residue.StrongHealth != reused || w.Pool.Slot(slot).Residue.StrongHealth != reused {
				t.Fatal("compound cannon construction erased retained contact strength")
			}
			if err := w.Step(Input{}); err != nil {
				t.Fatal(err)
			}
			w.Player.X, w.Player.Y = actor.thirdCannon.X+32, 176
			w.Equipment.Shield = 39
			score := w.Score
			if err := w.Step(Input{}); err != nil {
				t.Fatal(err)
			}
			wantShield := 31
			if reused {
				wantShield = 23
			}
			if w.Equipment.Shield != wantShield || !w.PlayerAlive || !actor.Active || actor.thirdCannon.Stage != 1 || actor.Health != data.FixedSprites.Third.Cannon.Health[1] || w.Score != score+500 {
				t.Fatalf("source cannon contact: shield=%d want=%d alive=%v active=%v stage=%d health=%d score=%d want=%d", w.Equipment.Shield, wantShield, w.PlayerAlive, actor.Active, actor.thirdCannon.Stage, actor.Health, w.Score, score+500)
			}
			if actor.part.StrongHealth != reused || w.Pool.Slot(slot).Residue.StrongHealth != reused {
				t.Fatal("compound cannon contact lost its inherited strength")
			}
		})
	}
}
