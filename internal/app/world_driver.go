package app

import (
	"fmt"
	"slices"
	"xenon2/internal/engine"
)

// worldDriver translates snapshots without owning or approximating game rules.
type worldDriver struct {
	effects     []SpriteView
	sparks      []SpriteView
	weapons     []engine.WeaponRenderItem
	session     *engine.Session
	turnChanged bool
	world       *engine.World
	sprites     []SpriteView
}

func newWorldDriver(bundle *Bundle, level int) (*worldDriver, error) {
	l := bundle.Levels[level-1]
	world, err := engine.NewWorld(engine.LevelData{Number: level, Terrain: &l.Terrain, Paths: &l.Paths, Encounters: &l.Encounters, Actors: &l.Actors, FixedSprites: &l.FixedSprites, FixedTiles: &l.FixedTiles, PlayerStencil: &bundle.Stencil, Rules: &l.Rules, Ships: &bundle.Ships, Common: &bundle.Common, Guardians: l.Guardians, GuardianGroups: l.GuardianGroups, GuardianParts: l.GuardianParts})
	if err != nil {
		return nil, err
	}
	return &worldDriver{world: world}, nil
}

func (d *worldDriver) SetEffectActivity(active [4]bool) { d.world.EffectActive = active }

func (d *worldDriver) ConsumeEffectStop() bool {
	stop := d.world.StopEffectsRequested
	d.world.StopEffectsRequested = false
	return stop
}

func (d *worldDriver) Advance(input Input) error {
	if d.session != nil {
		changed, err := d.session.Advance(engine.Input{Motion: input.Motion, Fire: input.Fire, Dive: input.DivePressed})
		d.turnChanged = d.turnChanged || changed
		d.world = d.session.ActiveWorld()
		return err
	}
	return d.world.Step(engine.Input{Motion: input.Motion, Fire: input.Fire, Dive: input.DivePressed})
}

func (d *worldDriver) ConsumeTurnChange() bool {
	changed := d.turnChanged
	d.turnChanged = false
	return changed
}

func (d *worldDriver) ConsumeImmediateSounds() [4]string {
	requests := d.world.ImmediateSoundRequests
	d.world.ImmediateSoundRequests = [4]string{}
	return requests
}

func (d *worldDriver) ConsumeSoundRequests() [4]string {
	requests := d.world.SoundRequests
	d.world.SoundRequests = [4]string{}
	return requests
}
func (d *worldDriver) AdvancePALTick() { d.world.AdvancePALTick() }

func (d *worldDriver) Frame() SceneFrame {
	w := d.world
	d.sprites = d.sprites[:0]
	for _, actor := range w.Actors {
		if actor.Active && actor.Visible && (actor.Sprite != "" || actor.Patch != nil) {
			layer := actor.ActorList
			if layer == "transient" {
				layer = "effects"
			}
			order := actor.Order
			if order == 0 {
				order = actor.ID
			}
			view := SpriteView{ID: actor.ID, Order: order, Layer: layer, Atlas: actor.Atlas, Sprite: actor.Sprite, X: actor.X, Y: actor.Y, PreviousX: actor.PreviousX, PreviousY: actor.PreviousY, Interpolate: true, Flash: actor.Flash, Materializing: actor.Materializing}
			if actor.Patch != nil {
				view.Kind = "tiles"
				view.Patch = *actor.Patch
			}
			d.sprites = append(d.sprites, view)
			for _, extra := range actor.Extras {
				d.sprites = append(d.sprites, SpriteView{ID: actor.ID, Layer: layer, Atlas: extra.Atlas, Sprite: extra.Sprite, X: extra.X, Y: extra.Y})
			}
		}
	}
	for _, shot := range w.Projectiles {
		if shot.Active {
			d.sprites = append(d.sprites, SpriteView{ID: shot.ID, Order: shot.ID, Layer: "effects", Atlas: shot.Atlas, Sprite: shot.Sprite, X: shot.X, Y: shot.Y, PreviousX: shot.PreviousX, PreviousY: shot.PreviousY, Interpolate: true})
		}
	}
	for _, shot := range w.SmallShots {
		if shot.Active {
			d.sprites = append(d.sprites, SpriteView{ID: shot.ID, Order: shot.ID, Layer: "effects", Atlas: "common", Sprite: shot.Shot.SpriteName, X: float64(shot.Shot.X), Y: float64(shot.Shot.Y), PreviousX: float64(shot.PreviousX), PreviousY: float64(shot.PreviousY), Interpolate: true})
		}
	}
	for _, item := range w.Collectibles {
		if item.Active {
			order := item.Order
			if order == 0 {
				order = item.ID
			}
			d.sprites = append(d.sprites, SpriteView{ID: item.ID, Order: order, Layer: "effects", Atlas: "common", Sprite: item.Sprite, X: item.X, Y: item.Y, PreviousX: item.PreviousX, PreviousY: item.PreviousY, Interpolate: true})
		}
	}
	if w.Weapons != nil {
		d.weapons = w.Weapons.RenderState(d.weapons[:0])
		for _, item := range d.weapons {
			if item.Active {
				layer := "effects"
				if item.Kind == "attachment" {
					layer = "equipment"
				} else if item.Kind == "spark" {
					layer = "sparks"
				}
				d.sprites = append(d.sprites, SpriteView{ID: item.ID, Order: item.ID, Kind: item.Kind, Layer: layer, Atlas: "common", Sprite: item.Sprite, X: item.X, Y: item.Y, PreviousX: item.PreviousX, PreviousY: item.PreviousY, Interpolate: true, Tier: item.Tier, Length: item.Length})
			}
		}
	}
	d.effects = d.effects[:0]
	d.sparks = d.sparks[:0]
	kept := d.sprites[:0]
	for _, sprite := range d.sprites {
		if sprite.Layer == "effects" {
			d.effects = append(d.effects, sprite)
		} else if sprite.Layer == "sparks" {
			d.sparks = append(d.sparks, sprite)
		} else {
			kept = append(kept, sprite)
		}
	}
	d.sprites = kept
	slices.SortFunc(d.effects, func(a, b SpriteView) int {
		if a.Order > b.Order {
			return -1
		}
		if a.Order < b.Order {
			return 1
		}
		return 0
	})
	d.sprites = append(d.sprites, d.effects...)
	slices.SortFunc(d.sparks, func(a, b SpriteView) int { return b.ID - a.ID })
	d.sprites = append(d.sprites, d.sparks...)
	frame := SceneFrame{ContinueCredits: w.ContinueCredits, DivePhase: w.RenderDivePhase, Level: w.Level.Number, CameraY: float64(w.RenderScrollY), BackgroundY: float64((192 - w.BackgroundY) % 192), Player: w.Player, PlayerAlive: w.PlayerAlive, PlayerSprite: w.PlayerSprite, Ready: w.Ready, GameOver: w.GameOver, Sprites: d.sprites, TerrainMap: w.Level.Terrain.Map, Score: w.Score, Money: w.Money, Shield: w.Equipment.Shield, Lives: w.Equipment.Lives, Diagnostic: true, FreezeInterpolation: w.ScreenClearFrames > 0, PaletteMask: w.ScreenClearPaletteMask, Shades: w.Equipment.ShadesFrames > 0}
	frame.PlayerNumber, frame.PlayerCount = 1, 1
	frame.PlayerScores[0] = w.DisplayScore
	frame.PlayerLives[0] = w.Equipment.Lives
	frame.PlayerShields[0] = w.Equipment.Shield
	if d.session != nil {
		frame.PlayerNumber = d.session.Current + 1
		frame.PlayerCount = d.session.PlayerCount
		for i, player := range d.session.Players {
			if player != nil {
				frame.PlayerScores[i] = player.DisplayScore
				frame.PlayerLives[i] = player.Equipment.Lives
				frame.PlayerShields[i] = player.Equipment.Shield
			}
		}
	}
	return frame
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
	if g.starfield != nil && g.Screen == TitleScreen {
		driver.world.SetRandomState(*g.starfield.Random)
	}
	g.onContinue = func() error {
		if !driver.world.AcceptContinue() {
			return fmt.Errorf("continue unavailable")
		}
		g.View = driver.Frame()
		g.rememberFrameHistory()
		return nil
	}
	g.clock = engine.NewFrameClock(25, 60)
	g.palClock = engine.NewFrameClock(50, 60)
	return nil
}

func (g *Game) StartSession(level, players int) error {
	l := g.Bundle.Levels[level-1]
	data := engine.LevelData{Number: level, Terrain: &l.Terrain, Paths: &l.Paths, Encounters: &l.Encounters, Actors: &l.Actors, FixedSprites: &l.FixedSprites, FixedTiles: &l.FixedTiles, PlayerStencil: &g.Bundle.Stencil, Rules: &l.Rules, Ships: &g.Bundle.Ships, Common: &g.Bundle.Common, Guardians: l.Guardians, GuardianGroups: l.GuardianGroups, GuardianParts: l.GuardianParts}
	session, err := engine.NewSession(data, players, *g.starfield.Random)
	if err != nil {
		return err
	}
	driver := &worldDriver{world: session.ActiveWorld(), session: session}
	g.SetDriver(driver)
	g.clock = engine.NewFrameClock(25, 60)
	g.palClock = engine.NewFrameClock(50, 60)
	g.readyRunning, g.gameOverRunning = false, false
	g.onContinue = func() error {
		if !driver.world.AcceptContinue() {
			return fmt.Errorf("continue unavailable")
		}
		g.View = driver.Frame()
		g.rememberFrameHistory()
		return nil
	}
	return nil
}
