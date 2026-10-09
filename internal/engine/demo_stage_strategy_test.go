package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

func TestDemoSecondDefenseTargetsClosedOppositeSideWithoutMutation(t *testing.T) {
	w := testWorld(t)
	w.Level.Number, w.ScrollY, w.Player.X, w.Player.Y = 2, 2560, 232, 166
	scheduler := NewSecondDefenseScheduler()
	scheduler.DefenseFlags = 2
	w.secondScheduler = &scheduler
	state := NewSecondDefenseNodeState(1, 4, 164, 12)
	actor := &WorldActor{Active: true, Health: 12, secondNode: &state, Collision: CollisionRect{Right: -1, Bottom: -1}}
	w.secondNodes[1] = actor
	beforePlayer, beforeEquipment, beforeRandom, beforeScheduler, beforeNode := w.Player, w.Equipment, w.RandomState(), scheduler, state
	pilot := DemoPilot{}
	input, handled := pilot.StageInput(w)
	if !handled || !input.Motion.Left || input.Motion.Right {
		t.Fatalf("closed left node did not request ordinary crossing: %+v handled=%v", input, handled)
	}
	if w.Player != beforePlayer || w.Equipment != beforeEquipment || w.RandomState() != beforeRandom || scheduler != beforeScheduler || state != beforeNode || actor.Health != 12 {
		t.Fatal("stage policy modified the arena rather than producing commands")
	}
	for range 4 {
		if next, handled := pilot.StageInput(w); !handled || next != input {
			t.Fatal("unchanged arena produced different input")
		}
	}
}

func TestDemoSecondDefenseHoldsOnlyVisibleNodeAndReleasesReadyTrigger(t *testing.T) {
	w := testWorld(t)
	w.Level.Number, w.ScrollY, w.Player.X, w.Player.Y = 2, 2560, 232, 166
	scheduler := NewSecondDefenseScheduler()
	scheduler.DefenseFlags = 2
	w.secondScheduler = &scheduler
	state := NewSecondDefenseNodeState(0, 14, 164, 12)
	w.secondNodes[0] = &WorldActor{Active: true, Health: 12, secondNode: &state}
	pilot := DemoPilot{}
	if input, handled := pilot.StageInput(w); !handled || !input.Motion.Down {
		t.Fatal("visible open node did not retain its firing window")
	}
	state.TileY = 184
	w.Player.Y = 176
	if input, handled := pilot.StageInput(w); !handled || input.Motion.Down {
		t.Fatal("off-screen node froze scrolling at the bottom")
	}
	w.blockedFireUntilRelease = true
	if input, _ := pilot.StageInput(w); input.Fire {
		t.Fatal("stage policy bypassed the READY trigger-release rule")
	}
	w.Ready = true
	if _, handled := pilot.StageInput(w); handled {
		t.Fatal("stage policy took over READY admission")
	}
}

// This uses original resources and public session commands from a basic level-2
// start. It proves the middle boundary only, without supplying campaign gear or
// arranging node damage, cash, invulnerability or the final guardian outcome.
func TestDemoSecondDefenseMiddleShopThroughPublicCommandsOptional(t *testing.T) {
	session, err := NewSession(playableOriginalWorldData(t, 2), 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	pilot := DemoPilot{}
	losses, continues, nodeHits, nodeDeaths := 0, 0, 0, 0
	for pass := 0; pass < 6500; pass++ {
		w := session.ActiveWorld()
		if w.GameOver {
			if !session.AcceptContinue() {
				t.Fatal("ordinary credits exhausted before the middle shop")
			}
			continues++
			w = session.ActiveWorld()
		}
		if w.ShopReady {
			if pass != 3841 || !w.secondMiddleReleased || w.secondDefenseRemaining != 0 || nodeDeaths != 3 || nodeHits != 36 || w.LevelFinished || w.Equipment.Lives != 1 || w.Equipment.Shield != 23 || w.Score != 2600 || w.Money != 1150 || losses != 5 || continues != 1 {
				t.Fatalf("middle boundary differs: pass%d released=%v nodes=%d kills=%d hits=%d final=%v lives%d shield%d score%d cash%d losses%d continues%d", pass, w.secondMiddleReleased, w.secondDefenseRemaining, nodeDeaths, nodeHits, w.LevelFinished, w.Equipment.Lives, w.Equipment.Shield, w.Score, w.Money, losses, continues)
			}
			t.Logf("Middle shop after %d public commands: lives=%d shield=%d score=%d money=%d losses=%d continues=%d", pass, w.Equipment.Lives, w.Equipment.Shield, w.Score, w.Money, losses, continues)
			return
		}
		for range 3 {
			w.AdvancePALTick()
		}
		input := pilot.NormalInput(w)
		beforeLives := w.Equipment.Lives
		var beforeHP [3]int
		for index, node := range w.secondNodes {
			if node != nil {
				beforeHP[index] = node.Health
			}
		}
		if _, err := session.Advance(input); err != nil {
			t.Fatal(err)
		}
		w = session.ActiveWorld()
		if w.Equipment.Lives < beforeLives {
			losses++
		}
		for index, node := range w.secondNodes {
			if node != nil && node.Health < beforeHP[index] {
				nodeHits++
				if node.Health == 0 {
					nodeDeaths++
				}
			}
		}
	}
	t.Fatal("bounded public-input policy did not reach the middle shop")
}

func TestDemoSecondDefenseTargetsHeadAndLeavesOffscreenPredictionToAvoidance(t *testing.T) {
	w := testWorld(t)
	w.Level.Number, w.ScrollY, w.Player.X, w.Player.Y = 2, 2560, 232, 166
	scheduler := NewSecondDefenseScheduler()
	scheduler.DefenseFlags = 3
	w.secondScheduler = &scheduler
	state := NewSecondDefenseNodeState(0, 14, 164, 12)
	w.secondNodes[0] = &WorldActor{Active: true, Health: 12, secondNode: &state}
	body := &WorldActor{Active: true, Visible: true, Health: 2, X: 232, Y: 100, PreviousX: 232, secondSegment: &SecondDefenseSegment{}, secondPart: &visualassets.GuardianComponent{Index: 1}, Collision: CollisionRect{Left: 220, Top: 90, Right: 240, Bottom: 110}}
	head := &WorldActor{Active: true, Visible: true, Health: 2, X: 100, Y: 100, PreviousX: 80, secondSegment: &SecondDefenseSegment{}, secondPart: &visualassets.GuardianComponent{Index: 0}, Collision: CollisionRect{Left: 90, Top: 90, Right: 110, Bottom: 110}}
	w.Actors = []*WorldActor{body, head}
	pilot := DemoPilot{}
	x, y, node, found := pilot.secondDefenseTarget(w)
	if !found || node || x != 220 || y != 148 {
		t.Fatalf("shielded arena target differs: x=%d y=%d node=%v found=%v", x, y, node, found)
	}
	head.X, head.PreviousX = 20, 50
	if _, handled := pilot.StageInput(w); handled {
		t.Fatal("off-screen predicted head bypassed ordinary terrain avoidance")
	}
}
