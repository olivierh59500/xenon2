package engine

import "xenon2/internal/visualassets"

func (w *World) spawnFourthCrawler(record visualassets.FixedEncounter) bool {
	if w.Level.Number != 4 || record.EnemyKind != 3 {
		return false
	}
	art := w.fixedKinds[3]
	if art == nil || art.Behavior != "fourth-crawler" {
		return false
	}
	state, err := NewFourthCrawler(record, *art)
	if err != nil {
		w.poolError = err
		return true
	}
	variant := &art.Variants[state.Variant]
	state.Visible = state.Y-w.ScrollY <= art.MotionParameters["draw_bottom"]
	actor := &WorldActor{Active: true, Visible: true, ActorList: "moving", Atlas: "fixed", X: float64(state.X), Y: float64(state.Y - w.ScrollY), Health: art.Health, Score: art.Score, fourthCrawler: &state, fourthCrawlerArt: art, Collision: state.Collision, part: &visualassets.ActorPart{ResourceTag: variant.ResourceTag, StrongHealth: true, MotionMode: "fourth-crawler", DamageMode: "fourth-crawler"}}
	actor.Sprite = state.Sprite(*art)
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	if err := w.bindWorldActor(actor); err != nil {
		w.poolError = err
		return true
	}
	actor.Binding.Residue.SetFireState(state.Fire.Accumulator, state.Fire.Rate)
	actor.Binding.Residue.X, actor.Binding.Residue.Y = int16(state.X), int16(state.Y)
	actor.Binding.Residue.VerticalFraction = uint16(state.Nest)
	w.storeWorldResidue(actor.Binding)
	w.updateFourthCrawlerCover(actor)
	w.Actors = append([]*WorldActor{actor}, w.Actors...)
	return true
}

func (w *World) advanceFourthCrawler(actor *WorldActor) error {
	state, art := actor.fourthCrawler, actor.fourthCrawlerArt
	_, err := state.Advance(*art, FourthCrawlerInput{ScrollY: w.ScrollY, MaximumScrollY: w.MaximumScrollY, PlayerX: w.Player.X, PlayerY: w.Player.Y}, func(name string) visualassets.CollisionBox { return w.movingSpriteBoxes[name] }, w.random.Next)
	if err != nil {
		return err
	}
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	actor.X, actor.Y = float64(state.X), float64(state.Y-w.ScrollY)
	actor.Sprite, actor.Active, actor.Visible, actor.Health, actor.Collision, actor.Flash = state.Sprite(*art), !state.Removed, state.Visible, int(state.Health), state.Collision, state.Flash
	w.updateFourthCrawlerCover(actor)
	w.storeFourthCrawlerResidue(actor)
	return nil
}

func (w *World) updateFourthCrawlerCover(actor *WorldActor) {
	state, art := actor.fourthCrawler, actor.fourthCrawlerArt
	actor.TileOverlays = actor.TileOverlays[:0]
	if !state.Visible || state.Phase >= 2 {
		return
	}
	variant := art.Variants[state.Variant]
	if variant.Cover != nil {
		actor.TileOverlays = append(actor.TileOverlays, WorldTileOverlay{Patch: *variant.Cover, X: float64(art.MotionTables["cover_x"][state.Nest]), Y: float64(art.MotionTables["cover_world_y"][state.Nest] - w.ScrollY)})
	}
}

func (w *World) storeFourthCrawlerResidue(actor *WorldActor) {
	if actor.Binding.EntityID == 0 {
		return
	}
	state := actor.fourthCrawler
	r := &actor.Binding.Residue
	r.X, r.Y = int16(state.X), int16(state.Y)
	r.Counter, r.VerticalVelocity, r.VerticalFraction = int16(state.Phase), int16(state.VerticalChoice), uint16(state.Nest)
	r.MountOffsetX, r.MountOffsetY = int16(state.HomeX), int16(state.HomeY)
	r.Health = state.Health
	r.SetFireState(state.Fire.Accumulator, state.Fire.Rate)
	w.storeWorldResidue(actor.Binding)
	if !actor.Active {
		w.retireWorldActor(actor.Binding)
	}
}

func (w *World) damageFourthCrawler(actor *WorldActor, amount uint16) {
	state, art := actor.fourthCrawler, actor.fourthCrawlerArt
	image := w.actorRegion(actor)
	event := state.Strike(*art, amount, w.ScrollY, image, nil)
	if !event.Destroyed {
		actor.Health, actor.Flash = int(state.Health), state.Flash
		w.storeFourthCrawlerResidue(actor)
		return
	}
	w.spawnSecondNamedExplosion(event.ExplosionX, event.ExplosionY, "explosion-large")
	w.Score += event.Score
	animation, ok := w.commonAnimations["cash-large"]
	if ok {
		binding, err := w.reserveWorldActor(24, ActorPoolProjectile, false)
		if err != nil {
			w.poolError = err
			return
		}
		direction := uint8(w.random.Next()) & 7
		coin := &WorldCollectible{ID: binding.EntityID, Binding: binding, Cash: 100, X: float64(event.CashX), Y: float64(event.CashY), Active: true, Motion: CashMotion{X: event.CashX, Y: event.CashY, Mode: 7, Direction: direction}, animation: animation, animationState: NewAnimation(animation.Animation)}
		coin.PreviousX, coin.PreviousY = coin.X, coin.Y
		coin.Sprite = coin.animationState.Sprite(animation.Animation)
		w.poolCollectibles[binding.Slot] = coin
		w.Collectibles = append([]*WorldCollectible{coin}, w.Collectibles...)
	}
	actor.X, actor.Y = float64(state.X), float64(state.Y-w.ScrollY)
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	actor.Sprite, actor.Health, actor.Collision, actor.Flash = state.Sprite(*art), int(state.Health), state.Collision, false
	w.updateFourthCrawlerCover(actor)
	w.storeFourthCrawlerResidue(actor)
}
