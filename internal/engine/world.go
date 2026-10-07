package engine

import (
	"fmt"

	"xenon2/internal/visualassets"
)

// LevelData contains decoded graphics and semantic game data only.
type LevelData struct {
	Number        int
	Terrain       *visualassets.Terrain
	Paths         *visualassets.Paths
	Encounters    *visualassets.Encounters
	Actors        *visualassets.Actors
	FixedSprites  *visualassets.FixedSprites
	FixedTiles    *visualassets.FixedTiles
	PlayerStencil *visualassets.PlayerTerrainStencil
	Rules         *visualassets.LevelRules
	Ships         *visualassets.ShipArt
}

type Input struct {
	Motion     MotionInput
	Fire, Dive bool
}

// WorldActor separates simulation positions from interpolated display state.
// Fixed actors retain map coordinates; moving actors retain their path state.
type WorldActor struct {
	ID                         int
	X, Y, PreviousX, PreviousY float64
	Sprite, Atlas              string
	Active                     bool
	Visible                    bool
	Health, Score              int
	part                       *visualassets.ActorPart
	animation                  visualassets.ActorAnimation
	animationState             AnimationState
	path                       *visualassets.Path
	motion                     PathMotionState
	leader                     *WorldActor
	mapY                       int
	fixed                      bool
	entrySelected              bool
	fire                       EnemyFireState
	Collision                  CollisionRect
}

type WorldProjectile struct {
	ID                         int
	X, Y, PreviousX, PreviousY float64
	Sprite, Atlas              string
	Active                     bool
	Motion                     DirectionalProjectile
}

type WorldSmallShot struct {
	ID                   int
	PreviousX, PreviousY int
	Shot                 SmallShot
	Active               bool
}

// World is the independent game simulation. Scenery mechanisms, weapons and
// stage transitions are integrated as their original rules are verified.
type World struct {
	Level                            LevelData
	Player, PreviousPlayer           PlayerMotionState
	ScrollY, PreviousScrollY         int
	RenderScrollY                    int
	ScrollDelta                      int
	BackgroundY, PreviousBackgroundY int
	VisitedScrollY                   int
	Equipment                        Equipment
	PlayerAlive                      bool
	Coverage                         *TerrainCoverage
	Rewind                           TerrainRewind
	Actors                           []*WorldActor
	Projectiles                      []*WorldProjectile
	SmallShots                       []*WorldSmallShot
	Frame                            uint64
	Money, Score                     int
	MinimumScrollY, MaximumScrollY   int
	cursor                           EncounterCursor
	random                           RandomState
	kinds                            map[int]*visualassets.WaveActor
	paths                            map[int]*visualassets.Path
	fixedKinds                       map[int]*visualassets.FixedSpriteKind
	nextActorID                      int
	fire                             FireCadence
	previousFire                     bool
	movingSpriteBoxes                map[string]visualassets.CollisionBox
	shotSpriteBoxes                  map[string]visualassets.CollisionBox
	playerCollision                  CollisionRect
}

func NewWorld(data LevelData) (*World, error) {
	if data.Number < 1 || data.Number > 5 || data.Terrain == nil || data.Paths == nil || data.Encounters == nil || data.Actors == nil {
		return nil, fmt.Errorf("level needs its complete decoded data")
	}
	if data.Terrain.Columns != 20 || data.Terrain.Rows != 300 || data.Terrain.TileSize != 16 || len(data.Terrain.Map) != 6000 {
		return nil, fmt.Errorf("invalid original terrain dimensions")
	}
	terrain := *data.Terrain
	terrain.Map = append([]uint16(nil), terrain.Map...)
	data.Terrain = &terrain
	w := &World{
		Level: data, Player: PlayerMotionState{X: 160, Y: 176},
		ScrollY: 4608, PreviousScrollY: 4608, RenderScrollY: 4608, VisitedScrollY: 4608,
		MaximumScrollY: 4608, ScrollDelta: 1, Equipment: NewEquipment(), PlayerAlive: true, cursor: NewEncounterCursor(), random: NewRandomState(),
		kinds: make(map[int]*visualassets.WaveActor), paths: make(map[int]*visualassets.Path), fixedKinds: make(map[int]*visualassets.FixedSpriteKind),
	}
	w.PreviousPlayer = w.Player
	w.fire = NewFireCadence(w.Equipment)
	w.movingSpriteBoxes = make(map[string]visualassets.CollisionBox)
	for _, sprite := range data.Actors.Atlas.Sprites {
		if sprite.Collision != nil {
			w.movingSpriteBoxes[sprite.Name] = *sprite.Collision
		}
	}
	if data.Rules != nil {
		if data.Rules.Level != data.Number {
			return nil, fmt.Errorf("level rules do not match their terrain")
		}
		w.MinimumScrollY = data.Rules.InitialMinimumScrollY
		w.MaximumScrollY = data.Rules.InitialMaximumScrollY
		w.shotSpriteBoxes = make(map[string]visualassets.CollisionBox)
		for _, sprite := range data.Rules.EnemyShots.Sprites {
			if sprite.Collision != nil {
				w.shotSpriteBoxes[sprite.Name] = *sprite.Collision
			}
		}
	}
	w.Rewind = NewTerrainRewind(w.ScrollY, w.Player.X, w.Player.Y)
	if data.PlayerStencil != nil {
		stencil := data.PlayerStencil
		if stencil.Width < 1 || stencil.Width > 32 || stencil.Height < 1 || len(stencil.Rows) != stencil.Height {
			return nil, fmt.Errorf("invalid player terrain stencil")
		}
		var err error
		w.Coverage, err = NewTerrainCoverage(data.Terrain)
		if err != nil {
			return nil, err
		}
		w.Level.Terrain.Map = w.Coverage.Map
	}
	for i := range data.Paths.Paths {
		path := &data.Paths.Paths[i]
		if err := ValidatePath(path); err != nil {
			return nil, err
		}
		if w.paths[path.ID] != nil {
			return nil, fmt.Errorf("duplicate path %d", path.ID)
		}
		w.paths[path.ID] = path
	}
	for i := range data.Actors.Kinds {
		kind := &data.Actors.Kinds[i]
		w.kinds[kind.Kind] = kind
	}
	if data.FixedSprites != nil {
		for i := range data.FixedSprites.Kinds {
			kind := &data.FixedSprites.Kinds[i]
			w.fixedKinds[kind.Kind] = kind
		}
	}
	for _, wave := range data.Encounters.Moving {
		if w.kinds[wave.EnemyKind] == nil || w.paths[wave.PathID] == nil {
			return nil, fmt.Errorf("wave references unknown kind or path")
		}
	}
	return w, nil
}

// Step follows the original update ordering: player, existing actors, timers,
// then encounter activation and scroll advancement. New actors first move on
// the next pass. Drawing never changes these states.
func (w *World) Step(input Input) error {
	w.Frame++
	w.PreviousPlayer = w.Player
	w.PreviousScrollY = w.ScrollY
	w.RenderScrollY = w.ScrollY
	w.PreviousBackgroundY = w.BackgroundY
	backgroundStep := (w.ScrollDelta >> 1) + (int(w.Frame&1) & w.ScrollDelta)
	w.BackgroundY = (w.BackgroundY - backgroundStep + 192) % 192
	w.Player.SpeedTier = w.Equipment.SpeedTier
	w.updatePlayerCollision()
	if w.PlayerAlive {
		for _, actor := range w.Actors {
			if actor.Active && !actor.fixed && actor.Collision.Intersects(w.playerCollision) {
				w.damagePlayer(ContactDamage(actor.part.StrongHealth))
				w.damageActor(actor, 127)
				break
			}
		}
	}
	if w.PlayerAlive {
		touching := false
		if w.Coverage != nil {
			touching = w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil)
		}
		handled, crushed := w.Rewind.Advance(&w.Player, w.ScrollY, 1, touching)
		if crushed {
			w.PlayerAlive = false
			w.Equipment.Shield = 0
		}
		if !handled {
			w.Player.Advance(input.Motion, MotionContext{ScrollY: w.ScrollY, VisitedScrollY: w.VisitedScrollY, BaseScrollStep: 1})
			w.Rewind.Record(w.ScrollY, w.Player.X, w.Player.Y)
			if w.Coverage != nil && w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
				w.Rewind.Timer = 1
				w.Player.Inertia = 0
				if limit := w.ScrollY + 16; limit > w.VisitedScrollY {
					w.VisitedScrollY = limit
				}
			}
		}
	} else {
		w.Player.ScrollStep = 0
	}
	for _, actor := range w.Actors {
		if !actor.Active || actor.fixed {
			continue
		}
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		actor.Visible = true
		actor.animationState.Advance(actor.animation)
		if actor.part.MotionMode == "follow-leader" {
			actor.X, actor.Y = actor.leader.X, actor.leader.Y
			actor.Active = actor.leader.Active
		} else {
			if err := actor.motion.Advance(actor.path, &w.Level.Paths.SineTable, func() uint16 { return uint16(w.random.Next()) }); err != nil {
				return err
			}
			actor.X, actor.Y = float64(actor.motion.X>>16), float64(actor.motion.Y>>16)
			actor.Active = actor.motion.Active
			if !actor.Active && actor.part.Linked {
				w.despawnGroup(actor)
			}
		}
		if actor.part != nil && len(actor.part.EntryAnimations) != 0 && !actor.entrySelected {
			actor.animation = actor.part.EntryAnimations[entryEdge(int(actor.X), int(actor.Y))]
			actor.animationState = NewAnimation(actor.animation)
			actor.entrySelected = true
		}
		actor.selectSprite()
		if box, ok := w.movingSpriteBoxes[actor.Sprite]; ok {
			actor.Collision = ActorCollisionRect(box, int(actor.X), int(actor.Y))
		}
		if actor.Active && actor.part.MotionMode != "follow-leader" {
			shot, fired, err := actor.fire.Tick(w.random.Next, w.Player.X-int(actor.X), w.Player.Y-int(actor.Y))
			if err != nil {
				return err
			}
			if fired {
				w.spawnEnemyShot(int(actor.X), int(actor.Y), shot)
			}
		}
	}
	if input.Fire && !w.previousFire {
		w.fire.QueueTrigger()
	}
	w.previousFire = input.Fire
	w.fire.ApplyEquipment(w.Equipment)
	if w.fire.TakePulse() && w.PlayerAlive {
		for _, weapon := range []WeaponSlot{w.Equipment.Primary, w.Equipment.Rear, w.Equipment.Side} {
			switch weapon.Item {
			case ItemForwardShot, ItemDoubleShot, ItemRearShot, ItemSideShot:
				shots, err := AppendSmallWeaponShots(nil, weapon, w.Player.X, w.Player.Y)
				if err != nil {
					return err
				}
				for _, shot := range shots {
					w.nextActorID++
					w.SmallShots = append([]*WorldSmallShot{{ID: w.nextActorID, PreviousX: shot.X, PreviousY: shot.Y, Shot: shot, Active: true}}, w.SmallShots...)
				}
			}
		}
	}
	// Both families belonged to the same newest-first projectile list. Merge
	// their stable creation IDs to preserve hits and removals in that order.
	for enemy, player := 0, 0; enemy < len(w.Projectiles) || player < len(w.SmallShots); {
		if player == len(w.SmallShots) || (enemy < len(w.Projectiles) && w.Projectiles[enemy].ID > w.SmallShots[player].ID) {
			if err := w.advanceEnemyShot(w.Projectiles[enemy]); err != nil {
				return err
			}
			enemy++
		} else {
			w.advanceSmallShot(w.SmallShots[player])
			player++
		}
	}
	for _, actor := range w.Actors {
		if !actor.Active || !actor.fixed {
			continue
		}
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		actor.Visible = true
		actor.animationState.Advance(actor.animation)
		actor.Y = float64(actor.mapY - w.ScrollY)
		actor.selectSprite()
	}
	w.Equipment.AdvanceTimers()
	if err := w.fire.Tick(input.Fire); err != nil {
		return err
	}
	var spawnErr error
	w.cursor.Activate(w.ScrollY, w.Level.Encounters,
		func(wave visualassets.Wave) {
			if spawnErr == nil {
				spawnErr = w.spawnWave(wave)
			}
		}, w.spawnFixed)
	if spawnErr != nil {
		return spawnErr
	}
	scroll := w.ScrollY - w.Player.ScrollStep
	if scroll < w.MinimumScrollY {
		scroll = w.MinimumScrollY
	}
	if scroll > w.MaximumScrollY {
		scroll = w.MaximumScrollY
	}
	w.ScrollDelta = w.ScrollY - scroll
	w.ScrollY = scroll
	// The native scroll limit allows sixteen pixels back from the current
	// position, and remains bounded by the original map's initial view.
	if scroll+16 < w.VisitedScrollY {
		w.VisitedScrollY = scroll + 16
	}
	w.compactActors()
	return nil
}

func (w *World) advanceEnemyShot(projectile *WorldProjectile) error {
	projectile.PreviousX, projectile.PreviousY = projectile.X, projectile.Y
	var err error
	projectile.Active, err = projectile.Motion.Advance(w.ScrollDelta)
	if err != nil {
		return err
	}
	projectile.X, projectile.Y = float64(projectile.Motion.X>>16), float64(projectile.Motion.Y>>16)
	if projectile.Active && w.PlayerAlive {
		if box, ok := w.shotSpriteBoxes[projectile.Sprite]; ok && ActorCollisionRect(box, int(projectile.X), int(projectile.Y)).Intersects(w.playerCollision) {
			w.damagePlayer(EnemyBulletDamage)
			projectile.Active = false
		}
	}
	return nil
}

func (w *World) advanceSmallShot(shot *WorldSmallShot) {
	shot.PreviousX, shot.PreviousY = shot.Shot.X, shot.Shot.Y
	shot.Active = shot.Shot.Advance(!w.PlayerAlive)
	if !shot.Active {
		return
	}
	for _, actor := range w.Actors {
		if actor.Active && !actor.fixed && actor.Collision.Contains(shot.Shot.X, shot.Shot.Y) {
			w.damageActor(actor, shot.Shot.Damage)
			shot.Active = false
			return
		}
	}
}

// entryEdge reproduces the source's corner comparisons, including the
// asymmetric signs used while an enemy is outside two viewport edges.
func entryEdge(x, y int) int {
	if x < 0 {
		if y < 0 {
			if y < x {
				return 1
			}
			return 0
		}
		if -(y - 192) < x {
			return 3
		}
		return 0
	}
	if y < 0 {
		if y < -(x - 320) {
			return 1
		}
		return 2
	}
	if x-320 < y-192 {
		return 2
	}
	return 3
}

func (w *World) spawnWave(wave visualassets.Wave) error {
	kind, path := w.kinds[wave.EnemyKind], w.paths[wave.PathID]
	for member := range wave.Count {
		var leader *WorldActor
		group := make([]*WorldActor, 0, len(kind.Parts))
		for partIndex := range kind.Parts {
			part := &kind.Parts[partIndex]
			config, err := FormationMotion(wave, member, partIndex, *part)
			if err != nil {
				return err
			}
			motion, err := NewPathMotion(path, config)
			if err != nil {
				return err
			}
			// Native creation consumes one shared random call per part, even
			// when the formation's firing rate is zero.
			fire := NewEnemyFireState(wave.FireDelay, w.random.Next())
			w.nextActorID++
			actor := &WorldActor{ID: w.nextActorID, Atlas: "moving", Active: true, Score: part.Score, part: part, path: path, motion: motion,
				animation: part.Animation, animationState: NewAnimation(part.Animation), leader: leader, fire: fire}
			if w.Level.Rules != nil {
				actor.Health = w.Level.Rules.OrdinaryHealthMultiplier
				if part.StrongHealth {
					actor.Health = w.Level.Rules.StrongHealthMultiplier
					if kind.StrongHealthOverride > 0 {
						actor.Health = kind.StrongHealthOverride
					}
				}
			}
			actor.X, actor.Y = float64(motion.X>>16), float64(motion.Y>>16)
			actor.PreviousX, actor.PreviousY = actor.X, actor.Y
			if leader == nil {
				leader = actor
			}
			actor.selectSprite()
			if box, ok := w.movingSpriteBoxes[actor.Sprite]; ok {
				actor.Collision = ActorCollisionRect(box, int(actor.X), int(actor.Y))
			} else {
				actor.Collision = CollisionRect{Right: -1, Bottom: -1}
			}
			group = append(group, actor)
		}
		// The original prepends each formation member to its actor list while
		// retaining the order of parts within that member.
		w.Actors = append(group, w.Actors...)
	}
	return nil
}

func (w *World) updatePlayerCollision() {
	w.playerCollision = CollisionRect{Right: -1, Bottom: -1}
	if w.Level.Ships == nil {
		return
	}
	name := fmt.Sprintf("player-ship-%d", w.Player.BankFrame())
	for _, sprite := range w.Level.Ships.Atlas.Sprites {
		if sprite.Name == name && sprite.Collision != nil {
			w.playerCollision = ActorCollisionRect(*sprite.Collision, w.Player.X, w.Player.Y)
			return
		}
	}
}

func (w *World) damagePlayer(amount int) {
	result := ApplyShieldDamage(w.Equipment.Shield, amount, w.Equipment.Protection, false)
	w.Equipment.Shield = result.Shield
	if result.Destroyed {
		w.PlayerAlive = false
	}
}

func (w *World) damageActor(actor *WorldActor, amount uint16) {
	target := actor
	if actor.part != nil && actor.part.DamageMode == "group" && actor.leader != nil {
		target = actor.leader
	}
	result := ApplyEnemyDamage(uint16(target.Health), amount)
	target.Health = int(result.Health)
	if !result.Destroyed {
		return
	}
	w.Score += target.Score
	target.Active = false
	if actor.part != nil && actor.part.DamageMode == "group" {
		w.despawnGroup(target)
	}
}

func (w *World) despawnGroup(actor *WorldActor) {
	leader := actor
	if actor.leader != nil {
		leader = actor.leader
	}
	for _, member := range w.Actors {
		if member == leader || member.leader == leader {
			member.Active = false
		}
	}
}

func (w *World) spawnEnemyShot(x, y int, shot EnemyShot) {
	if w.Level.Rules == nil {
		return
	}
	w.nextActorID++
	p := &WorldProjectile{ID: w.nextActorID, X: float64(x), Y: float64(y), PreviousX: float64(x), PreviousY: float64(y),
		Sprite: w.Level.Rules.DefaultEnemyShot, Atlas: "enemy-shots", Active: true,
		Motion: DirectionalProjectile{X: int32(x) << 16, Y: int32(y) << 16, Direction: shot.Direction, Speed: shot.Speed}}
	w.Projectiles = append([]*WorldProjectile{p}, w.Projectiles...)
}

func (w *World) spawnFixed(record visualassets.FixedEncounter) {
	kind := w.fixedKinds[record.EnemyKind]
	if kind == nil {
		return
	}
	variant := record.Variant
	if kind.VariantSelection == "initial-x-side" {
		variant = 0
		if record.X >= 160 {
			variant = 1
		}
	}
	for _, v := range kind.Variants {
		if v.ID != variant {
			continue
		}
		w.nextActorID++
		a := &WorldActor{ID: w.nextActorID, X: float64(record.X - 8), Y: float64(record.Y - 8 - w.ScrollY), Atlas: "fixed", Active: true,
			mapY: record.Y - 8, fixed: true, Health: kind.Health, Score: kind.Score,
			animation: v.Animation, animationState: NewAnimation(v.Animation)}
		a.PreviousX, a.PreviousY = a.X, a.Y
		a.selectSprite()
		w.Actors = append([]*WorldActor{a}, w.Actors...)
		return
	}
}

func (a *WorldActor) selectSprite() {
	if a.part != nil && len(a.part.HeadingFrames) != 0 {
		heading := int(uint8(uint32(a.motion.AngleFixed) >> 16))
		index := ((heading + a.part.HeadingOffset) >> a.part.HeadingShift) % len(a.part.HeadingFrames)
		a.Sprite = a.part.HeadingFrames[index]
		return
	}
	a.Sprite = a.animationState.Sprite(a.animation)
}

func (w *World) compactActors() {
	live := w.Actors[:0]
	for _, actor := range w.Actors {
		if actor.Active {
			live = append(live, actor)
		}
	}
	clear(w.Actors[len(live):])
	w.Actors = live
	projectiles := w.Projectiles[:0]
	for _, projectile := range w.Projectiles {
		if projectile.Active {
			projectiles = append(projectiles, projectile)
		}
	}
	clear(w.Projectiles[len(projectiles):])
	w.Projectiles = projectiles
	shots := w.SmallShots[:0]
	for _, shot := range w.SmallShots {
		if shot.Active {
			shots = append(shots, shot)
		}
	}
	clear(w.SmallShots[len(shots):])
	w.SmallShots = shots
}
