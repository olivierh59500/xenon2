package engine

import (
	"testing"

	"xenon2/internal/visualassets"
)

func TestWorldPoolSharesCapacityAndEvictsWithoutReward(t *testing.T) {
	w := testWorld(t)
	w.Level.Rules = &visualassets.LevelRules{DefaultEnemyShot: "enemy-shot"}
	for range ActorPoolCapacity - 4 {
		actor := &WorldActor{Active: true, ActorList: "moving", Health: 1, Score: 100,
			part: &visualassets.ActorPart{ResourceTag: 200, DamageMode: "individual"}}
		if err := w.bindWorldActor(actor); err != nil {
			t.Fatal(err)
		}
		w.Actors = append([]*WorldActor{actor}, w.Actors...)
	}
	victim := w.Actors[0]
	w.spawnEnemyShot(90, 100, EnemyShot{Speed: 4})
	if victim.Active || w.Score != 0 || w.Money != 0 || len(w.Collectibles) != 0 {
		t.Fatal("capacity eviction must discard the moving head without a death reward")
	}
	if len(w.Projectiles) != 1 || w.Projectiles[0].Binding.Slot != victim.Binding.Slot || w.Projectiles[0].ID == victim.ID {
		t.Fatal("new projectile must reuse the evicted physical slot with a new identity")
	}
	if w.Pool.FreeFirst() != NoActorSlot {
		t.Fatal("shared pool exceeded its fixed capacity")
	}
}

func TestWorldPoolDeathWaitsForOwningPhase(t *testing.T) {
	w := testWorld(t)
	actor := &WorldActor{Active: true, ActorList: "moving", Health: 1,
		part: &visualassets.ActorPart{ResourceTag: 200, DamageMode: "individual"}}
	if err := w.bindWorldActor(actor); err != nil {
		t.Fatal(err)
	}
	w.Actors = []*WorldActor{actor}
	w.damageActor(actor, 1)
	if actor.Active || w.Pool.Slot(actor.Binding.Slot).ResourceTag != 4 {
		t.Fatal("damage must mark the original slot dead without releasing it")
	}
	w.compactActors()
	w.releaseDeadPoolEntries(ActorPoolProjectile)
	if !w.Pool.Slot(actor.Binding.Slot).allocated {
		t.Fatal("projectile phase released a moving entry")
	}
	w.releaseDeadPoolEntries(ActorPoolMoving)
	if w.Pool.Slot(actor.Binding.Slot).allocated || w.Pool.FreeFirst() != actor.Binding.Slot {
		t.Fatal("next moving visit must release the dead entry")
	}
}

func TestWorldPoolAllocatesShadowEntriesBeforeWeaponMount(t *testing.T) {
	w := testWorld(t)
	common := &visualassets.SpriteAtlas{}
	var err error
	w.Weapons, err = NewWeaponRuntime(common)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.initializeWorldPool(); err != nil {
		t.Fatal(err)
	}
	for i, binding := range w.poolShadows {
		if binding.Slot != i || w.Pool.Slot(i).ResourceTag != 196 {
			t.Fatal("four source shadow entries must reserve the first slots")
		}
	}
	if w.Weapons.mounts[0].Binding.Slot != 4 || w.Pool.Slot(4).ResourceTag != 160 || w.Pool.First(ActorPoolPlayer) != 0 {
		t.Fatal("basic weapon follows shadows; ship has dedicated state outside this pool")
	}
}

func TestWorldProjectileSelfRemovalRetainsSlotUntilNextTraversal(t *testing.T) {
	w := testWorld(t)
	w.Level.Rules = &visualassets.LevelRules{DefaultEnemyShot: "enemy-shot"}
	w.spawnEnemyShot(0, 500, EnemyShot{Speed: 4})
	shot := w.Projectiles[0]
	if err := w.advancePooledProjectiles(Input{}); err != nil {
		t.Fatal(err)
	}
	if shot.Active || !w.Pool.Slot(shot.Binding.Slot).allocated || w.Pool.Slot(shot.Binding.Slot).ResourceTag != 4 {
		t.Fatal("self-removal must leave a dead entry until the next source traversal")
	}
	if err := w.advancePooledProjectiles(Input{}); err != nil {
		t.Fatal(err)
	}
	if w.Pool.Slot(shot.Binding.Slot).allocated {
		t.Fatal("next traversal did not release the dead projectile")
	}
}

func TestWorldCheckpointCleanupDoesNotLeakActorSlots(t *testing.T) {
	w := testWorld(t)
	w.Level.Rules = &visualassets.LevelRules{DefaultEnemyShot: "enemy-shot"}
	for range 20 {
		w.spawnEnemyShot(90, 100, EnemyShot{Speed: 4})
		w.RestartCheckpoint()
		if len(w.poolBindings) != 4 || w.poolError != nil {
			t.Fatal("checkpoint cleanup leaked a projectile or released a persistent shadow")
		}
	}
}
