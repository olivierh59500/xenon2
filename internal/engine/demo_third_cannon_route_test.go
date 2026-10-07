package engine

import "testing"

// These source scenes arrange encounter boundaries and use original damage
// callbacks to reproduce destroyed terrain. They verify navigation geometry,
// not a campaign victory or survival under ordinary player input.
func thirdRouteCannonBirth(t testing.TB, w *World, x, worldY int) *WorldActor {
	t.Helper()
	for _, record := range w.Level.Encounters.Fixed {
		if record.EnemyKind != 5 || record.X-8 != x || record.Y-8 != worldY {
			continue
		}
		w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = record.TriggerY, record.TriggerY+16, record.TriggerY+16
		w.Player.Y, w.Player.Inertia = 176, 0
		found := false
		for candidate := 14; candidate <= 304; candidate++ {
			if !w.Coverage.Touches(candidate, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
				w.Player.X, found = candidate, true
				break
			}
		}
		if !found {
			t.Fatal("source encounter boundary has no clear ship position")
		}
		w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
		w.cursor = EncounterCursor{MovingHighWater: record.TriggerY + 1, FixedHighWater: record.TriggerY + 1}
		if err := w.Step(Input{}); err != nil {
			t.Fatal(err)
		}
		for _, actor := range w.Actors {
			if actor.Active && actor.thirdCannon != nil && actor.thirdCannon.X == x && actor.thirdCannon.WorldY == worldY {
				if actor.Visible || actor.thirdCannon.Stage != 0 || actor.Health != 24 || actor.Binding.EntityID == 0 {
					t.Fatal("source cannon did not enter its untouched late-spawn state")
				}
				return actor
			}
		}
		t.Fatal("crossing the original encounter did not create its cannon")
	}
	t.Fatal("missing original cannon encounter")
	return nil
}

func thirdRouteCannonDamageStage(t testing.TB, w *World, actor *WorldActor) {
	t.Helper()
	if err := w.advanceThirdStage(); err != nil {
		t.Fatal(err)
	}
	w.advanceThirdCannon(actor)
	if actor.Collision.Empty() {
		t.Fatal("source cannon weak point did not become collidable")
	}
	x, y := (actor.Collision.Left+actor.Collision.Right)/2, (actor.Collision.Top+actor.Collision.Bottom)/2
	for range 24 {
		if !w.weaponHitPoint(x, y, 1) {
			t.Fatal("ordinary one-point projectile callback did not reach the cannon weak point")
		}
	}
}

func TestThirdBlockingCannonDestructionOpensOriginalPointRouteOptional(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	leftRear := thirdRouteCannonBirth(t, w, 16, 1888)
	thirdRouteCannonDamageStage(t, w, leftRear)
	thirdRouteCannonDamageStage(t, w, leftRear)
	blocking := thirdRouteCannonBirth(t, w, 224, 1696)
	leftFront := thirdRouteCannonBirth(t, w, 32, 1488)
	thirdRouteCannonDamageStage(t, w, leftFront)
	thirdRouteCannonDamageStage(t, w, leftFront)

	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 1360, 1376, 1376
	w.Player.X, w.Player.Y = 85, 176
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("source scene after the left cannon defeat is not clear")
	}
	n := demoNavigation{practiced: true}
	if _, _, found := n.pointWaypoint(w, 256, 1440); found {
		t.Fatal("forward firing point was reachable before the right blocker was destroyed")
	}
	score := w.Score
	thirdRouteCannonDamageStage(t, w, blocking)
	if !blocking.Active || blocking.thirdCannon.Stage != 1 || blocking.Health != 24 || w.Score != score+500 {
		t.Fatal("first defeat must preserve the upper cannon stage and award 500 points")
	}
	if _, _, found := n.pointWaypoint(w, 256, 1440); found {
		t.Fatal("first-stage defeat removed the upper terrain blocker prematurely")
	}
	thirdRouteCannonDamageStage(t, w, blocking)
	if blocking.Active || !blocking.thirdCannon.Removed || w.Score != score+1000 {
		t.Fatal("second defeat did not clear the blocker and award the second 500 points")
	}
	x, y, found := n.pointWaypoint(w, 256, 1440)
	if !found || !demoTestSegmentClear(&n, 85, 1536, demoNavPoint{x, y}) {
		t.Fatalf("source destruction did not open a clear first route leg: %d/%d found %v", x, y, found)
	}
	end := n.path[len(n.path)-1]
	if absDemo(end.x-256) > 6 || absDemo(end.y-1440) > 6 || w.Equipment.Lives != 3 || w.Equipment.Shield != 39 || w.Cheats.Enabled() {
		t.Fatalf("source point route or untouched initial equipment changed: end %+v ships %d shield %d", end, w.Equipment.Lives, w.Equipment.Shield)
	}
}

func TestThirdCannonPointRouteKeepsOriginalStateOptional(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 3))
	if err != nil {
		t.Fatal(err)
	}
	left := thirdRouteCannonBirth(t, w, 16, 1888)
	thirdRouteCannonDamageStage(t, w, left)
	thirdRouteCannonDamageStage(t, w, left)
	thirdRouteCannonBirth(t, w, 224, 1696)
	for _, fixture := range []struct {
		name      string
		x, worldY int
	}{{"full-approach", 77, 2204}, {"rear-turn", 176, 1862}} {
		t.Run(fixture.name, func(t *testing.T) {
			w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = fixture.worldY-176, fixture.worldY-160, fixture.worldY-160
			w.Player.X, w.Player.Y = fixture.x, 176
			w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
			player, rewind, random, pool := w.Player, w.Rewind, w.RandomState(), *w.Pool
			stage, equipment, score, money := w.ThirdStage, w.Equipment, w.Score, w.Money
			camera, maximum := w.ScrollY, w.MaximumScrollY
			terrain := append([]uint16(nil), w.Coverage.Map...)
			n := demoNavigation{practiced: true}
			x, y, found := n.pointWaypoint(w, 256, 1832)
			if !found || !demoTestSegmentClear(&n, fixture.x, fixture.worldY, demoNavPoint{x, y}) {
				t.Fatalf("right-cannon firing route was not found: %d/%d found %v nodes%d startSolid%v finalNil%v leftStage%d leftRemoved%v", x, y, found, len(n.nodes), n.touching(fixture.x, fixture.worldY), w.ThirdFinal == nil, left.thirdCannon.Stage, left.thirdCannon.Removed)
			}
			rearwardLeg := false
			for index, point := range n.path {
				if n.touching(point.x, point.y) || point.y > demoScrollMaximum(w, w.ScrollY, w.MaximumScrollY)+176 {
					t.Fatal("firing route includes covered or source-inaccessible terrain")
				}
				if index > 0 && point.y > n.path[index-1].y {
					rearwardLeg = true
				}
			}
			end := n.path[len(n.path)-1]
			if !rearwardLeg || absDemo(end.x-256) > 6 || absDemo(end.y-1832) > 6 {
				t.Fatalf("point navigation omitted the real rearward turn or exact firing lane: end %+v reverse %v", end, rearwardLeg)
			}
			if w.Player != player || w.Rewind != rewind || w.RandomState() != random || *w.Pool != pool || w.ThirdStage != stage || w.Equipment != equipment || w.Score != score || w.Money != money || w.ScrollY != camera || w.MaximumScrollY != maximum {
				t.Fatal("planning changed live source state")
			}
			for index, id := range terrain {
				if w.Coverage.Map[index] != id {
					t.Fatal("planning changed mutable terrain")
				}
			}
		})
	}
}

func BenchmarkThirdCannonPointNavigation(b *testing.B) {
	w, err := NewWorld(playableOriginalWorldData(b, 3))
	if err != nil {
		b.Fatal(err)
	}
	left := thirdRouteCannonBirth(b, w, 16, 1888)
	thirdRouteCannonDamageStage(b, w, left)
	thirdRouteCannonDamageStage(b, w, left)
	thirdRouteCannonBirth(b, w, 224, 1696)
	w.ScrollY, w.MaximumScrollY, w.VisitedScrollY = 2028, 2044, 2044
	w.Player.X, w.Player.Y = 77, 176
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	b.Run("first-route", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			n := demoNavigation{practiced: true}
			if _, _, found := n.pointWaypoint(w, 256, 1832); !found {
				b.Fatal("source firing route was not found")
			}
		}
	})
	b.Run("cached-waypoint", func(b *testing.B) {
		n := demoNavigation{practiced: true}
		if _, _, found := n.pointWaypoint(w, 256, 1832); !found {
			b.Fatal("source firing route was not found")
		}
		b.ReportAllocs()
		for b.Loop() {
			if _, _, found := n.pointWaypoint(w, 256, 1832); !found {
				b.Fatal("unchanged source firing route was discarded")
			}
		}
	})
}
