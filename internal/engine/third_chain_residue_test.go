package engine

import (
	"strconv"
	"testing"
	"xenon2/internal/visualassets"
)

func TestThirdChainEveryPartDamageMatchesOriginalCallbackOptional(t *testing.T) {
	data := originalWorldData(t, 3)
	rows := 0
	nativeCombatRows(t, "third-chain-damage.csv", func(v []int64) {
		w, err := NewWorld(data)
		if err != nil {
			t.Fatal(err)
		}
		w.ScrollY, w.ScrollDelta, w.Player.Y = 2700, 0, 120
		w.spawnThirdChain(visualassets.FixedEncounter{EnemyKind: 1, Variant: int(v[0]), X: 128, Y: 2850})
		var head *WorldActor
		for _, actor := range w.Actors {
			if actor.thirdChainPart == 1 {
				head = actor
			}
		}
		if head == nil {
			t.Fatal("original chain constructor did not create its head")
		}
		relative := func(slot int) int {
			if slot == NoActorSlot {
				return -1
			}
			return slot - head.Binding.Slot + 1
		}
		headSlot := w.Pool.Slot(head.Binding.Slot)
		hitSlot := w.Pool.Slot(head.thirdChainMembers[v[1]].Binding.Slot)
		if relative(headSlot.Residue.FollowingSlot) != int(v[9]) || relative(hitSlot.Residue.FollowingSlot) != int(v[11]) || headSlot.Linked != (v[10] != 0) || headSlot.SkipDeathEffect != (v[10]&128 != 0) || hitSlot.Linked != (v[12] != 0) || hitSlot.SkipDeathEffect != (v[12]&128 != 0) {
			t.Fatalf("chain constructor lost original member links or signed flags: %v", v)
		}
		for w.Pool.FreeFirst() != NoActorSlot && v[13] != 0 {
			if v[13] == 1 {
				w.spawnEnemyShot(80, 48, EnemyShot{Speed: 4})
			} else if _, err := w.reserveWorldActor(200, ActorPoolMoving, true); err != nil {
				t.Fatal(err)
			}
		}
		w.damageActor(head.thirdChainMembers[v[1]], uint16(v[2]))
		live, effects := 0, 0
		for _, actor := range head.thirdChainMembers {
			if actor.Active {
				live++
			}
		}
		for slot := w.Pool.First(ActorPoolProjectile); slot != NoActorSlot; slot = w.Pool.Next(slot) {
			actor := w.poolActors[slot]
			if actor != nil && actor.ID == w.Pool.Slot(slot).EntityID && actor.ActorList == "transient" && actor.part != nil && actor.part.MotionMode == "finite-effect" {
				effects++
			}
		}
		if head.Health != int(v[4]) || w.Score != int(v[5]) || w.Pool.Slot(head.Binding.Slot).ResourceTag != int16(v[6]) || live != int(v[7]) || effects != int(v[8]) {
			t.Fatalf("chain variant%d part%d damage%d: HP%d score%d tag%d live%d effects%d; original%v", v[0], v[1], v[2], head.Health, w.Score, w.Pool.Slot(head.Binding.Slot).ResourceTag, live, effects, v)
		}
		rows++
	})
	if rows != 96 {
		t.Fatalf("incomplete original chain damage coverage: %d", rows)
	}
}

func TestThirdChainDeathRetiresAllLinkedMembersBeforeCountingOptional(t *testing.T) {
	for variant := range 2 {
		w, err := NewWorld(originalWorldData(t, 3))
		if err != nil {
			t.Fatal(err)
		}
		w.spawnThirdChain(visualassets.FixedEncounter{EnemyKind: 1, Variant: variant, X: 128, Y: 2850})
		var head *WorldActor
		for _, actor := range w.Actors {
			if actor.thirdChainPart == 1 {
				head = actor
			}
		}
		if head == nil {
			t.Fatal("original chain constructor did not create its head")
		}
		w.damageActor(head, 127)
		for _, actor := range head.thirdChainMembers {
			if actor.Active || w.Pool.Slot(actor.Binding.Slot).ResourceTag != 4 || w.Pool.Slot(actor.Binding.Slot).Linked {
				t.Fatalf("variant%d left a live linked chain member after head death", variant)
			}
		}
		// The source group skipper assumes that a dead head leaves no live
		// linked body behind. It must not revisit an orphan after this callback.
		w.countSourceMovingActors()
	}
}

// Whole-pixel chains retain coordinate fractions while publishing their native
// phase/speed/base-X/spacing words at 0x28/0x2a/0x2e/0x30.
func TestOriginalThirdChainPublishesBodyStateAndPreservesFractionsOptional(t *testing.T) {
	for variant := range 2 {
		t.Run(strconv.Itoa(variant), func(t *testing.T) {
			w, err := NewWorld(originalWorldData(t, 3))
			if err != nil {
				t.Fatal(err)
			}
			w.ScrollY, w.ScrollDelta, w.Player.Y = 2700, 1, 160
			w.ScrollDelta = 0
			for range 10 {
				w.spawnEnemyShot(80, 30, EnemyShot{Direction: 1, Speed: 6})
			}
			for range 40 {
				if err := w.advancePooledProjectiles(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			retained := w.Pool.Slot(w.Pool.FreeFirst()).Residue
			if retained.XFraction == 0 || retained.YFraction == 0 {
				t.Fatal("expired original bullets did not establish retained fractions")
			}
			w.ScrollDelta = 1
			w.spawnThirdChain(visualassets.FixedEncounter{EnemyKind: 1, Variant: variant, X: 128, Y: 2850})
			var head *WorldActor
			for _, actor := range w.Actors {
				if actor.thirdChainPart == 1 {
					head = actor
				}
			}
			if head == nil {
				t.Fatal("original chain head missing")
			}
			baseX := int16(120 + variant*32)
			check := func() {
				t.Helper()
				for index, actor := range head.thirdChainMembers {
					r := w.Pool.Slot(actor.Binding.Slot).Residue
					phase, speed, spacing := 0, 0, head.thirdChain.Parts[index].Spacing
					if index == 0 {
						phase, speed, spacing = head.thirdChain.Phase, head.thirdChain.Speed, head.thirdChain.AmplitudeOrCooldown
					}
					if r.XFraction != retained.XFraction || r.YFraction != retained.YFraction || r.Counter != int16(phase) || r.Direction != int16(speed) || r.VerticalVelocity != baseX || r.VerticalFraction != uint16(int16(spacing)) || r.StrongHealth || r.WaveBonusToken != 0 || r.EmitterClock != retained.EmitterClock {
						t.Fatalf("part%d retained/source state differs: %+v phase%d speed%d spacing%d", index, r, phase, speed, spacing)
					}
				}
			}
			check()
			activePhase := false
			for pass := 0; pass < 60; pass++ {
				w.advanceThirdChain(head)
				activePhase = activePhase || head.thirdChain.Phase != 0
				check()
			}
			if !activePhase {
				t.Fatal("original source proximity did not activate chain extension")
			}
			slot := head.Binding.Slot
			for pass := 0; head.Active && pass < 120; pass++ {
				if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
					t.Fatal(err)
				}
			}
			if head.Active {
				t.Fatal("extended chain did not reach its native expiry boundary")
			}
			if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
				t.Fatal(err)
			}
			var bullet *WorldProjectile
			for count := 0; count < 12; count++ {
				w.spawnEnemyShot(100, 90, EnemyShot{Direction: 1, Speed: 6})
				if w.Projectiles[0].Binding.Slot == slot {
					bullet = w.Projectiles[0]
					break
				}
			}
			if bullet == nil {
				t.Fatal("ordinary allocator did not reuse the expired chain head")
			}
			if err := w.advancePooledProjectiles(Input{}); err != nil {
				t.Fatal(err)
			}
			wantX := (int32(100)<<16 | int32(retained.XFraction)) + 11585*6*4
			wantY := (int32(90)<<16 | int32(retained.YFraction)) - 11585*6*4 + int32(w.ScrollDelta)<<16
			if bullet.Motion.X != wantX || bullet.Motion.Y != wantY {
				t.Fatal("expired chain body changed the next native fractional bullet trajectory")
			}
		})
	}
}
