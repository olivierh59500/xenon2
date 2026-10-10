package app

import (
	"fmt"
	"os"
	"testing"

	"xenon2/internal/engine"
)

// This diagnostic runs the actual frontend/merchant journey. It records the
// first damaged third-stage ship without granting equipment or changing input.
func TestExpertThirdStageDamageTraceOptional(t *testing.T) {
	if os.Getenv("XENON2_EXPERT_DAMAGE_TRACE") == "" {
		t.Skip("enable the actual expert damage trace explicitly")
	}
	g, err := NewConfiguredGame(frontendGame(t).Bundle, Config{Level: 1, StartScreen: PresentationScreen, Mute: true, Demo: true, HumanDemo: true})
	if err != nil {
		t.Fatal(err)
	}
	var previous *engine.World
	lastShop, third := false, false
	for update := 0; update < 60*2400; update++ {
		var before *engine.World
		if d, ok := g.Driver.(*worldDriver); ok {
			before = d.world
		}
		oldHP, oldFrame, camera := 0, uint64(0), 0
		var player engine.PlayerMotionState
		var prefix engine.CollisionRect
		var actors [engine.ActorPoolCapacity]*engine.WorldActor
		var shots [engine.ActorPoolCapacity]*engine.WorldProjectile
		var bounds [engine.ActorPoolCapacity]engine.CollisionRect
		ac, sc := 0, 0
		if before != nil {
			oldHP, oldFrame, camera, player = before.Equipment.Shield, before.Frame, before.ScrollY, before.Player
			if before.Level.Number == 3 {
				for _, sprite := range g.Bundle.Ships.Atlas.Sprites {
					if sprite.Name == fmt.Sprintf("player-ship-%d", player.BankFrame()) && sprite.Collision != nil {
						prefix = engine.ActorCollisionRect(*sprite.Collision, player.X, player.Y)
					}
				}
				for _, actor := range before.Actors {
					if actor.Active && ac < len(actors) {
						actors[ac], bounds[ac] = actor, actor.Collision
						ac++
					}
				}
				for _, shot := range before.Projectiles {
					if shot.Active && sc < len(shots) {
						shots[sc] = shot
						sc++
					}
				}
			}
		}
		advanceFrontend(t, g, inputFrame{})
		d, ok := g.Driver.(*worldDriver)
		if !ok || d.session == nil {
			continue
		}
		w := d.world
		if w != previous {
			t.Logf("ENTRY L%d F%d C%d RNG%+v equipment%+v cash%d score%d", w.Level.Number, w.Frame, w.ScrollY, w.RandomState(), w.Equipment, w.Money, w.Score)
			previous = w
		}
		shop := g.Screen == ShopScreen
		if shop && !lastShop {
			t.Logf("SHOP L%d final%v F%d equipment%+v cash%d", w.Level.Number, g.shopFinal, w.Frame, w.Equipment, w.Money)
		}
		lastShop = shop
		third = third || w.Level.Number == 3
		if w == before && w.Level.Number == 3 && w.Frame != oldFrame && w.Equipment.Shield < oldHP {
			var motion engine.MotionInput
			fire, dive := false, false
			if g.demo != nil {
				motion, fire, dive = g.demo.controls.gameMotion, g.demo.controls.fire, g.demo.controls.divePressed
			}
			t.Logf("DAMAGE F%d C%d->%d P%+v->%+v HP%d->%d prefix%+v motion%+v fire%v dive%v rewind%+v", oldFrame, camera, w.ScrollY, player, w.Player, oldHP, w.Equipment.Shield, prefix, motion, fire, dive, w.Rewind)
			for _, actor := range w.Actors {
				if actor.Active && !actor.Visible && actor.Sprite == "" && actor.Health > 0 && !actor.Collision.Empty() && actor.Collision.Top >= -64 && actor.Collision.Bottom <= 256 {
					t.Logf("LIVE_TERRAIN_TARGET id%d xy%.0f,%.0f HP%d bounds%+v", actor.ID, actor.X, actor.Y, actor.Health, actor.Collision)
				}
			}
			for i, actor := range actors[:ac] {
				if bounds[i].Intersects(prefix) || !bounds[i].Empty() && bounds[i].Left <= player.X+36 && bounds[i].Right >= player.X-36 && bounds[i].Top <= player.Y+40 && bounds[i].Bottom >= player.Y-40 {
					t.Logf("NEAR_ACTOR id%d sprite%s atlas%s health%d before%+v after%+v", actor.ID, actor.Sprite, actor.Atlas, actor.Health, bounds[i], actor.Collision)
				}
			}
			for _, shot := range shots[:sc] {
				if shot.X >= float64(prefix.Left-12) && shot.X <= float64(prefix.Right+12) && shot.Y >= float64(prefix.Top-12) && shot.Y <= float64(prefix.Bottom+12) {
					t.Logf("NEAR_SHOT id%d sprite%s active%v pos%.0f,%.0f dir%d speed%d", shot.ID, shot.Sprite, shot.Active, shot.X, shot.Y, shot.Motion.Direction, shot.Motion.Speed)
				}
			}
		}
		if third && !w.PlayerAlive {
			t.Logf("FIRST_THIRD_LOSS F%d C%d RNG%+v equipment%+v", w.Frame, w.ScrollY, w.RandomState(), w.Equipment)
			return
		}
		if third && !g.DemoActive() && g.Screen == TitleScreen {
			t.Logf("COMPLETE_WITHOUT_THIRD_LOSS F%d equipment%+v", w.Frame, w.Equipment)
			return
		}
	}
	t.Fatal("bounded expert damage trace did not reach its completion boundary")
}
