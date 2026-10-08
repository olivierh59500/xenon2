package engine

import "testing"

func originalSecondNodeContactFixture(t testing.TB, x, y int) *World {
	t.Helper()
	e := NewEquipment()
	for _, i := range []Item{ItemPowerup, ItemCannon, ItemRearShot, ItemSpeedup, ItemSpeedup, ItemAutofire, ItemAutofire} {
		e.ApplyItem(i)
	}
	d := playableOriginalWorldData(t, 2)
	d.InitialEquipment = &e
	w, err := NewWorld(d)
	if err != nil {
		t.Fatal(err)
	}
	w.Ready, w.MaterializationFrames = false, 0
	w.ScrollY, w.MinimumScrollY, w.MaximumScrollY, w.VisitedScrollY = 2528, 2528, 2544, 2544
	w.Player.X, w.Player.Y, w.Player.SpeedTier = x, y, 2
	w.Rewind = NewTerrainRewind(2528, x, y)
	w.Frame = 2113
	w.cursor = EncounterCursor{MovingHighWater: 0, FixedHighWater: 0}
	if err := w.advanceSecondDefenseWaves(); err != nil {
		t.Fatal(err)
	}
	if err := w.advanceActorPhase(ActorPoolMoving, Input{}); err != nil {
		t.Fatal(err)
	}
	var head *WorldActor
	for _, a := range w.Actors {
		if a.Active && a.secondSegment != nil && a.secondSegment.Stream == 0 && a.secondPart.Index == 0 {
			head = a
			break
		}
	}
	if head == nil {
		t.Fatal("original scheduler did not create stream-zero head")
	}
	w.damageSecondSegment(head, uint16(head.Health))
	if w.secondScheduler.DefenseFlags != 2 {
		t.Fatal("native head destruction did not leave one active stream")
	}

	w.advanceSecondNode(w.secondNodes[0])
	node := w.secondNodes[0]
	if node.Collision != (CollisionRect{Left: 224, Top: 96, Right: 239, Bottom: 111}) || node.Visible {
		t.Fatalf("source hidden node prefix differs: %+v visible%v", node.Collision, node.Visible)
	}
	// Preserve the native callback and leave the captured four health using
	// ordinary two-damage point hits, rather than replacing its state or prefix.
	for hit := 0; hit < 64 && node.Health > 4; hit++ {
		before := node.Health
		if !w.weaponHitPoint(232, 104, 2) {
			t.Fatal("original node point callback rejected captured health preparation")
		}
		if node.Health != before-2 {
			t.Fatal("original node hit was consumed without the expected damage")
		}
	}
	if node.Health != 4 {
		t.Fatal("bounded native point hits did not reach captured four health")
	}
	w.advanceSecondNode(node)
	return w
}
func TestOriginalSecondNodeRiskAvoidsCapturedPersistentContact(t *testing.T) {
	w := originalSecondNodeContactFixture(t, 255, 101)
	p := DemoPilot{practicedRoute: true}
	before := forecastIsolationDigest(w)
	chosen := demoSecondArenaBeam(w, &p, MotionInput{Left: true, Down: true})
	if forecastIsolationDigest(w) != before {
		t.Fatal("node risk planning changed the source world, scheduler, gates or random stream")
	}
	if chosen != (MotionInput{Down: true}) {
		t.Fatalf("beam failed to stay outside the persistent node edge: %+v", chosen)
	}
	var safe, unsafe WorldForecast
	if err := safe.Load(w); err != nil {
		t.Fatal(err)
	}
	if err := unsafe.Load(w); err != nil {
		t.Fatal(err)
	}
	for pass := 0; pass < 6; pass++ {
		for range 3 {
			safe.AdvancePALTick()
			unsafe.AdvancePALTick()
		}
		bad := Input{Motion: MotionInput{Down: true}}
		if pass == 0 {
			bad.Motion.Left = true
		}
		a, err := safe.Advance(Input{Motion: chosen})
		if err != nil {
			t.Fatal(err)
		}
		b, err := unsafe.Advance(bad)
		if err != nil {
			t.Fatal(err)
		}
		if !a.Alive || a.Shield != 39 {
			t.Fatalf("ordinary avoidance lost shield at pass%d: %+v", pass, a)
		}
		if pass >= 1 && pass <= 4 && b.Shield != 39-8*pass {
			t.Fatalf("captured persistent contact differs at pass%d: %+v", pass, b)
		}
		if safe.State().secondScheduler.DefenseFlags != 2 || unsafe.State().secondScheduler.DefenseFlags != 2 {
			t.Fatal("fixture replaced native stream callbacks or closed the node")
		}
	}
	if unsafe.State().Equipment.Shield != 7 || unsafe.State().secondNodes[0].Health != 4 || safe.State().secondNodes[0].Health != 4 {
		t.Fatal("original persistent tag84 callback or captured health changed")
	}
	// After passing below the node, ordinary alignment still permits a real
	// forward shot. The safety fix must not make an intact node untargetable.
	for range 3 {
		for range 3 {
			safe.AdvancePALTick()
		}
		if _, err := safe.Advance(Input{Motion: MotionInput{Left: true}}); err != nil {
			t.Fatal(err)
		}
	}
	if !presentationShotOpportunityForMotion(safe.State(), MotionInput{}) {
		t.Fatal("safe legal controls lost the intact node firing opportunity")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("fixture validation changed the live source state")
	}
}

func TestOriginalSecondNodeRiskPreservesRewindFallback(t *testing.T) {
	w := originalSecondNodeContactFixture(t, 255, 101)
	w.Rewind.Timer = 1
	before := forecastIsolationDigest(w)
	a, b := DemoPilot{practicedRoute: true}, DemoPilot{practicedRoute: true}
	fallback := MotionInput{Down: true, Left: true}
	if got, want := demoSecondArenaBeam(w, &a, fallback), demoSecondArenaBeamOriginal(w, &b, fallback); got != want {
		t.Fatal("unsupported rewind changed the original stream beam")
	}
	if forecastIsolationDigest(w) != before {
		t.Fatal("rewind fallback changed source state")
	}
}
