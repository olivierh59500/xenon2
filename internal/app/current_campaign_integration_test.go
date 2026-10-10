package app

import (
	"testing"

	"xenon2/internal/engine"
)

// This runs from the ordinary intro with a test-only five-stage tour limit.
// The driver/session, native guardians, coins, merchants and loader own every
// transition. No equipment, RNG, health or completion state is assigned here.
func verifyCurrentFourStageCampaign(t *testing.T) *Game {
	t.Helper()
	g := presentationFrontendGame(t)
	var middle, final [6]bool
	var previous *engine.World
	lastLevel := 0
	fourthFinalAdmission, fourthRearUpgrade := false, false
	for update := 0; update < 60*2400; update++ {
		advanceFrontend(t, g, inputFrame{})
		d, ok := g.Driver.(*worldDriver)
		if !ok || d.world == nil {
			continue
		}
		w := d.world
		if d.diagnostic || w.Cheats.Enabled() || !g.DemoActive() || !w.PlayerAlive || w.GameOver || w.Equipment.Lives != 3 || w.ContinueCredits != 2 {
			t.Fatalf("current campaign lost an earned reserve: level %d frame %d shield %d lives %d credits %d", w.Level.Number, w.Frame, w.Equipment.Shield, w.Equipment.Lives, w.ContinueCredits)
		}
		if w.Level.Number != lastLevel {
			if lastLevel != 0 && (!middle[lastLevel] || !final[lastLevel]) {
				t.Fatalf("stage %d omitted a genuine merchant", lastLevel)
			}
			if w.Level.Number == 4 {
				if w.Frame != 0 || w.Score != 163850 || w.Equipment.Shield != 39 || w.Equipment.Primary.Item != engine.ItemForwardShot || w.Equipment.Primary.Tier != 1 || w.Equipment.Mounts[0].Item != engine.ItemCannon || w.Equipment.Rear.Item != engine.ItemRearShot || w.Equipment.Rear.Tier != 1 {
					t.Fatal("current three-stage source admission changed")
				}
			}
			if w.Level.Number == 5 {
				if previous == nil || previous.FourthMiddle == nil || !previous.FourthMiddle.Defeated || previous.FourthMiddle.OuterTargets != 0 || int16(previous.FourthMiddle.Parts[4].Health) > 0 || previous.FourthFinal == nil || !previous.FourthFinal.Defeated || previous.FourthFinal.EyesRemaining != 0 || int16(previous.FourthFinal.Parts[0].Health) > 0 || previous.PendingExitDrops != 0 || !fourthFinalAdmission || !fourthRearUpgrade {
					t.Fatal("fifth admission bypassed a real fourth-stage guardian or reward gate")
				}
				e := w.Equipment
				if w.Frame != 0 || w.Score != 203700 || w.Money != 0 || e.Shield != 39 || e.Primary.Item != engine.ItemForwardShot || e.Primary.Tier != 1 || e.Mounts[0].Item != engine.ItemLaser || e.Mounts[0].Tier != 0 || e.Rear.Item != engine.ItemRearShot || e.Rear.Tier != 2 || e.Side.Item != engine.ItemNone || w.RandomState() != (engine.RandomState{A: 1213999438, B: 2626127522}) {
					t.Fatalf("actual fifth READY admission changed: frame %d cash %d score %d RNG %+v equipment %+v", w.Frame, w.Money, w.Score, w.RandomState(), e)
				}
				t.Logf("Current intro completes four stages and enters fifth READY after %.2f simulated seconds: three ships, two continues, shield 39, score %d, earned Forward1/Laser0/Rear2.", float64(update+1)/60, w.Score)
				return g
			}
			lastLevel = w.Level.Number
		}
		previous = w
		if w.Level.Number == 4 && w.FourthFinal != nil && !fourthFinalAdmission {
			fourthFinalAdmission = true
			fourthRearUpgrade = w.Equipment.Rear.Item == engine.ItemRearShot && w.Equipment.Rear.Tier == 2
			if w.Frame != 6706 || w.ScrollY != 143 || w.Equipment.Shield != 39 || !fourthRearUpgrade {
				t.Fatal("current final admission lost its real repair or collected Rear2")
			}
		}
		if g.Screen != ShopScreen {
			continue
		}
		if !w.ShopReady || w.PendingExitDrops != 0 {
			t.Fatal("frontend entered a merchant before real reward drain")
		}
		if g.shopFinal {
			if !middle[lastLevel] || !w.LevelFinished || !w.ExitReady {
				t.Fatal("final merchant bypassed native stage completion")
			}
			if !final[lastLevel] && lastLevel == 4 {
				if w.Frame != 7747 || w.ScrollY != 0 || w.Equipment.Shield != 7 || w.Money != 4250 || w.FourthFinal == nil || !w.FourthFinal.Defeated || w.FourthFinal.EyesRemaining != 0 || int16(w.FourthFinal.Parts[0].Health) > 0 {
					t.Fatal("current fourth final merchant lost native damage or coins")
				}
				t.Logf("Real fourth final merchant: frame %d, shield %d, earned cash %d.", w.Frame, w.Equipment.Shield, w.Money)
			}
			final[lastLevel] = true
		} else {
			if w.LevelFinished {
				t.Fatal("intermediate merchant completed the level")
			}
			if !middle[lastLevel] && lastLevel == 4 {
				if w.Frame != 4692 || w.ScrollY != 2155 || w.Equipment.Shield != 27 || w.Money != 750 || w.FourthMiddle == nil || !w.FourthMiddle.Defeated || w.FourthMiddle.OuterTargets != 0 || int16(w.FourthMiddle.Parts[4].Health) > 0 {
					t.Fatal("current fourth intermediate merchant lost native damage or coins")
				}
				t.Logf("Real fourth intermediate merchant: frame %d, shield %d, earned cash %d.", w.Frame, w.Equipment.Shield, w.Money)
			}
			middle[lastLevel] = true
		}
	}
	w := g.Driver.(*worldDriver).world
	t.Fatalf("current four-stage journey did not reach fifth READY: level %d frame %d camera %d shield %d", w.Level.Number, w.Frame, w.ScrollY, w.Equipment.Shield)
	return nil
}
