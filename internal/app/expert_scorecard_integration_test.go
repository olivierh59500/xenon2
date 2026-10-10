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
		var seenRewards, collectedRewards, missedRewards [4][2]int
		type rewardObservation struct {
			item        *engine.WorldCollectible
			level, kind int
			counted     bool
		}
		var rewards []rewardObservation
		knownRewards := make(map[*engine.WorldCollectible]bool)
		observeRewards := func(w *engine.World) {
			if w == nil || w.Level.Number < 1 || w.Level.Number > 3 {
				return
			}
			for _, item := range w.Collectibles {
				if !item.Active || knownRewards[item] {
					continue
				}
				knownRewards[item] = true
				kind := 0 // Cash and equipment bubbles are reported separately.
				if item.Cash == 0 {
					kind = 1
				}
				rewards = append(rewards, rewardObservation{item: item, level: w.Level.Number, kind: kind})
				seenRewards[w.Level.Number][kind]++
			}
			for i := range rewards {
				r := &rewards[i]
				if r.counted || r.item.Active {
					continue
				}
				r.counted = true
				// Original cash/pickup animations loop or hold. A known live
				// bubble retires through collection, Y=200 expiry or eviction.
				if r.item.Binding.EntityID != 0 && r.item.Motion.Y < 200 {
					collectedRewards[r.level][r.kind]++
				} else {
					missedRewards[r.level][r.kind]++
				}
			}
		}
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
				observeRewards(before)
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
			observeRewards(w)
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
					t.Logf("EXPERT_OBSERVED_BUBBLES explicit%v level%d cashSeen%d cashCollected%d cashMissed%d equipmentSeen%d equipmentCollected%d equipmentMissed%d", explicit, level,
						seenRewards[level][0], collectedRewards[level][0], missedRewards[level][0], seenRewards[level][1], collectedRewards[level][1], missedRewards[level][1])
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
