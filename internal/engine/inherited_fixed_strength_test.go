package engine

import (
	"fmt"
	"testing"

	"xenon2/internal/visualassets"
)

func reclaimOriginalStrongFixedSlots(t *testing.T, w *World, record visualassets.FixedEncounter, count int) map[int]bool {
	t.Helper()
	birthY := 392
	if w.Level.Number == 5 {
		birthY = w.fixedKinds[record.EnemyKind].MotionParameters["clip_bottom"]
	}
	w.ScrollY, w.ScrollDelta = record.Y-8-birthY, 0
	if w.Level.Number == 3 {
		w.MaximumScrollY = record.Y - 8 - 209
	}
	for range count {
		w.spawnFixed(record)
	}
	slots := make(map[int]bool, count)
	for _, actor := range w.Actors {
		if actor.Active && actor.fixedKind != nil && actor.fixedKind.Kind == record.EnemyKind {
			if !actor.part.StrongHealth || !actor.Binding.Residue.StrongHealth {
				t.Fatal("source fixed sprite must initialize a strong physical slot")
			}
			slots[actor.Binding.Slot] = true
		}
	}
	// The source's off-screen boundary retires these actual sprite families
	// without a death effect, then the following moving traversal frees them.
	for range 2 {
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	for slot := range slots {
		if w.Pool.Slot(slot).allocated || !w.Pool.Slot(slot).Residue.StrongHealth {
			t.Fatal("real fixed sprite expiration must reclaim its strong physical slot")
		}
	}
	if len(slots) != count || !slots[w.Pool.FreeFirst()] {
		t.Fatal("source expiration must reclaim distinct slots at the free-list head")
	}
	return slots
}

func TestCommonFixedConstructorsRetainSourceStrengthOptional(t *testing.T) {
	for _, family := range []struct {
		level, kind, part, tag, score int
		name                          string
	}{
		{3, 4, -1, 240, 100, "third-tile"},
		{5, 2, -1, 264, 100, "fifth-small-tile"},
		{5, 1, 0, 296, 200, "fifth-left-post"},
		{5, 1, 1, 304, 0, "fifth-barrier"},
		{5, 1, 2, 300, 200, "fifth-right-post"},
		{5, 3, -1, 244, 400, "fifth-aiming"},
		{5, 4, -1, 248, 500, "fifth-radial"},
		{5, 9, -1, 244, 400, "fifth-persistent"},
	} {
		for _, reused := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reused=%v", family.name, reused), func(t *testing.T) {
				data := originalWorldData(t, family.level)
				precursorKind := 2
				if family.level == 5 {
					precursorKind = 8
				}
				var record, precursor visualassets.FixedEncounter
				for _, candidate := range data.Encounters.Fixed {
					if candidate.EnemyKind == family.kind && record.EnemyKind == 0 {
						record = candidate
					}
					if candidate.EnemyKind == precursorKind && precursor.EnemyKind == 0 {
						precursor = candidate
					}
				}
				if record.EnemyKind != family.kind || precursor.EnemyKind != precursorKind {
					t.Fatal("original fixed encounter records are required")
				}
				w, err := NewWorld(data)
				if err != nil {
					t.Fatal(err)
				}
				w.Ready, w.MaterializationFrames, w.InvulnerableFrames = false, 0, 0
				w.Level.Encounters = &visualassets.Encounters{}
				w.MaximumScrollY, w.VisitedScrollY = 4607, 4607
				var slots map[int]bool
				if reused {
					count := 1
					if family.kind == 1 {
						count = 3
					}
					slots = reclaimOriginalStrongFixedSlots(t, w, precursor, count)
				}
				w.ScrollY, w.ScrollDelta = record.Y-8-100, 0
				w.MaximumScrollY, w.VisitedScrollY = 4607, 4607
				w.Player.X, w.Player.Y = 160, 176
				w.spawnFixed(record)
				var actor *WorldActor
				for _, candidate := range w.Actors {
					if family.part < 0 && candidate.fixedTileArt != nil && candidate.fixedTileArt.Kind == family.kind ||
						family.part >= 0 && candidate.fifthTile != nil && candidate.fifthTile.Part == family.part {
						actor = candidate
						break
					}
				}
				if actor == nil || actor.part.ResourceTag != family.tag || actor.Health != actor.fixedTileArt.Health {
					t.Fatal("source tile constructor must retain its own tag and health")
				}
				slot := actor.Binding.Slot
				if reused && !slots[slot] || actor.Binding.Residue.StrongHealth != reused || w.Pool.Slot(slot).Residue.StrongHealth != reused {
					t.Fatal("source tile constructor must preserve the selected physical slot's strength")
				}
				if err := w.Step(Input{}); err != nil {
					t.Fatal(err)
				}
				if !actor.Active || actor.Collision.Right < actor.Collision.Left {
					t.Fatal("source tile update must publish its real collision rectangle")
				}
				w.Player.X, w.Player.Y = (actor.Collision.Left+actor.Collision.Right)/2, (actor.Collision.Top+actor.Collision.Bottom)/2
				if family.part == 0 {
					w.Player.X = actor.Collision.Left - 5
				} else if family.part == 2 {
					w.Player.X = actor.Collision.Right + 8
				}
				w.Equipment.Shield = 39
				score, health := w.Score, actor.Health
				if err := w.Step(Input{}); err != nil {
					t.Fatal(err)
				}
				wantShield := 31
				if reused {
					wantShield = 23
				}
				if w.Equipment.Shield != wantShield || !w.PlayerAlive || w.Score != score+family.score {
					t.Fatalf("source tile contact: shield=%d want=%d alive=%v score=%d want=%d", w.Equipment.Shield, wantShield, w.PlayerAlive, w.Score, score+family.score)
				}
				if family.part == 1 {
					if !actor.Active || actor.Health != health {
						t.Fatal("barrier contact must retain its harmless damage callback")
					}
				} else if actor.Active || actor.Health != int(uint16(health)-127) {
					t.Fatal("source contact must destroy the damageable tile")
				}
				if family.kind == 9 && !w.fifthDestroyedTurrets[record.State2] {
					t.Fatal("persistent turret contact must retain its source selector callback")
				}
				if actor.part.StrongHealth != reused || w.Pool.Slot(slot).Residue.StrongHealth != reused {
					t.Fatal("source tile contact lost its retained strength")
				}
			})
		}
	}
}
