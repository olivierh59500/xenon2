package engine

import "testing"

func BenchmarkNativeMotionOriginalRoutes(b *testing.B) {
	for _, fixture := range []struct {
		name                           string
		x, y, camera, targetX, targetY int
	}{
		{"rear-corner", 193, 176, 1896, 202, 2077},
		{"three-pixel-lane", 251, 176, 1732, 254, 1908},
	} {
		b.Run(fixture.name, func(b *testing.B) {
			w := nativeMotionFixture(b, fixture.x, fixture.y, fixture.camera)
			var planner nativeMotionPlanner
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if !planner.search(w, fixture.targetX, fixture.targetY) {
					b.Fatal("source route was not found")
				}
			}
			b.ReportMetric(float64(planner.expanded), "expanded/op")
			b.ReportMetric(float64(planner.nodeCount()), "states/op")
		})
	}
}

func TestNativeMotionTimeBoundUsesSourceMovement(t *testing.T) {
	cases := 0
	for speed := 0; speed <= 2; speed++ {
		for _, height := range []int{16, 120, 175, 176, 180} {
			for _, deviation := range []int{0, 1, 33, 34, 35, 40, 100} {
				for _, input := range demoDirections {
					startPlayer := PlayerMotionState{X: 160, Y: height, SpeedTier: speed}
					startScroll := ScrollState{Y: 1000, Maximum: 10000, DeviationPasses: deviation}
					player, scroll := startPlayer, startScroll
					for passes := 1; passes <= 40; passes++ {
						player.Advance(input, MotionContext{ScrollY: scroll.Y, VisitedScrollY: 10000, BaseScrollStep: 1})
						// A loose reverse limit isolates the source speed bound.
						scroll.Maximum = 10000
						scroll.Advance(player.ScrollStep, 1, input.Down)
						start := demoMotionForecast{player: startPlayer, scroll: startScroll}
						bound := nativeMotionTimeBound(start, player.X, player.Y+scroll.Y, speed)
						if bound > passes {
							t.Fatalf("source travel time overestimated: speed%d height%d deviation%d input%+v passes%d bound%d", speed, height, deviation, input, passes, bound)
						}
						cases++
					}
				}
			}
		}
	}
	if cases != 37800 {
		t.Fatal("incomplete source movement bound coverage")
	}
}

// These original-map poses isolate clear route execution. They do not recreate
// the enemies or earned inventory of a connected campaign.
func nativeMotionFixture(t testing.TB, x, y, camera int) *World {
	t.Helper()
	w, err := NewWorld(playableOriginalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = camera, 0, camera+16, camera+16
	w.Player.X, w.Player.Y = x, y
	w.Equipment.ApplyItem(ItemSpeedup)
	w.Equipment.ApplyItem(ItemSpeedup)
	w.Rewind = NewTerrainRewind(camera, x, y)
	w.ThirdMiddle = &ThirdGuardianState{Defeated: true}
	w.cursor = EncounterCursor{MovingHighWater: 0, FixedHighWater: 0}
	if w.Coverage.Touches(x, y, camera, *w.Level.PlayerStencil) {
		t.Fatal("original native-motion fixture starts in covered terrain")
	}
	return w
}

func TestNativeMotionOriginalCornerAndThreePixelAlignmentOptional(t *testing.T) {
	for _, fixture := range []struct {
		name                                     string
		x, y, camera, targetX, targetY, commands int
	}{
		{"rear-corner", 193, 176, 1896, 202, 2077, 5},
		{"three-pixel-lane", 251, 176, 1732, 254, 1908, 7},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			w := nativeMotionFixture(t, fixture.x, fixture.y, fixture.camera)
			before := forecastDigest(w)
			var planner nativeMotionPlanner
			if !planner.search(w, fixture.targetX, fixture.targetY) || len(planner.commands) != fixture.commands {
				t.Fatalf("bounded source route differs: nodes%d commands%d", planner.expanded, len(planner.commands))
			}
			if planner.expanded > nativeMotionNodeBudget || forecastDigest(w) != before {
				t.Fatal("planning exceeded its bound or changed live source state")
			}
			for index, input := range planner.commands {
				for range 3 {
					w.AdvancePALTick()
				}
				if err := w.Step(Input{Motion: input}); err != nil {
					t.Fatal(err)
				}
				if !nativeMotionMatches(w, planner.states[index+1]) || w.Rewind.Timer != 0 || w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
					t.Fatalf("source movement differs at command%d: ship%+v camera%d", index+1, w.Player, w.ScrollY)
				}
			}
			if w.Player.X != fixture.targetX || w.Player.Y+w.ScrollY != fixture.targetY {
				t.Fatal("nearby position was accepted without exact alignment")
			}
			if fixture.name == "three-pixel-lane" {
				navigation := demoNavigation{practiced: true}
				navigation.refresh(w)
				if !navigation.clearSegment(w.Player.X, w.Player.Y+w.ScrollY, demoNavPoint{254, 1932}) {
					t.Fatal("exact source alignment did not unlock the following clear segment")
				}
			}
		})
	}
}

func TestNativeMotionCommittedCommandsKeepTheirIntermediateGoalOptional(t *testing.T) {
	w := nativeMotionFixture(t, 193, 176, 1896)
	var planner nativeMotionPlanner
	input, found := planner.command(w, 202, 2077, 256, 1832)
	if !found {
		t.Fatal("source corner commitment was not created")
	}
	for command := 0; command < 5; command++ {
		if err := w.Step(Input{Motion: input}); err != nil {
			t.Fatal(err)
		}
		if command == 4 {
			break
		}
		want := planner.commands[planner.at]
		// A geometric lookahead can change during braking. Its parent firing
		// target stays fixed, so the already validated sequence must continue.
		input, found = planner.command(w, 254, 1932, 256, 1832)
		if !found || input != want {
			t.Fatal("a changing geometric lookahead restarted committed braking")
		}
	}
	if w.Player.X != 202 || w.Player.Y+w.ScrollY != 2077 {
		t.Fatal("committed source commands did not escape the rearward corner")
	}
}

func TestNativeMotionCommitmentRejectsMismatchTargetAndCoveredPoseOptional(t *testing.T) {
	for _, failure := range []string{"state", "target", "terrain"} {
		t.Run(failure, func(t *testing.T) {
			w := nativeMotionFixture(t, 193, 176, 1896)
			var planner nativeMotionPlanner
			input, found := planner.command(w, 202, 2077, 256, 1832)
			if !found {
				t.Fatal("missing source corner plan")
			}
			if err := w.Step(Input{Motion: input}); err != nil {
				t.Fatal(err)
			}
			targetX := 256
			switch failure {
			case "state":
				w.Player.Inertia++
			case "target":
				targetX = 160
			case "terrain":
				state := planner.states[len(planner.states)-1]
				var opaque uint16
				for _, tile := range w.Level.Terrain.Tiles {
					if !tile.Masked {
						opaque = tile.ID
						break
					}
				}
				if opaque == 0 {
					t.Fatal("original map has no opaque coverage tile")
				}
				w.setSecondMapCell(state.player.X/16, (state.player.Y+state.scroll.Y)/16, opaque)
			}
			if _, found := planner.continueRoute(w, targetX, 1832); found {
				t.Fatal("invalid source state, target or terrain retained a committed command")
			}
		})
	}
}

func TestNativeMotionOtherCandidateDoesNotEraseCommitmentOptional(t *testing.T) {
	w := nativeMotionFixture(t, 193, 176, 1896)
	var planner nativeMotionPlanner
	input, found := planner.command(w, 202, 2077, 256, 1832)
	if !found {
		t.Fatal("source corner plan was not created")
	}
	if err := w.Step(Input{Motion: input}); err != nil {
		t.Fatal(err)
	}
	at, want := planner.at, planner.commands[planner.at]
	if _, found := planner.continueRoute(w, 160, 1832); found || planner.world != w || planner.at != at {
		t.Fatal("querying another cannon erased or consumed the retained target's command")
	}
	if input, found := planner.continueRoute(w, 256, 1832); !found || input != want || planner.at != at+1 {
		t.Fatal("the retained cannon target did not resume after another candidate was queried")
	}
}
