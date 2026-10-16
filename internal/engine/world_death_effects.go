package engine

import "xenon2/internal/visualassets"

func (w *World) actorRegion(actor *WorldActor) visualassets.SpriteRegion {
	var atlas *visualassets.SpriteAtlas
	switch actor.Atlas {
	case "common":
		atlas = w.Level.Common
	case "fixed":
		if w.Level.FixedSprites != nil {
			atlas = &w.Level.FixedSprites.Atlas
		}
	case "guardians":
		if w.Level.Guardians != nil {
			atlas = &w.Level.Guardians.Atlas
		}
	case "guardian-parts":
		atlas = w.Level.GuardianParts
	default:
		if w.Level.Actors != nil {
			atlas = &w.Level.Actors.Atlas
		}
	}
	if atlas != nil {
		for _, region := range atlas.Sprites {
			if region.Name == actor.Sprite {
				return region
			}
		}
	}
	return visualassets.SpriteRegion{}
}

func (w *World) spawnActorDeathEffect(actor *WorldActor) {
	region := w.actorRegion(actor)
	name := "explosion-small"
	if actor.part != nil && actor.part.StrongHealth {
		name = "explosion-large"
	}
	w.spawnSecondNamedExplosion(int(actor.X)-region.AnchorX+region.Width/2, int(actor.Y)-region.AnchorY+(region.Height-1)/2, name)
}
