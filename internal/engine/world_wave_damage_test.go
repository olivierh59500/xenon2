package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func waveDamageReclaimFixture(t *testing.T, reward bool) (*World, *WorldActor) {
	t.Helper()
	w := testWorld(t)
	w.Level.Actors.Atlas.Sprites = []visualassets.SpriteRegion{{Name: "enemy", Width: 24, Height: 18, AnchorX: 4, AnchorY: 6}}
	for _, name := range []string{"explosion-large", "cash-large"} {
		w.commonAnimations[name] = visualassets.NamedActorAnimation{Ending: "hold", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: name}}}}
	}
	a := &WorldActor{Active: true, ActorList: "moving", Atlas: "moving", Sprite: "enemy", X: 70, Y: 90, Health: 3, Score: 100,
		part: &visualassets.ActorPart{ResourceTag: 200, MotionMode: "path", DamageMode: "individual", StrongHealth: true}}
	if err := w.bindWorldActor(a); err != nil {
		t.Fatal(err)
	}
	if reward {
		w.WaveBonuses.BeginPass(0)
		a.WaveToken = w.WaveBonuses.Register(true, 1)
	}
	w.initializeWaveActorResidue(a, a)
	w.Actors = []*WorldActor{a}
	return w, a
}

func fillWaveDamageMovingTail(t *testing.T, w *World) {
	t.Helper()
	for w.Pool.FreeFirst() != NoActorSlot {
		if _, err := w.reserveWorldActor(200, ActorPoolMoving, true); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWaveNonlethalDamagePublishesHealthBeforeTheNextCallback(t *testing.T) {
	w, actor := waveDamageReclaimFixture(t, false)
	w.damageActor(actor, 1)
	if actor.Health != 2 || !actor.Active || !actor.Flash || w.Pool.Slot(actor.Binding.Slot).Residue.Health != 2 || w.Pool.Slot(actor.Binding.Slot).ResourceTag != 200 || w.Score != 0 || len(w.Actors) != 1 {
		t.Fatal("nonlethal source subtraction was deferred or ran a death callback")
	}
}

func TestWaveDeathRetiresItsReclaimedExplosionOrCash(t *testing.T) {
	for _, reward := range []bool{false, true} {
		t.Run(map[bool]string{false: "explosion", true: "cash"}[reward], func(t *testing.T) {
			w, actor := waveDamageReclaimFixture(t, reward)
			slot, identity := actor.Binding.Slot, actor.ID
			fillWaveDamageMovingTail(t, w)
			w.damageActor(actor, 3)
			current := w.Pool.Slot(slot)
			if actor.Active || current.EntityID == identity || current.ResourceTag != 4 || current.list != ActorPoolProjectile || w.Score != 100 || w.Money != 0 {
				t.Fatalf("death did not follow the replaced physical entry: %+v score%d", current, w.Score)
			}
			if reward {
				if len(w.Collectibles) != 1 || w.Collectibles[0].Active || w.Collectibles[0].X != 78 || w.Collectibles[0].Y != 92 || w.Collectibles[0].Binding.Slot != slot {
					t.Fatal("cash ignored the reused explosion center or survived the final dead write")
				}
			} else if w.poolActors[slot] == nil || w.poolActors[slot].Active || w.poolActors[slot].X != 78 || w.poolActors[slot].Y != 92 {
				t.Fatal("the replacement explosion survived its parent's final dead write")
			}
			if err := w.advancePooledProjectiles(Input{}); err != nil {
				t.Fatal(err)
			}
			if w.Pool.Slot(slot).allocated {
				t.Fatal("next projectile traversal did not release the reclaimed dead entry")
			}
		})
	}
}

func TestCarrierKeepsItsLiveTagUntilRewardAllocation(t *testing.T) {
	w := testWorld(t)
	w.commonAnimations["pickup-14"] = visualassets.NamedActorAnimation{ResourceTag: 104, Ending: "hold", Animation: visualassets.ActorAnimation{Frames: []visualassets.AnimationFrame{{Sprite: "gift"}}}}
	actor := &WorldActor{Active: true, ActorList: "moving", X: 96, Y: 88, CarriedReward: 14,
		part: &visualassets.ActorPart{ResourceTag: 100, MotionMode: "path", DamageMode: "drop-equipment"}}
	if err := w.bindWorldActor(actor); err != nil {
		t.Fatal(err)
	}
	w.initializeWaveActorResidue(actor, actor)
	actor.Binding.Residue.VerticalFraction = 14
	w.storeWorldResidue(actor.Binding)
	w.Actors = []*WorldActor{actor}
	slot, equipment := actor.Binding.Slot, w.Equipment
	fillWaveDamageMovingTail(t, w)
	w.damageActor(actor, 1)
	if actor.Active || w.Pool.Slot(slot).ResourceTag != 4 || w.Pool.Slot(slot).list != ActorPoolMoving || len(w.Collectibles) != 1 || !w.Collectibles[0].Active || w.Collectibles[0].Binding.Slot == slot || w.Collectibles[0].Reward != 14 || w.Equipment != equipment || w.Score != 0 || w.Money != 0 {
		t.Fatal("early carrier retirement changed reclamation, lost its gift or granted equipment")
	}
}
