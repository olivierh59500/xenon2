package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestDemoPilotUsesCommandsWithoutMutatingWorld(t *testing.T) {
	w := testWorld(t)
	pilot := DemoPilot{}
	player, equipment, random, frame, pool := w.Player, w.Equipment, w.RandomState(), w.Frame, *w.Pool
	first := pilot.NormalInput(w)
	for range 20 {
		if got := pilot.NormalInput(w); got != first {
			t.Fatal("same world state produced different commands")
		}
	}
	if w.Player != player || w.Equipment != equipment || w.RandomState() != random || w.Frame != frame || *w.Pool != pool {
		t.Fatal("pilot changed game state instead of producing ordinary input")
	}
	w.Ready = true
	if input := pilot.NormalInput(w); !input.Fire || input.Motion != (MotionInput{}) {
		t.Fatal("READY admission was not an ordinary fire command")
	}
	w.Ready = false
	w.blockedFireUntilRelease = true
	if pilot.NormalInput(w).Fire {
		t.Fatal("pilot did not release the trigger after READY")
	}
}

func demoPilotCoverage(w *World, points ...[2]int) {
	var full [16]uint16
	for row := range full {
		full[row] = 0xffff
	}
	w.Coverage = &TerrainCoverage{Columns: 20, Rows: 300, Map: make([]uint16, 6000), coverage: map[uint16][16]uint16{1: full}}
	w.Level.PlayerStencil = &visualassets.PlayerTerrainStencil{Width: 1, Height: 1, Rows: []uint32{0x80000000}}
	for _, point := range points {
		w.Coverage.Map[(point[1]/16)*20+point[0]/16] = 1
	}
}

func TestDemoPilotPrefersReachableBonusWithoutChangingInventory(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y = 160, 120
	demoPilotCoverage(w, [2]int{176, w.ScrollY + 120})
	w.Collectibles = []*WorldCollectible{{Active: true, X: 176, Y: 120}, {Active: true, X: 100, Y: 120}}
	p := DemoPilot{}
	before, random := w.Equipment, w.RandomState()
	i := p.NormalInput(w)
	if !i.Motion.Left || i.Motion.Right {
		t.Fatalf("pilot did not prefer the accessible bonus over the nearer blocked one: %+v", i)
	}
	if w.Equipment != before || w.RandomState() != random || !w.Collectibles[0].Active || !w.Collectibles[1].Active {
		t.Fatal("bonus planning awarded inventory or modified the game")
	}
}

func TestDemoPilotDiveRequiresChargeThreatAndClearResurfacing(t *testing.T) {
	w := testWorld(t)
	w.MaterializationFrames = 0
	w.Actors = []*WorldActor{{Active: true, ActorList: "moving", Collision: CollisionRect{Left: 0, Top: 0, Right: 319, Bottom: 191}}}
	p := DemoPilot{}
	if p.NormalInput(w).Dive {
		t.Fatal("pilot requested a dive without a charge")
	}
	w.Equipment.DiveCharges = 1
	if !p.NormalInput(w).Dive || w.Equipment.DiveCharges != 1 {
		t.Fatal("unavoidable threat did not produce an ordinary dive command without spending the charge")
	}
	demoPilotCoverage(w, [2]int{w.Player.X, w.ScrollY - 136 + w.Player.Y})
	if p.NormalInput(w).Dive {
		t.Fatal("pilot requested a dive into known resurfacing terrain")
	}
	w.Coverage = nil
	w.Actors = nil
	if p.NormalInput(w).Dive {
		t.Fatal("pilot used dive without an immediate threat")
	}
}

func TestDemoNavigationUsesWholeStencilAndRetainsCachedRoute(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y = 160, 120
	demoPilotCoverage(w, [2]int{176, w.ScrollY + 120})
	n := demoNavigation{}
	x, y, found := n.waypoint(w, w.ScrollY+w.Player.Y-64)
	if !found || n.touching(x, y) {
		t.Fatal("navigation failed to produce a clear terrain waypoint")
	}
	path, frame := &n.path[0], n.frame
	w.Frame++
	x2, y2, found := n.waypoint(w, w.ScrollY+w.Player.Y-64)
	if !found || x2 != x || y2 != y || &n.path[0] != path || n.frame != frame {
		t.Fatal("unchanged corridor unnecessarily discarded its cached route")
	}
	w.Coverage.Rows = 1
	if _, _, found = n.waypoint(w, 0); found {
		t.Fatal("unsupported diagnostic map dimensions were treated as an original level")
	}
}

func BenchmarkDemoPilotCachedNavigation(b *testing.B) {
	w, err := NewWorld(LevelData{Number: 1, Terrain: &visualassets.Terrain{Columns: 20, Rows: 300, TileSize: 16, Map: make([]uint16, 6000)}, Paths: &visualassets.Paths{}, Actors: &visualassets.Actors{}, Encounters: &visualassets.Encounters{}})
	if err != nil {
		b.Fatal(err)
	}
	demoPilotCoverage(w)
	p := DemoPilot{}
	p.NormalInput(w)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		p.NormalInput(w)
	}
}

func TestDemoPilotDoesNotHoldReverseAtBottomUnderThreat(t *testing.T) {
	w := testWorld(t)
	w.Player.Y = 176
	w.MaterializationFrames = 0
	w.Actors = []*WorldActor{{Active: true, ActorList: "moving", Collision: CollisionRect{Left: 0, Top: 0, Right: 319, Bottom: 191}}}
	before := w.Player
	i := (&DemoPilot{}).NormalInput(w)
	if i.Motion.Down {
		t.Fatal("unavoidable short-term hazard caused a persistent bottom reverse request")
	}
	if w.Player != before {
		t.Fatal("reverse avoidance changed physical player state")
	}
}

func TestDemoPilotSecondOpeningCheckpointOptional(t *testing.T) {
	data := playableOriginalWorldData(t, 2)
	s, err := NewSession(data, 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	pilot := DemoPilot{}
	for pass := 0; pass < 700; pass++ {
		w := s.ActiveWorld()
		for range 3 {
			w.AdvancePALTick()
		}
		if _, err := s.Advance(pilot.NormalInput(w)); err != nil {
			t.Fatal(err)
		}
		w = s.ActiveWorld()
		if w.Checkpoint.ScrollY <= 4032 {
			if pass+1 != 578 || w.Checkpoint.ScrollY != 4032 || w.Equipment.Lives != 3 || w.Equipment.Shield != 27 || w.ContinueCredits != 2 || w.LevelFinished {
				t.Fatalf("verified opening policy changed: pass%d checkpoint%d lives%d shield%d credits%d", pass+1, w.Checkpoint.ScrollY, w.Equipment.Lives, w.Equipment.Shield, w.ContinueCredits)
			}
			t.Logf("Reached level2 checkpoint4032 through%d ordinary public commands with all3 ships", pass+1)
			return
		}
	}
	t.Fatal("bounded pilot opening did not reach checkpoint4032")
}

func TestDemoPilotFirstMiddleShopThroughPublicCommandsOptional(t *testing.T) {
	data := playableOriginalWorldData(t, 1)
	s, err := NewSession(data, 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	pilot := DemoPilot{}
	losses, continues := 0, 0
	for pass := 0; pass < 4000; pass++ {
		w := s.ActiveWorld()
		if w.GameOver {
			if !s.AcceptContinue() {
				t.Fatal("pilot exhausted its ordinary continue credits before the middle shop")
			}
			continues++
		}
		w = s.ActiveWorld()
		if w.ShopReady {
			if pass != 2025 || w.FirstMiddle == nil || !w.FirstMiddle.Crossed || w.ScrollY != 2495 || w.Equipment.Lives != 3 || w.Equipment.Shield != 39 || w.Money != 800 || w.Score != 11300 || w.ContinueCredits != 2 || losses != 0 || continues != 0 || w.LevelFinished {
				t.Fatalf("bounded middle-shop outcome: pass%d camera%d lives%d shield%d score%d money%d losses%d continues%d", pass, w.ScrollY, w.Equipment.Lives, w.Equipment.Shield, w.Score, w.Money, losses, continues)
			}
			t.Logf("Reached the genuine middle-shop request after%d ordinary commands with all three ships, full shield and both continues", pass)
			return
		}
		for range 3 {
			w.AdvancePALTick()
		}
		lives := w.Equipment.Lives
		if _, err := s.Advance(pilot.NormalInput(w)); err != nil {
			t.Fatal(err)
		}
		if s.ActiveWorld().Equipment.Lives < lives {
			losses++
		}
	}
	t.Fatal("bounded pilot did not cross the first middle-shop gate")
}

func TestDemoNavigationCachedBitsMatchOriginalStencilOptional(t *testing.T) {
	w, err := NewWorld(playableOriginalWorldData(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	n := demoNavigation{}
	n.refresh(w)
	for y := 2800; y < 3400; y += 11 {
		for x := 14; x <= 304; x += 7 {
			if n.touching(x, y) != w.Coverage.Touches(x, y, 0, *w.Level.PlayerStencil) {
				t.Fatalf("cached navigation coverage differs at%d,%d", x, y)
			}
		}
	}
}
