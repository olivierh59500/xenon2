package engine

import "fmt"

// The carrier payload uses this original nineteen-entry reward catalogue,
// independently of the twenty-five entries offered by the shop.
var carrierItems = [19]Item{
	ItemSpeedup, ItemAutofire, ItemHealth1, ItemHealth2, ItemRearShot,
	ItemSideShot, ItemPowerup, ItemCannon, ItemDrone, ItemLaser,
	ItemElectroBall, ItemMineSmall, ItemMissileLauncher, ItemHomingMissile,
	ItemFlamer, ItemBomb, ItemNone, ItemDive, ItemNone,
}

func (w *World) spawnWaveCash(x, y int, heavy bool) {
	name := "cash-small"
	if heavy {
		name = "cash-large"
	}
	animation, ok := w.commonAnimations[name]
	if !ok {
		return
	}
	tag := int16(80)
	if heavy {
		tag = 84
	}
	binding, err := w.reserveWorldActor(tag, ActorPoolProjectile, false)
	if err != nil {
		w.poolError = err
		return
	}
	p := &WorldCollectible{ID: binding.EntityID, Binding: binding, Cash: CashValue(heavy), X: float64(x), Y: float64(y), PreviousX: float64(x), PreviousY: float64(y), Active: true,
		Motion: CashMotion{X: x, Y: y, Mode: 7, Direction: uint8(w.random.Next() & 7)}, animation: animation, animationState: NewAnimation(animation.Animation)}
	p.Sprite = p.animationState.Sprite(animation.Animation)
	w.poolCollectibles[binding.Slot] = p
	w.Collectibles = append([]*WorldCollectible{p}, w.Collectibles...)
}

// spawnExitCash creates the original alternating small-head and large-tail
// rewards. Cash insertion order differs from ordinary projectile creation.
func (w *World) spawnExitCash(pairs int) {
	for range pairs {
		for _, heavy := range []bool{false, true} {
			name := "cash-small"
			if heavy {
				name = "cash-large"
			}
			animation, ok := w.commonAnimations[name]
			if !ok {
				continue
			}
			x, _ := w.random.Below(300)
			y, _ := w.random.Below(180)
			tag := int16(80)
			if heavy {
				tag = 84
			}
			binding, err := w.reserveWorldActor(tag, ActorPoolProjectile, heavy)
			if err != nil {
				w.poolError = err
				return
			}
			p := &WorldCollectible{ID: binding.EntityID, Binding: binding, Cash: CashValue(heavy), X: float64(x + 10), Y: float64(y + 6), Active: true,
				Motion: CashMotion{X: int(x) + 10, Y: int(y) + 6, Mode: 7}, animation: animation, animationState: NewAnimation(animation.Animation)}
			p.PreviousX, p.PreviousY = p.X, p.Y
			p.Sprite = p.animationState.Sprite(animation.Animation)
			w.poolCollectibles[binding.Slot] = p
			w.PendingExitDrops++
			if heavy {
				w.nextTailOrder--
				p.Order = w.nextTailOrder
				w.Collectibles = append(w.Collectibles, p)
			} else {
				w.Collectibles = append([]*WorldCollectible{p}, w.Collectibles...)
			}
		}
	}
}

func (w *World) spawnPickup(reward, x, y int) {
	if reward < 0 || reward >= len(carrierItems) {
		return
	}
	animation, ok := w.commonAnimations[fmt.Sprintf("pickup-%d", reward)]
	if !ok {
		return
	}
	binding, err := w.reserveWorldActor(int16(animation.ResourceTag), ActorPoolProjectile, false)
	if err != nil {
		w.poolError = err
		return
	}
	p := &WorldCollectible{ID: binding.EntityID, Binding: binding, Reward: reward, X: float64(x), Y: float64(y), PreviousX: float64(x), PreviousY: float64(y), Active: true,
		Motion: CashMotion{X: x, Y: y, Mode: 7}, animation: animation, animationState: NewAnimation(animation.Animation)}
	p.Sprite = p.animationState.Sprite(animation.Animation)
	w.poolCollectibles[binding.Slot] = p
	w.Collectibles = append([]*WorldCollectible{p}, w.Collectibles...)
}

func (w *World) advanceCollectible(p *WorldCollectible) {
	if !p.Active {
		return
	}
	p.PreviousX, p.PreviousY = p.X, p.Y
	if !p.animationState.AdvanceNamed(p.animation) {
		p.Active = false
		return
	}
	p.Active = p.Motion.Advance()
	if !p.Active {
		w.consumeExitDrop()
	}
	p.X, p.Y = float64(p.Motion.X), float64(p.Motion.Y)
	p.Sprite = p.animationState.Sprite(p.animation.Animation)
	if !p.Active || !w.PlayerAlive || w.Dive.Phase != 0 {
		return
	}
	box, ok := w.movingSpriteBoxes[p.Sprite]
	if !ok || !ActorCollisionRect(box, p.Motion.X, p.Motion.Y).Intersects(w.playerCollision) {
		return
	}
	p.Active = false
	if p.Cash > 0 {
		w.Money += p.Cash
		w.SoundRequests[2] = "synthesized-effect-05"
		w.consumeExitDrop()
		return
	}
	if p.animation.SoundEffect != "" {
		w.SoundRequests[2] = p.animation.SoundEffect
	}
	w.applyCarrierReward(p.Reward)
}

func (w *World) consumeExitDrop() {
	if w.PendingExitDrops > 0 {
		w.PendingExitDrops--
		if w.PendingExitDrops == 0 {
			w.ShopReady = true
			w.ExitReady = w.LevelFinished
		}
	}
}

func (w *World) applyCarrierReward(reward int) {
	if reward < 0 || reward >= len(carrierItems) {
		return
	}
	switch reward {
	case 16:
		w.InvulnerableFrames += 170
	case 18:
		w.ScreenClearFrames = 31
		w.ScreenClearPaletteMask = uint16(w.random.Next())
		w.SoundRequests[2] = "synthesized-effect-02"
	default:
		w.Equipment.ApplyItem(carrierItems[reward])
	}
}

// AdvancePALTick advances display-timed effects at fifty ticks per second.
// The original palette strobe pauses ordinary gameplay for thirty-one VBLs.
func (w *World) AdvancePALTick() {
	if w.ScreenClearFrames == 0 {
		return
	}
	w.ScreenClearFrames--
	if w.ScreenClearFrames != 0 {
		w.ScreenClearPaletteMask = uint16(w.random.Next())
		return
	}
	w.ScreenClearPaletteMask = 0
	// The original supernova invokes each eligible enemy's own damage
	// callback. Carriers can release further equipment during the blast.
	for _, actor := range w.Actors {
		if actor.Active && actor.ActorList == "moving" && actor.part.ResourceTag != 0x50 && actor.part.ResourceTag != 0x54 {
			w.damageActor(actor, 127)
		}
	}
	for _, projectile := range w.Projectiles {
		projectile.Active = false
	}
}
