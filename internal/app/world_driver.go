package app

import (
	"fmt"
	"slices"
	"xenon2/internal/engine"
)

// worldDriver translates snapshots without owning or approximating game rules.
type worldDriver struct {
	diagnostic  bool
	drawOrder   map[int]int
	effects     []SpriteView
	sparks      []SpriteView
	weapons     []engine.WeaponRenderItem
	session     *engine.Session
	turnChanged bool
	world       *engine.World
	sprites     []SpriteView
	hud         []SpriteView
}

func newWorldDriver(bundle *Bundle, level int) (*worldDriver, error) {
	l := bundle.Levels[level-1]
	world, err := engine.NewWorld(engine.LevelData{Number: level, Terrain: &l.Terrain, Paths: &l.Paths, Encounters: &l.Encounters, Actors: &l.Actors, FixedSprites: &l.FixedSprites, FixedTiles: &l.FixedTiles, PlayerStencil: &bundle.Stencil, Rules: &l.Rules, Ships: &bundle.Ships, Common: &bundle.Common, Guardians: l.Guardians, GuardianGroups: l.GuardianGroups, GuardianParts: l.GuardianParts})
	if err != nil {
		return nil, err
	}
	world.ResetBackgroundStars()
	return &worldDriver{world: world, diagnostic: true}, nil
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
	d.drawOrder = w.ActorDrawOrder(d.drawOrder)
	d.sprites = d.sprites[:0]
	for index, shadow := range w.Shadows {
		if shadow.Visible {
			d.sprites = append(d.sprites, SpriteView{ID: -2001 - index, Layer: "shadows", Atlas: "common", Sprite: fmt.Sprintf("player-shadow-%d", shadow.Counter), X: float64(shadow.X), Y: float64(shadow.Y), PreviousX: float64(shadow.PreviousX), PreviousY: float64(shadow.PreviousY), Interpolate: true})
		}
	}
	for _, actor := range w.Actors {
		if actor.Active && actor.Visible && (actor.Sprite != "" || actor.Patch != nil || actor.DrawKind != "") {
			layer := actor.ActorList
			if layer == "transient" {
				layer = "effects"
			}
			order := actor.Order
			if order == 0 {
				order = actor.ID
			}
			view := SpriteView{ID: actor.ID, Order: order, Layer: layer, Atlas: actor.Atlas, Sprite: actor.Sprite, X: actor.X, Y: actor.Y, PreviousX: actor.PreviousX, PreviousY: actor.PreviousY, Interpolate: true, Flash: actor.Flash, Materializing: actor.Materializing}
			if actor.DrawKind != "" {
				view.Kind = actor.DrawKind
				view.Length = actor.DrawLength
				if actor.DrawUp {
					view.Tier = 1
				}
			}
			if actor.Patch != nil {
				view.Kind = "tiles"
				view.Patch = *actor.Patch
			}
			if actor.DrawKind != "assembly" {
				d.sprites = append(d.sprites, view)
			}
			for _, overlay := range actor.TileOverlays {
				d.sprites = append(d.sprites, SpriteView{ID: actor.ID, Layer: layer, Kind: "tiles", Patch: overlay.Patch, X: overlay.X, Y: overlay.Y, PreviousX: overlay.PreviousX, PreviousY: overlay.PreviousY, Interpolate: overlay.Interpolate})
			}
			for _, extra := range actor.Extras {
				d.sprites = append(d.sprites, SpriteView{ID: actor.ID, Layer: layer, Atlas: extra.Atlas, Sprite: extra.Sprite, X: extra.X, Y: extra.Y, PreviousX: extra.PreviousX, PreviousY: extra.PreviousY, Interpolate: extra.Interpolate})
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
	for i := range d.sprites {
		if order, ok := d.drawOrder[d.sprites[i].ID]; ok {
			d.sprites[i].Order = order
		}
	}
	// Compound bodies use physical moving-list order even when their storage
	// slices were appended during construction.
	slices.SortStableFunc(d.sprites, func(a, b SpriteView) int {
		rank := func(layer string) int {
			switch layer {
			case "shadows":
				return 0
			case "equipment":
				return 1
			case "moving":
				return 2
			case "effects":
				return 3
			case "scenery":
				return 4
			case "sparks":
				return 5
			}
			return 6
		}
		if a.Layer != b.Layer {
			return rank(a.Layer) - rank(b.Layer)
		}
		if a.Layer != "sparks" && a.Layer != "shadows" {
			return b.Order - a.Order
		}
		return 0
	})
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
	d.hud = d.hud[:0]
	for _, item := range w.HUD {
		d.hud = append(d.hud, SpriteView{Atlas: item.Atlas, Sprite: item.Sprite, X: item.X, Y: item.Y})
	}
	frame := SceneFrame{BackgroundStars: w.BackgroundStars, ContinueCredits: w.ContinueCredits, DivePhase: w.RenderDivePhase, Level: w.Level.Number, CameraY: float64(w.RenderScrollY), BackgroundY: float64((192 - w.BackgroundY) % 192), Player: w.Player, PlayerAlive: w.PlayerAlive, PlayerSprite: w.PlayerSprite, Ready: w.Ready, GameOver: w.GameOver, Sprites: d.sprites, TerrainMap: w.RenderTerrainMap, Score: w.Score, Money: w.Money, Shield: w.Equipment.Shield, Lives: w.Equipment.Lives, Diagnostic: d.diagnostic, FreezeInterpolation: w.ScreenClearFrames > 0, PaletteMask: w.ScreenClearPaletteMask, Shades: w.Equipment.ShadesFrames > 0}
	frame.PlayerNumber, frame.PlayerCount = 1, 1
	frame.ActorTerrainMap, frame.ActorCameraY = w.ActorRenderTerrainMap, float64(w.ActorRenderScrollY)
	frame.HUD = d.hud
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
		accepted := false
		if driver.session != nil {
			accepted = driver.session.AcceptContinue()
			driver.world = driver.session.ActiveWorld()
		} else {
			accepted = driver.world.AcceptContinue()
		}
		if !accepted {
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
		if !driver.session.AcceptContinue() {
			return fmt.Errorf("continue unavailable")
		}
		driver.world = driver.session.ActiveWorld()
		g.View = driver.Frame()
		g.rememberFrameHistory()
		return nil
	}
	return nil
}
