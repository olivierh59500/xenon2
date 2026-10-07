package engine

import (
	"fmt"
	"testing"
)

func forecastIsolationDigest(w *World) string {
	return forecastDigest(w) + forecastDigest(w.weaponTargets) + forecastDigest(w.weaponTargetActors) + forecastDigest(w.Weapons.context.Equipment) + forecastDigest(w.Weapons.context.Targets)
}

func forecastIsolationActor(t *testing.T, w *World, id int) *WorldActor {
	t.Helper()
	for _, actor := range w.Actors {
		if actor.ID == id {
			return actor
		}
	}
	for _, actor := range w.poolActors {
		if actor != nil && actor.ID == id {
			return actor
		}
	}
	t.Fatalf("forecast lost source actor %d", id)
	return nil
}

func TestWorldForecastOwnsMutableSourceStateAcrossFiveLevelsOptional(t *testing.T) {
	for level := 1; level <= 5; level++ {
		t.Run(fmt.Sprintf("level%d", level), func(t *testing.T) {
			live := forecastOriginalScene(t, level, true)
			if level == 1 {
				if err := live.spawnFirstMiddleStream(0, 0); err != nil {
					t.Fatal(err)
				}
				live.advanceFirstGuardian()
			}
			if level == 2 {
				live.advanceSecondGuardian()
			}
			if level == 5 {
				live.advanceFifthGuardian(false)
				live.fifthMiddleBodyRender(live.fifthMiddleActors[0])
			}
			// Original equipment initializers populate actual mounts, projectiles
			// and their world-bound callbacks; this is not a campaign fixture.
			live.Equipment.ApplyItem(ItemCannon)
			live.Equipment.ApplyItem(ItemLaser)
			if err := live.Weapons.AdvanceEquipment(live.weaponContext(Input{Fire: true}, true)); err != nil {
				t.Fatal(err)
			}
			live.spawnEnemyShot(16, 16, EnemyShot{Direction: 2, Speed: 6})
			before := forecastIsolationDigest(live)
			checkLive := func(action string) {
				t.Helper()
				if forecastIsolationDigest(live) != before {
					t.Fatalf("%s changed the live world", action)
				}
			}
			var forecast WorldForecast
			if err := forecast.Load(live); err != nil {
				t.Fatal(err)
			}
			checkLive("loading")
			for range 3 {
				forecast.AdvancePALTick()
			}
			if _, err := forecast.Advance(Input{Fire: true}); err != nil {
				t.Fatal(err)
			}
			checkLive("forecast-only firing and actor callbacks")
			if err := forecast.Load(live); err != nil {
				t.Fatal(err)
			}
			state := forecast.State()
			if state == live || state.Pool == live.Pool || state.Weapons == live.Weapons {
				t.Fatal("forecast reused a live owner")
			}
			chains := make(map[*ThirdChainState]*ThirdChainState)
			markers := make(map[*[2]*WorldActor]*[2]*WorldActor)
			for _, source := range live.Actors {
				actor := forecastIsolationActor(t, state, source.ID)
				if actor == source || actor.part == source.part || state.poolActors[actor.Binding.Slot] != actor {
					t.Fatal("actor, mutable descriptor or physical index was not isolated")
				}
				if source.leader != nil && actor.leader != forecastIsolationActor(t, state, source.leader.ID) {
					t.Fatal("forecast owner link escaped its cloned actor tree")
				}
				if source.thirdChain != nil {
					if actor.thirdChain == source.thirdChain || chains[source.thirdChain] != nil && chains[source.thirdChain] != actor.thirdChain {
						t.Fatal("shared chain controller was leaked or duplicated")
					}
					chains[source.thirdChain] = actor.thirdChain
				}
				if source.firstMiddleMarkers != nil {
					if actor.firstMiddleMarkers == source.firstMiddleMarkers || markers[source.firstMiddleMarkers] != nil && markers[source.firstMiddleMarkers] != actor.firstMiddleMarkers {
						t.Fatal("shared marker array was leaked or duplicated")
					}
					markers[source.firstMiddleMarkers] = actor.firstMiddleMarkers
				}
				actor.Health--
				actor.part.DamageMode = "forecast-isolation"
				if actor.Patch != nil && len(actor.Patch.Tiles) != 0 {
					actor.Patch.Tiles[0] ^= 1
				}
			}
			checkLive("actor descriptor, patch and tree mutation")
			for _, chain := range chains {
				chain.Parts[0].X++
			}
			for _, marker := range markers {
				marker[0].Health--
			}
			state.Pool.Slot(0).Residue.Counter++
			state.Level.Terrain.Map[0] ^= 1
			state.RenderTerrainMap[0] ^= 1
			state.ActorRenderTerrainMap[0] ^= 1
			switch level {
			case 1:
				state.FirstMiddle.GateCounters[0]++
				state.firstGuardianBody.Tiles[0] ^= 1
			case 2:
				state.SecondGuardian.BodyWorldY++
				state.secondNodes[0].secondNode.Phase++
				state.secondGuardianBody.Tiles[0] ^= 1
			case 3:
				state.ThirdFinal.Health--
			case 4:
				state.FourthMiddle.Parts[0].Counter++
			case 5:
				state.FifthMiddle.Parts[0].Health--
				state.fifthMiddleActors[0].fifthBodyRender.pass++
			}
			checkLive("controller, shared array, tile map and physical residue mutation")
			context := state.Weapons.context
			if state.Weapons.newID == nil || context.NextID == nil || context.NextRandom == nil || context.Equipment != &state.Equipment {
				t.Fatal("forecast weapon callbacks were not rebound to their owner")
			}
			id := state.nextActorID
			if state.Weapons.newID() != id+1 || state.nextActorID != id+1 || context.NextID() != id+2 || state.nextActorID != id+2 {
				t.Fatal("forecast ID callbacks did not update the forecast counter")
			}
			context.NextRandom()
			context.Equipment.Shield--
			if len(context.Targets) != 0 {
				context.Targets[0].ResourceTag++
			}
			checkLive("weapon ID, random, equipment and target callbacks")
		})
	}
}
