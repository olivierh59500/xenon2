package engine

import (
	"reflect"
	"testing"

	"xenon2/internal/visualassets"
)

func TestDemoFifthColumnPredictionMatchesOriginalProjectilePhaseOptional(t *testing.T) {
	for _, speed := range []int{-16, 16} {
		for _, delta := range []int{-2, 0, 1, 2} {
			w := fifthResourceWorld(t)
			w.Player.X, w.Player.Y = 300, 176
			w.MaterializationFrames = 0
			w.updatePlayerCollision()
			w.ScrollDelta = delta
			w.spawnFifthColumn(FifthGuardianLaser{X: 150, Y: 40, Speed: speed})
			actor := w.Actors[len(w.Actors)-1]
			if !demoActorHazard(actor) || actor.ActorList != "transient" || actor.fifthColumn.Length != 0 {
				t.Fatal("newborn projectile-list column is absent from movement hazards")
			}
			before := forecastIsolationDigest(w)
			var views [18]demoActorView
			for index := range views {
				view, supported := demoActorPrediction(w, actor, index+1, w.ScrollY)
				if !supported {
					t.Fatal("source column became unsupported within the prediction horizon")
				}
				views[index] = view
			}
			if forecastIsolationDigest(w) != before {
				t.Fatal("column prediction changed the live pool, controller or random state")
			}
			for index, view := range views {
				if err := w.advancePooledProjectiles(Input{}); err != nil {
					t.Fatal(err)
				}
				if view.Active != actor.Active || view.X != int(actor.X) || view.Y != int(actor.Y) || actor.Active && view.Bounds != actor.Collision {
					t.Fatalf("speed%d delta%d pass%d predicted%+v actual xy%v/%v bounds%+v active%v", speed, delta, index+1, view, actor.X, actor.Y, actor.Collision, actor.Active)
				}
			}
			if actor.Active || demoActorHazard(actor) {
				t.Fatal("expired column remains a movement hazard")
			}
		}
	}
}

func TestPresentationMovementAvoidsGrowingFifthColumnOptional(t *testing.T) {
	w := fifthResourceWorld(t)
	w.Ready, w.MaterializationFrames = false, 0
	w.Player.X, w.Player.Y, w.Player.Inertia = 160, 104, 0
	w.Equipment.ApplyItem(ItemSpeedup)
	w.Equipment.ApplyItem(ItemSpeedup)
	w.Player.SpeedTier = w.Equipment.SpeedTier
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	w.ScrollDelta = 1
	// Isolate this source projectile's movement response from terrain and births.
	w.Coverage = nil
	w.Level.Encounters = &visualassets.Encounters{}
	w.updatePlayerCollision()
	w.spawnFifthColumn(FifthGuardianLaser{X: 150, Y: 40, Speed: 16})
	actor := w.Actors[len(w.Actors)-1]
	before := forecastIsolationDigest(w)
	risk, _ := presentationMotionScore(w, MotionInput{}, 160, 104)
	if risk <= 0 {
		t.Fatal("expert movement misses the column's growth and later independent travel")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("column avoidance changed the live game")
	}
	actor.Active = false
	if got, _ := presentationMotionScore(w, MotionInput{}, 160, 104); got != 0 {
		t.Fatalf("inactive column still penalizes movement: %v", got)
	}
	actor.Active = true
	var held, guarded WorldForecast
	if err := held.Load(w); err != nil {
		t.Fatal(err)
	}
	if err := guarded.Load(w); err != nil {
		t.Fatal(err)
	}
	turned := false
	pilot := PresentationPilot{goal: presentationGoal{x: 160, y: 104}}
	for range 8 {
		if _, err := held.Advance(Input{}); err != nil {
			t.Fatal(err)
		}
		s := guarded.State()
		input := pilot.tacticalMotion(s, MotionInput{})
		turned = turned || input != (MotionInput{})
		if _, err := guarded.Advance(Input{Motion: input}); err != nil {
			t.Fatal(err)
		}
	}
	if !turned || held.State().Equipment.Shield != 33 || guarded.State().Equipment.Shield != 39 {
		t.Fatalf("ordinary column avoidance: turned%v held%d guarded%d", turned, held.State().Equipment.Shield, guarded.State().Equipment.Shield)
	}
}

func TestDemoFifthColumnPredictionRejectsUnsupportedState(t *testing.T) {
	for _, fixture := range []struct {
		world  *World
		actor  *WorldActor
		passes int
	}{
		{}, {world: &World{}, actor: &WorldActor{}},
		{world: &World{}, actor: &WorldActor{fifthColumn: &FifthLaserColumnState{}}, passes: 1},
	} {
		if got, supported := demoFifthColumnPrediction(fixture.world, fixture.actor, fixture.passes); supported || !reflect.DeepEqual(got, demoActorView{}) {
			t.Fatal("invalid column state produced a prediction")
		}
	}
}
