package app

import "xenon2/internal/engine"

// worldDriver translates snapshots without owning or approximating game rules.
type worldDriver struct {
	world   *engine.World
	sprites []SpriteView
}

func newWorldDriver(bundle *Bundle, level int) (*worldDriver, error) {
	l := bundle.Levels[level-1]
	world, err := engine.NewWorld(engine.LevelData{Number: level, Terrain: &l.Terrain, Paths: &l.Paths, Encounters: &l.Encounters, Actors: &l.Actors, FixedSprites: &l.FixedSprites, FixedTiles: &l.FixedTiles, PlayerStencil: &bundle.Stencil, Rules: &l.Rules, Ships: &bundle.Ships})
	if err != nil {
		return nil, err
	}
	return &worldDriver{world: world}, nil
}

func (d *worldDriver) Advance(input Input) error {
	return d.world.Step(engine.Input{Motion: input.Motion, Fire: input.Fire, Dive: input.DivePressed})
}

func (d *worldDriver) Frame() SceneFrame {
	w := d.world
	d.sprites = d.sprites[:0]
	for _, actor := range w.Actors {
		if actor.Active && actor.Visible && actor.Sprite != "" {
			d.sprites = append(d.sprites, SpriteView{ID: actor.ID, Atlas: actor.Atlas, Sprite: actor.Sprite, X: actor.X, Y: actor.Y, PreviousX: actor.PreviousX, PreviousY: actor.PreviousY, Interpolate: true})
		}
	}
	for _, shot := range w.Projectiles {
		if shot.Active {
			d.sprites = append(d.sprites, SpriteView{ID: shot.ID, Atlas: shot.Atlas, Sprite: shot.Sprite, X: shot.X, Y: shot.Y, PreviousX: shot.PreviousX, PreviousY: shot.PreviousY, Interpolate: true})
		}
	}
	for _, shot := range w.SmallShots {
		if shot.Active {
			d.sprites = append(d.sprites, SpriteView{ID: shot.ID, Atlas: "common", Sprite: shot.Shot.SpriteName, X: float64(shot.Shot.X), Y: float64(shot.Shot.Y), PreviousX: float64(shot.PreviousX), PreviousY: float64(shot.PreviousY), Interpolate: true})
		}
	}
	return SceneFrame{Level: w.Level.Number, CameraY: float64(w.RenderScrollY), BackgroundY: float64(w.BackgroundY), Player: w.Player, PlayerAlive: w.PlayerAlive, Sprites: d.sprites, TerrainMap: w.Level.Terrain.Map, Score: w.Score, Money: w.Money, Shield: w.Equipment.Shield, Lives: w.Equipment.Lives, Diagnostic: true}
}

func (g *Game) StartLevel(level int) error {
	if level < 1 || level > len(g.Bundle.Levels) {
		return nil
	}
	driver, err := newWorldDriver(g.Bundle, level)
	if err != nil {
		return err
	}
	g.SetDriver(driver)
	g.clock = engine.NewFrameClock(25, 60)
	return nil
}
