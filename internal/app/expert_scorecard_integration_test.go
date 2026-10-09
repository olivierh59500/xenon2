package app

import (
	"os"
	"testing"

	"xenon2/internal/engine"
)

// These observations come from the real intro, merchants and ordinary commands.
// A scoring defeat counts the final removal of a damageable actor; the two
// stages of a compound cannon are not counted as two separate enemies.
func TestExpertThreeLevelScorecardOptional(t *testing.T) {
	if os.Getenv("XENON2_EXPERT_SCORECARD") == "" {
		t.Skip("enable the full expert scorecard explicitly")
	}
	for _, explicit := range []bool{false, true} {
		g, err := NewConfiguredGame(frontendGame(t).Bundle, Config{Level: 1, StartScreen: PresentationScreen, Mute: true, Demo: explicit, HumanDemo: explicit})
		if err != nil {
			t.Fatal(err)
		}
		var losses, cash, defeats [4]int
		continues := 0
		finished := false
		for update := 0; update < 60*2400; update++ {
			var before *engine.World
			if d, ok := g.Driver.(*worldDriver); ok {
				before = d.world
			}
			oldMoney, oldScore, oldCredits, oldFrame, alive := 0, 0, 0, uint64(0), false
			var actors [engine.ActorPoolCapacity]*engine.WorldActor
			var health [engine.ActorPoolCapacity]int
			count := 0
			if before != nil {
				oldMoney, oldScore, oldCredits, oldFrame, alive = before.Money, before.Score, before.ContinueCredits, before.Frame, before.PlayerAlive
				for _, actor := range before.Actors {
					if actor.Active && actor.ActorList == "moving" && int16(uint16(actor.Health)) > 0 && count < len(actors) {
						actors[count], health[count] = actor, actor.Health
						count++
					}
				}
			}
			wasPlaying := g.Screen == LevelScreen
			advanceFrontend(t, g, inputFrame{})
			d, ok := g.Driver.(*worldDriver)
			if !ok || d.session == nil {
				continue
			}
			w := d.world
			level := w.Level.Number
			if level < 1 || level > 3 || w.Cheats.Enabled() || d.diagnostic {
				t.Fatal("scorecard left ordinary three-level gameplay")
			}
			if w == before {
				if alive && !w.PlayerAlive {
					losses[level]++
				}
				continues += max(0, oldCredits-w.ContinueCredits)
				if wasPlaying {
					cash[level] += max(0, w.Money-oldMoney)
				}
				if oldFrame != w.Frame && w.Score > oldScore {
					for index, actor := range actors[:count] {
						if health[index] > 0 && !actor.Active && int16(uint16(actor.Health)) <= 0 {
							defeats[level]++
						}
					}
				}
			}
			if level == 3 && !g.DemoActive() && g.Screen == TitleScreen {
				if !w.ThirdFinal.Defeated || !w.LevelFinished || !w.ExitReady || w.PendingExitDrops != 0 {
					t.Fatal("scorecard omitted the real final guardian, rewards or merchant")
				}
				for level := 1; level <= 3; level++ {
					t.Logf("EXPERT_SCORECARD explicit%v level%d shipsLost%d scoringDefeats%d collectedCash%d", explicit, level, losses[level], defeats[level], cash[level])
				}
				if continues != 0 || losses[3] != 0 || losses[1]+losses[2]+losses[3] > 1 {
					t.Fatal("expert tour exceeded one ordinary ship loss or spent a continue")
				}
				t.Logf("EXPERT_FINISH explicit%v seconds%.2f ships%d continues%d score%d", explicit, float64(update)/60, w.Equipment.Lives, w.ContinueCredits, w.Score)
				finished = true
				break
			}
		}
		if !finished {
			t.Fatal("expert scorecard did not complete its actual three-level tour")
		}
	}
}
