package engine

import (
	"fmt"
	"testing"
	"xenon2/internal/visualassets"
)

func TestWorldForecastReuseAcrossAllOriginalLevelsOptional(t *testing.T) {
	var forecast WorldForecast
	for cycle := 0; cycle < 2; cycle++ {
		for level := 1; level <= 5; level++ {
			for _, arena := range []bool{false, true} {
				t.Run(fmt.Sprintf("cycle%d/level%d/arena%v", cycle, level, arena), func(t *testing.T) {
					live := forecastOriginalScene(t, level, arena)
					before := forecastDigest(live)
					if err := forecast.Load(live); err != nil {
						t.Fatal(err)
					}
					if got := forecastDigest(forecast.State()); got != before {
						forecastFieldDifferences(t, forecast.State(), live)
						t.Fatal("reload retained another level's mutable state")
					}
					if forecast.State().Coverage != nil && &forecast.State().Coverage.Map[0] != &forecast.State().Level.Terrain.Map[0] {
						t.Fatal("reload separated mutable terrain and coverage")
					}
					for pass := 0; pass < 6; pass++ {
						for range 3 {
							forecast.AdvancePALTick()
						}
						result, err := forecast.Advance(Input{Fire: pass%2 == 0, Motion: demoDirections[pass%len(demoDirections)]})
						if err != nil {
							t.Fatal(err)
						}
						if result.Boundary != ForecastRunning {
							break
						}
					}
					if forecastDigest(live) != before {
						t.Fatal("reused forecast changed its source")
					}
					if err := forecast.Load(live); err != nil {
						t.Fatal(err)
					}
					if forecastDigest(forecast.State()) != before {
						t.Fatal("reloading the frozen source retained predicted births or terrain changes")
					}
				})
			}
		}
	}
}

func TestWorldForecastReuseCopiesOtherForecastAndOverlappingWrapperOptional(t *testing.T) {
	live := forecastOriginalScene(t, 3, true)
	original := forecastDigest(live)
	var first, second WorldForecast
	if err := first.Load(live); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		first.AdvancePALTick()
	}
	if _, err := first.Advance(Input{Fire: true, Motion: MotionInput{Right: true}}); err != nil {
		t.Fatal(err)
	}
	firstBefore := forecastDigest(first.State())
	if err := second.Load(first.State()); err != nil {
		t.Fatal(err)
	}
	if forecastDigest(second.State()) != firstBefore {
		t.Fatal("loading another forecast did not preserve its complete state")
	}
	if first.State() == second.State() || first.State().Pool != nil && first.State().Pool == second.State().Pool || first.State().Coverage != nil && first.State().Coverage == second.State().Coverage || first.State().Weapons != nil && first.State().Weapons == second.State().Weapons {
		t.Fatal("separate forecasts shared mutable containers")
	}
	for range 3 {
		second.AdvancePALTick()
	}
	if _, err := second.Advance(Input{Fire: true, Motion: MotionInput{Left: true}}); err != nil {
		t.Fatal(err)
	}
	if forecastDigest(first.State()) != firstBefore || forecastDigest(live) != original {
		t.Fatal("forecast-to-forecast copy leaked mutable callbacks")
	}
	state, before := second.State(), forecastDigest(second.State())
	if err := second.Load(state); err != nil || second.State() != state || forecastDigest(second.State()) != before {
		t.Fatalf("self-load changed the isolated prediction: %v", err)
	}
	wrapper := *second.State()
	if err := second.Load(&wrapper); err != nil || forecastDigest(second.State()) != before {
		t.Fatalf("overlapping shallow wrapper was corrupted during arena reset: %v", err)
	}
	if err := second.Load(nil); err == nil || second.State() != nil {
		t.Fatal("nil load retained a visible state handle")
	}
	if err := second.Load(live); err != nil || forecastDigest(second.State()) != original {
		t.Fatalf("reload after nil invalidation retained old prediction state: %v", err)
	}
}

func TestWorldForecastReuseRebindsRetainedWeaponCallbacksOptional(t *testing.T) {
	first := forecastOriginalScene(t, 1, false)
	second := forecastOriginalScene(t, 5, true)
	var forecast WorldForecast
	if err := forecast.Load(first); err != nil {
		t.Fatal(err)
	}
	if err := forecast.Load(second); err != nil {
		t.Fatal(err)
	}
	firstBefore, secondBefore := forecastDigest(first), forecastDigest(second)
	state := forecast.State()
	expectedID := state.nextActorID + 1
	if state.Weapons.newID() != expectedID || state.nextActorID != expectedID {
		t.Fatal("retained weapon ID callback belongs to a previous world")
	}
	state.Weapons.context.NextRandom()
	state.Weapons.context.SoundVoice(2, "synthesized-effect-05")
	if state.SoundRequests[2] != "synthesized-effect-05" || forecastDigest(first) != firstBefore || forecastDigest(second) != secondBefore {
		t.Fatal("reused weapon callbacks escaped the current prediction")
	}
}

func BenchmarkWorldForecastFrozenSourceReload(b *testing.B) {
	for level := 1; level <= 5; level++ {
		b.Run(fmt.Sprintf("level%d", level), func(b *testing.B) {
			source := forecastOriginalScene(b, level, true)
			var forecast WorldForecast
			if err := forecast.Load(source); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				for range 100 {
					if err := forecast.Load(source); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

func TestWorldForecastReuseReleasesShrinkingEntityGraph(t *testing.T) {
	large := testWorld(t)
	chain := &ThirdChainState{}
	for index := 0; index < 40; index++ {
		actor := &WorldActor{Active: true, ActorList: "moving", X: float64(index), part: &visualassets.ActorPart{ResourceTag: 200}, thirdChain: chain, Extras: []WorldSpriteAttachment{{Sprite: "first"}, {Sprite: "second"}}, TileOverlays: []WorldTileOverlay{{Patch: visualassets.TilePatch{Columns: 2, Rows: 1, Tiles: []uint16{1, 2}}}}}
		if err := large.bindWorldActor(actor); err != nil {
			t.Fatal(err)
		}
		large.Actors = append(large.Actors, actor)
	}
	for _, actor := range large.Actors {
		actor.leader = large.Actors[0]
		actor.fifthTileGroup = large.Actors
		actor.thirdChainMembers[0] = large.Actors[0]
	}
	var forecast WorldForecast
	if err := forecast.Load(large); err != nil {
		t.Fatal(err)
	}
	if len(forecast.State().Actors) != 40 || forecast.State().Actors[39].leader != forecast.State().Actors[0] || forecast.State().Actors[39].thirdChain != forecast.State().Actors[0].thirdChain {
		t.Fatal("growing graph lost source aliases")
	}
	small := testWorld(t)
	smallActor := &WorldActor{Active: true, ActorList: "moving", X: 250, part: &visualassets.ActorPart{ResourceTag: 200}}
	if err := small.bindWorldActor(smallActor); err != nil {
		t.Fatal(err)
	}
	small.Actors = []*WorldActor{smallActor}
	if err := forecast.Load(small); err != nil {
		t.Fatal(err)
	}
	state := forecast.State()
	if len(state.Actors) != 1 || state.Actors[0].X != 250 || state.Actors[0].thirdChain != nil || state.Actors[0].leader != nil || len(state.Actors[0].Extras) != 0 || len(state.Actors[0].TileOverlays) != 0 || len(state.Actors[0].fifthTileGroup) != 0 {
		t.Fatal("smaller source retained prior controller or attachment state")
	}
	for _, actor := range forecast.storage.actors[len(state.Actors):cap(forecast.storage.actors)] {
		if actor != nil {
			t.Fatal("shrunk pointer-list backing retained a prior entity")
		}
	}
	if err := forecast.Load(large); err != nil {
		t.Fatal(err)
	}
	state = forecast.State()
	state.Actors[0].TileOverlays[0].Patch.Tiles[0] = 9
	state.Actors[0].Extras[0].Sprite = "predicted"
	state.Actors[0].thirdChain.Phase = 14
	if large.Actors[0].TileOverlays[0].Patch.Tiles[0] != 1 || large.Actors[0].Extras[0].Sprite != "first" || large.Actors[0].thirdChain.Phase != 0 {
		t.Fatal("regrown graph reused mutable live-source buffers")
	}
	spawning := testWorld(t)
	if err := forecast.Load(spawning); err != nil {
		t.Fatal(err)
	}
	if _, err := forecast.Advance(Input{}); err != nil {
		t.Fatal(err)
	}
	if len(forecast.State().Actors) == 0 {
		t.Fatal("source encounter did not create forecast entities")
	}
	if err := forecast.Load(small); err != nil {
		t.Fatal(err)
	}
	if forecastDigest(forecast.State()) != forecastDigest(small) {
		t.Fatal("reload retained entities born during predicted Step")
	}
}
