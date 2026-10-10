package engine

import "testing"

func TestThirdScenerySurvivesCheckpointAndForecastRestorationOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	actor := w.thirdSceneryActor
	if actor == nil || !actor.Active {
		t.Fatal("original third-level scenery controller is missing")
	}
	binding := actor.Binding
	w.RestartCheckpoint()
	if w.thirdSceneryActor != actor || !actor.Active || actor.Binding != binding || w.Pool.Slot(binding.Slot).ResourceTag != 80 {
		t.Fatal("checkpoint cleanup released the persistent source scenery")
	}
	count := 0
	for _, member := range w.Actors {
		if member == actor {
			count++
		}
	}
	if count != 1 {
		t.Fatal("checkpoint lost or duplicated its scenery callback")
	}
	before := forecastIsolationDigest(w)
	var forecast WorldForecast
	if err := forecast.Load(w); err != nil {
		t.Fatal(err)
	}
	copy := forecast.State()
	if copy.thirdSceneryActor == actor || copy.poolActors[binding.Slot] != copy.thirdSceneryActor {
		t.Fatal("forecast scenery reference escaped its owned actor graph")
	}
	copy.RestartCheckpoint()
	copy.thirdSceneryActor.Patch.Tiles[0] ^= 1
	if forecastIsolationDigest(w) != before {
		t.Fatal("forecast restoration or scenery mutation changed the live game")
	}
}
