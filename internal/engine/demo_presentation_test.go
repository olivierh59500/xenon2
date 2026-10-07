package engine

import (
	"encoding/csv"
	"fmt"
	"os"
	"testing"

	"xenon2/internal/visualassets"
)

func presentationTestEnemy(id, x, y int) *WorldActor {
	return &WorldActor{ID: id, X: float64(x), Y: float64(y), PreviousX: float64(x), PreviousY: float64(y), Active: true, Visible: true, ActorList: "moving", Health: 3,
		Collision: CollisionRect{Left: x - 8, Top: y - 8, Right: x + 8, Bottom: y + 8}, part: &visualassets.ActorPart{DamageMode: "individual"}}
}

func TestPresentationPilotReleasesFireWithoutForwardOpportunity(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y = 160, 120
	p := PresentationPilot{}
	if p.NormalInput(w).Fire {
		t.Fatal("empty playfield caused an unnecessary firing command")
	}
	w.Actors = []*WorldActor{presentationTestEnemy(1, 250, 40)}
	if p.NormalInput(w).Fire {
		t.Fatal("off-axis enemy caused firing before alignment")
	}
	w.Actors[0] = presentationTestEnemy(2, 160, 170)
	if p.NormalInput(w).Fire {
		t.Fatal("enemy below a forward gun caused an unnecessary firing command")
	}
	w.Actors[0] = presentationTestEnemy(3, 160, 40)
	w.Actors[0].ActorList = "scenery"
	if p.NormalInput(w).Fire {
		t.Fatal("scenery outside the weapon callback caused firing")
	}
	w.Actors[0].ActorList, w.Actors[0].part.DamageMode = "moving", "block-shot"
	if p.NormalInput(w).Fire {
		t.Fatal("immune body segment caused firing")
	}
}

func TestPresentationPilotCommitsToEnemyAlignmentWithoutChangingWorld(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y = 160, 120
	w.Actors = []*WorldActor{presentationTestEnemy(1, 250, 40)}
	p := PresentationPilot{}
	p.NormalInput(w)
	w.Frame = 3
	p.NormalInput(w)
	w.Frame = 5
	player, equipment, random, pool, actor := w.Player, w.Equipment, w.RandomState(), *w.Pool, *w.Actors[0]
	input := p.NormalInput(w)
	if !input.Motion.Right || input.Fire {
		t.Fatalf("off-axis target should produce a deliberate rightward approach before firing: %+v", input)
	}
	for range 12 {
		if got := p.NormalInput(w); got != input {
			t.Fatalf("same pass changed the intended command: %+v => %+v", input, got)
		}
	}
	if w.Player != player || w.Equipment != equipment || w.RandomState() != random || *w.Pool != pool || w.Actors[0].Health != actor.Health || w.Actors[0].X != actor.X || w.Actors[0].Y != actor.Y {
		t.Fatal("presentation controller changed game state")
	}
	w.Actors = append(w.Actors, presentationTestEnemy(2, 90, 50))
	w.Frame = 6
	if got := p.NormalInput(w); !got.Motion.Right {
		t.Fatalf("new nearby enemy caused a one-pass attention reversal: %+v", got)
	}
}

func TestPresentationPilotHoldsNativeFireBurstThenPauses(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y = 160, 120
	w.Actors = []*WorldActor{presentationTestEnemy(1, 160, 40)}
	p := PresentationPilot{}
	for frame := 0; frame < 18; frame++ {
		w.Frame = uint64(frame)
		if !p.NormalInput(w).Fire {
			t.Fatalf("held firing burst was interrupted at pass %d", frame)
		}
	}
	for frame := 18; frame < 22; frame++ {
		w.Frame = uint64(frame)
		if p.NormalInput(w).Fire {
			t.Fatalf("pilot did not pause to reassess after its burst at pass %d", frame)
		}
	}
	w.Frame = 22
	if !p.NormalInput(w).Fire {
		t.Fatal("live aligned target did not restart a held burst")
	}
	w.Actors[0].Active = false
	w.Frame++
	if p.NormalInput(w).Fire {
		t.Fatal("pilot kept holding a burst after the target disappeared")
	}
	w.Ready = true
	if input := p.NormalInput(w); !input.Fire || input.Motion != (MotionInput{}) {
		t.Fatal("READY admission did not use an ordinary fire press")
	}
	w.Ready, w.blockedFireUntilRelease = false, true
	if p.NormalInput(w).Fire {
		t.Fatal("pilot did not release fire after READY")
	}
}

func TestPresentationPilotBonusApproachRejectsBlockedRoute(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y = 160, 120
	demoPilotCoverage(w, [2]int{192, w.ScrollY + 120})
	w.Collectibles = []*WorldCollectible{{ID: 1, Active: true, X: 224, Y: 120}, {ID: 2, Active: true, X: 112, Y: 120}}
	goal := presentationChooseGoal(w)
	if goal.bonus != w.Collectibles[1] {
		t.Fatal("reachable bonus did not take priority over a route crossing solid terrain")
	}
}

func TestPresentationPilotOnlyFiresAtActualDestructibleTerrain(t *testing.T) {
	w := testWorld(t)
	w.Player.X, w.Player.Y = 200, 176
	w.Level.Number, w.ScrollY = 2, 240
	demoPilotCoverage(w, [2]int{200, 400})
	if presentationShotOpportunity(w) {
		t.Fatal("ordinary solid terrain was mistaken for a bullet target")
	}
	w.secondTerrainCells = NewSecondTerrainCells([]visualassets.GuardianTerrainCell{{Quadrant: 0, X: 192, WorldY: 336}})
	if !presentationShotOpportunity(w) || !w.secondTerrainCells.Intact[0] {
		t.Fatal("real barrier ray was not recognized without changing the cell")
	}
	w.secondTerrainCells.Intact[0] = false
	if presentationShotOpportunity(w) {
		t.Fatal("cleared barrier kept an otherwise empty firing opportunity")
	}
}

// Original-resource motion checks complement visual recording review; they do
// not establish full-campaign completion or whether movement looks human.
func TestPresentationPilotOriginalOpeningTrajectoryOptional(t *testing.T) {
	s, err := NewSession(playableOriginalWorldData(t, 1), 1, NewRandomState())
	if err != nil {
		t.Fatal(err)
	}
	var trace *csv.Writer
	if path := os.Getenv("XENON2_PRESENTATION_TRACE"); path != "" {
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		trace = csv.NewWriter(file)
		defer trace.Flush()
		trace.Write([]string{"pass", "frame", "scroll_y", "ship_x", "ship_y", "left", "right", "up", "down", "fire", "shot_opportunity", "target_enemy", "target_bonus", "target_x", "target_y", "shield", "lives", "score", "money"})
	}
	p := PresentationPilot{}
	positions := map[[2]int]bool{}
	fired, empty, distance, losses, pauses, moving, passes := 0, 0, 0, 0, 0, 0, 0
	minX, maxX, minY, maxY := 320, 0, 192, 0
	previousFire := false
	for pass := 0; pass < 2000; pass++ {
		w := s.ActiveWorld()
		if w.GameOver || w.ShopReady || w.LevelFinished {
			break
		}
		for range 3 {
			w.AdvancePALTick()
		}
		input := p.NormalInput(w)
		opportunity := presentationShotOpportunity(w)
		if !w.Ready {
			if input.Fire {
				fired++
			}
			if !opportunity {
				empty++
				if input.Fire {
					t.Fatal("original playfield generated fire without a shot opportunity")
				}
			}
			if previousFire && !input.Fire {
				pauses++
			}
			previousFire = input.Fire
		}
		if trace != nil {
			enemy, bonus := 0, 0
			if p.goal.actor != nil {
				enemy = p.goal.actor.ID
			}
			if p.goal.bonus != nil {
				bonus = p.goal.bonus.ID
			}
			trace.Write([]string{fmt.Sprint(pass), fmt.Sprint(w.Frame), fmt.Sprint(w.ScrollY), fmt.Sprint(w.Player.X), fmt.Sprint(w.Player.Y), fmt.Sprint(input.Motion.Left), fmt.Sprint(input.Motion.Right), fmt.Sprint(input.Motion.Up), fmt.Sprint(input.Motion.Down), fmt.Sprint(input.Fire), fmt.Sprint(opportunity), fmt.Sprint(enemy), fmt.Sprint(bonus), fmt.Sprint(p.goal.x), fmt.Sprint(p.goal.y), fmt.Sprint(w.Equipment.Shield), fmt.Sprint(w.Equipment.Lives), fmt.Sprint(w.Score), fmt.Sprint(w.Money)})
		}
		x, y, lives, alive := w.Player.X, w.Player.Y, w.Equipment.Lives, w.PlayerAlive
		if _, err := s.Advance(input); err != nil {
			t.Fatal(err)
		}
		w = s.ActiveWorld()
		step := absDemo(w.Player.X-x) + absDemo(w.Player.Y-y)
		// Ordinary checkpoint respawns remain in the trace, but their camera
		// and ship resets do not count as input-driven pilot movement.
		if alive && w.PlayerAlive && w.Equipment.Lives == lives {
			distance += step
			if step > 0 {
				moving++
			}
		}
		if w.Equipment.Lives < lives {
			losses++
		}
		positions[[2]int{w.Player.X, w.Player.Y}] = true
		minX, maxX, minY, maxY = min(minX, w.Player.X), max(maxX, w.Player.X), min(minY, w.Player.Y), max(maxY, w.Player.Y)
		passes++
	}
	w := s.ActiveWorld()
	t.Logf("Opening: passes=%d positions=%d distance=%d moving=%d spanX=%d..%d spanY=%d..%d fire=%d empty=%d pauses=%d losses=%d lives=%d shield=%d scroll=%d score=%d money=%d shop=%t gameOver=%t", passes, len(positions), distance, moving, minX, maxX, minY, maxY, fired, empty, pauses, losses, w.Equipment.Lives, w.Equipment.Shield, w.ScrollY, w.Score, w.Money, w.ShopReady, w.GameOver)
	if passes < 1000 || len(positions) < 100 || maxX-minX < 80 || distance < 500 || empty < 100 || pauses < 10 || fired > passes*85/100 {
		t.Fatal("bounded original-resource pilot did not establish varied movement and selective firing")
	}
}
