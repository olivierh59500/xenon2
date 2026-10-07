package engine

import "testing"

func TestDemoMotionForecastUsesNativeReverseAndTerrainRewind(t *testing.T) {
	for _, test := range []struct {
		name  string
		input MotionInput
		timer int
	}{{"reverse", MotionInput{Down: true}, 0}, {"rewind", MotionInput{Right: true}, 1}} {
		t.Run(test.name, func(t *testing.T) {
			w := testWorld(t)
			w.Level.Encounters.Moving = nil
			w.Player.X, w.Player.Y = 200, 168
			w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 1000, 1016, 1016
			w.Rewind = NewTerrainRewind(1000, 200, 120)
			w.Rewind.Timer = test.timer
			forecast := newDemoMotionForecast(w)
			player, camera, rewind, random := w.Player, w.ScrollY, w.Rewind, w.RandomState()
			if !forecast.advance(w, test.input) {
				t.Fatal("source-safe movement was rejected")
			}
			if w.Player != player || w.ScrollY != camera || w.Rewind != rewind || w.RandomState() != random {
				t.Fatal("motion forecast changed the live game")
			}
			for pass := 0; pass < 5; pass++ {
				if pass > 0 && !forecast.advance(w, test.input) {
					t.Fatal("source-safe continued movement was rejected")
				}
				if err := w.Step(Input{Motion: test.input}); err != nil {
					t.Fatal(err)
				}
				if forecast.player != w.Player || forecast.scroll.Y != w.ScrollY || forecast.scroll.Maximum != w.MaximumScrollY || forecast.rewind != w.Rewind {
					t.Fatalf("pass%d forecast differs from World.Step: player %+v want %+v, camera%d want%d", pass, forecast.player, w.Player, forecast.scroll.Y, w.ScrollY)
				}
			}
			if test.name == "reverse" && w.ScrollY <= 997 {
				t.Fatal("held down never requested the native backward camera step")
			}
		})
	}
}

func TestDemoSecondBackwardForecastIncludesLivingNodeBoundOptional(t *testing.T) {
	w, err := NewWorld(originalWorldData(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	w.Level.Encounters.Moving, w.Level.Encounters.Fixed = nil, nil
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 2700, 2701, 2701
	w.secondBackward, w.BaseScrollStep = true, -1
	w.Player.X, w.Player.Y = 160, 160
	forecast := newDemoMotionForecast(w)
	for pass := 0; pass < 6; pass++ {
		if !forecast.advance(w, MotionInput{}) {
			t.Fatal("clear original backward movement rejected")
		}
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
		if forecast.player != w.Player || forecast.scroll.Y != w.ScrollY || forecast.scroll.Maximum != w.MaximumScrollY {
			t.Fatalf("backward pass%d missed living node source bound: forecast%+v actual camera%d max%d", pass, forecast.scroll, w.ScrollY, w.MaximumScrollY)
		}
	}
}
