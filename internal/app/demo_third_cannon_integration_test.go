package app

import (
	"os"
	"testing"
	"xenon2/internal/engine"
)

func TestDemoDestroysThirdBlockingCannonFromCompleteIntroOptional(t *testing.T) {
	if os.Getenv("XENON2_DEMO_PROGRESS_CHECK") == "" {
		t.Skip("enable the connected cannon route explicitly")
	}
	g := verifyDemoFirstTwoLevelsFromStart(t, 3, true)
	middleMerchant := false
	var cannon *engine.WorldActor
	for update := 0; update < 60*600; update++ {
		advanceFrontend(t, g, inputFrame{})
		d := g.Driver.(*worldDriver)
		w := d.world
		if w.Level.Number != 3 || w.Cheats.Enabled() || d.diagnostic || !g.DemoActive() {
			t.Fatal("cannon approach left the ordinary carried third-stage game")
		}
		if g.Screen == ShopScreen && !g.shopFinal {
			if w.ThirdMiddle == nil || !w.ThirdMiddle.Defeated || w.ThirdMiddle.EyeHealth != [2]uint16{} || w.PendingExitDrops != 0 {
				t.Fatal("middle merchant bypassed guardian defeat or reward collection")
			}
			middleMerchant = true
		}
		if cannon == nil {
			for _, actor := range w.Actors {
				worldY := int(actor.Y) + w.ActorRenderScrollY
				if actor.Active && actor.Binding.EntityID != 0 && w.Pool.Slot(actor.Binding.Slot).ResourceTag == 248 && int(actor.X) == 224 && worldY >= 1695 && worldY <= 1697 {
					cannon = actor
					break
				}
			}
		}
		if cannon != nil && !cannon.Active && cannon.Health == 0 {
			if !middleMerchant || !w.PlayerAlive || w.Equipment.Lives < 1 || w.ShopReady || w.LevelFinished {
				t.Fatal("cannon defeat bypassed its carried route or exhausted the player")
			}
			for row := 106; row < 112; row++ {
				for column := 14; column < 18; column++ {
					if w.Level.Terrain.Map[row*20+column] != 0 {
						t.Fatal("ordinary cannon defeat did not clear its original footprint")
					}
				}
			}
			t.Logf("Connected third blocker destroyed at frame%d camera%d: ships%d shield%d credits%d", w.Frame, w.ScrollY, w.Equipment.Lives, w.Equipment.Shield, w.ContinueCredits)
			return
		}
		if w.GameOver && w.ContinueCredits == 0 {
			t.Fatal("ordinary recovery exhausted before the blocking cannon was destroyed")
		}
	}
	t.Fatal("bounded carried route did not destroy the blocking cannon")
}
