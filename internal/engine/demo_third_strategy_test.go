package engine

import (
	"testing"
	"xenon2/internal/visualassets"
)

// This arranges an original final checkpoint, not a campaign victory. The map,
// gear, health, paths and encounter streams remain in use; every subsequent
// guardian hit and coin change must come from ordinary public commands.
func TestDemoThirdFinalWormBoundaryOptional(t *testing.T) {
	session, err := NewSession(playableOriginalWorldData(t, 3), 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	w := session.ActiveWorld()
	w.Checkpoint.ScrollY, w.Checkpoint.PlayerX = 176, 152
	w.MinimumScrollY = 0
	w.RestartCheckpoint()
	if w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
		t.Fatal("original final checkpoint geometry is not clear")
	}
	pilot := DemoPilot{}
	continues := 0
	for pass := 0; pass < 4000; pass++ {
		w = session.ActiveWorld()
		if w.GameOver {
			if !session.AcceptContinue() {
				t.Fatalf("credits exhausted at pass %d with final HP %d", pass, w.ThirdFinal.Health)
			}
			continues++
			w = session.ActiveWorld()
		}
		if w.ShopReady {
			if pass != 1540 || !w.LevelFinished || !w.ExitReady || !w.ThirdFinal.Defeated || w.ThirdFinal.Health != 0 || w.Equipment.Lives != 1 || w.Equipment.Shield != 31 || continues != 1 || w.PendingExitDrops != 0 {
				t.Fatalf("verified final outcome changed: pass %d HP %d lives %d shield %d continues %d pending %d", pass, w.ThirdFinal.Health, w.Equipment.Lives, w.Equipment.Shield, continues, w.PendingExitDrops)
			}
			t.Logf("Final shop after %d ordinary commands with 1 life, shield 31 and 1 legal continue", pass)
			return
		}
		for range 3 {
			w.AdvancePALTick()
		}
		input, handled := pilot.ThirdGuardianInput(w)
		if !handled {
			input = pilot.NormalInput(w)
		}
		if _, err = session.Advance(input); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatal("bounded ordinary controls did not defeat the final worm and reach its genuine shop")
}

func TestDemoThirdFinalPredictsOnlyHeadWithoutChangingWorld(t *testing.T) {
	w := testWorld(t)
	w.Level.Number, w.ScrollY, w.Player.X, w.Player.Y = 3, 160, 150, 176
	final := NewThirdFinalState(80)
	w.ThirdFinal = &final
	path := &visualassets.Path{Commands: []visualassets.PathCommand{{Kind: "origin", X: 160, Y: 220}, {Kind: "pause", Duration: 80}, {Kind: "jump", Target: 1}}}
	descriptor := &visualassets.GuardianComponent{Index: 0, PathBudget: 8, Behavior: "worm-head"}
	state, err := NewThirdFinalMember(path, *descriptor, w.ScrollY, 1)
	if err != nil {
		t.Fatal(err)
	}
	head := &WorldActor{Active: true, ActorList: "moving", X: 160, Y: 60, PreviousX: 160, PreviousY: 60, thirdPart: descriptor, thirdFinalMember: &state, path: path, Collision: CollisionRect{Left: 155, Top: 55, Right: 165, Bottom: 65}}
	neck := &WorldActor{Active: true, ActorList: "moving", X: 20, Y: 60, thirdPart: &visualassets.GuardianComponent{Index: 1}, thirdFinalMember: &ThirdFinalMember{}, path: path, Collision: CollisionRect{Left: 15, Top: 55, Right: 25, Bottom: 65}}
	w.Actors = []*WorldActor{neck, head}
	pilot := DemoPilot{}
	player, equipment, random, enemyMotion, health := w.Player, w.Equipment, w.RandomState(), state.Motion, final.Health
	input, handled := pilot.ThirdGuardianInput(w)
	if !handled || !input.Motion.Right || input.Motion.Left {
		t.Fatalf("policy selected a blocking neck instead of head: %+v handled %v", input, handled)
	}
	if w.Player != player || w.Equipment != equipment || w.RandomState() != random || state.Motion != enemyMotion || final.Health != health {
		t.Fatal("prediction changed game state instead of issuing controls")
	}
	w.blockedFireUntilRelease = true
	if input, _ = pilot.ThirdGuardianInput(w); input.Fire {
		t.Fatal("final policy bypassed READY trigger-release rule")
	}
	w.Ready = true
	if _, handled = pilot.ThirdGuardianInput(w); handled {
		t.Fatal("final policy took over READY admission")
	}
}
