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
	Common        *visualassets.SpriteAtlas
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
	CarriedReward              int
	WaveToken                  uint16
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

type WorldCollectible struct {
	ID                         int
	X, Y, PreviousX, PreviousY float64
	Sprite                     string
	Active                     bool
	Reward                     int
	Cash                       int
	Motion                     CashMotion
	animation                  visualassets.NamedActorAnimation
	animationState             AnimationState
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
	Dive                             DiveState
	RenderDivePhase                  int
	InvulnerableFrames               int
	MaterializationFrames            int
	SoundRequests                    [4]string
	ScreenClearFrames                int
	ScreenClearPaletteMask           uint16
	PendingExitDrops                 int
	ExitReady                        bool
	Checkpoint                       CheckpointState
	WaveBonuses                      WaveBonusCache
	Ready                            bool
	GameOver                         bool
	PlayerSprite                     string
	Actors                           []*WorldActor
	Projectiles                      []*WorldProjectile
	SmallShots                       []*WorldSmallShot
	Collectibles                     []*WorldCollectible
	Frame                            uint64
	Money, Score                     int
	MinimumScrollY, MaximumScrollY   int
	ScrollDeviationPasses            int
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
	commonAnimations                 map[string]visualassets.NamedActorAnimation
	deathAnimation                   visualassets.NamedActorAnimation
	deathState                       AnimationState
	blockedFireUntilRelease          bool
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
		MaximumScrollY: 4608, ScrollDelta: 1, Equipment: NewEquipment(), PlayerAlive: true, MaterializationFrames: 8, cursor: NewEncounterCursor(), random: NewRandomState(),
		kinds: make(map[int]*visualassets.WaveActor), paths: make(map[int]*visualassets.Path), fixedKinds: make(map[int]*visualassets.FixedSpriteKind),
	}
	w.PreviousPlayer = w.Player
	w.WaveBonuses = NewWaveBonusCache()
	w.Checkpoint = CheckpointState{ScrollY: w.ScrollY, PlayerX: w.Player.X, WorldY: w.Player.Y, Loadout: w.Equipment.WeaponLoadout}
	w.fire = NewFireCadence(w.Equipment)
	w.movingSpriteBoxes = make(map[string]visualassets.CollisionBox)
	w.commonAnimations = make(map[string]visualassets.NamedActorAnimation)
	if data.Common != nil {
		for _, sprite := range data.Common.Sprites {
			if sprite.Collision != nil {
				w.movingSpriteBoxes[sprite.Name] = *sprite.Collision
			}
		}
		for _, animation := range data.Common.Animations {
			w.commonAnimations[animation.ID] = animation
		}
	}
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
	w.VisitedScrollY = w.MaximumScrollY
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
	clear(w.SoundRequests[:])
	if w.ScreenClearFrames != 0 || w.GameOver {
		return nil
	}
	if w.Ready {
		if input.Fire {
			w.Ready = false
			w.blockedFireUntilRelease = true
		}
		return nil
	}
	if w.blockedFireUntilRelease {
		w.blockedFireUntilRelease = input.Fire
		input.Fire = false
	}
	w.Frame++
	w.WaveBonuses.BeginPass(w.Frame)
	w.PreviousPlayer = w.Player
	w.PreviousScrollY = w.ScrollY
	w.RenderScrollY = w.ScrollY
	w.PreviousBackgroundY = w.BackgroundY
	backgroundStep := (w.ScrollDelta >> 1) + (int(w.Frame&1) & w.ScrollDelta)
	w.BackgroundY = (w.BackgroundY - backgroundStep + 192) % 192
	w.Player.SpeedTier = w.Equipment.SpeedTier
	w.Player.ScrollStep = 1
	w.RenderDivePhase = w.Dive.Phase
	deathFinished := false
	if !w.PlayerAlive && len(w.deathAnimation.Animation.Frames) != 0 {
		w.deathState.Advance(w.deathAnimation.Animation)
		w.PlayerSprite = w.deathState.Sprite(w.deathAnimation.Animation)
		deathFinished = w.deathState.Remaining == 0 && w.MaterializationFrames >= 16
	}
	w.updatePlayerCollision()
	if w.PlayerAlive && w.Dive.Phase == 0 {
		contactRect := w.playerCollision
		if w.Equipment.ShadesFrames > 0 {
			contactRect = CollisionRect{Left: w.Player.X - 64, Top: w.Player.Y - 64, Right: w.Player.X + 64, Bottom: w.Player.Y + 64}
		}
		for _, actor := range w.Actors {
			if actor.Active && !actor.fixed && actor.Collision.Intersects(contactRect) {
				if w.Equipment.ShadesFrames == 0 {
					w.damagePlayer(ContactDamage(actor.part.StrongHealth))
				}
				w.damageActor(actor, 127)
				break
			}
		}
	}
	if w.PlayerAlive {
		touching := false
		if w.Coverage != nil && w.Dive.Phase == 0 {
			touching = w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil)
		}
		handled, crushed := false, false
		if w.Dive.Phase == 0 {
			handled, crushed = w.Rewind.Advance(&w.Player, w.ScrollY, 1, touching)
		}
		if crushed {
			w.destroyPlayer()
		}
		if !handled {
			w.Player.Advance(input.Motion, MotionContext{ScrollY: w.ScrollY, VisitedScrollY: w.VisitedScrollY, BaseScrollStep: 1})
			w.Rewind.Record(w.ScrollY, w.Player.X, w.Player.Y)
			if w.Dive.Phase == 0 && w.Coverage != nil && w.Coverage.Touches(w.Player.X, w.Player.Y, w.ScrollY, *w.Level.PlayerStencil) {
				w.Rewind.Timer = 1
				w.Player.Inertia = 0
				if limit := w.ScrollY + 16; limit > w.MaximumScrollY {
					w.MaximumScrollY = limit
					w.VisitedScrollY = limit
				}
			}
		}
	} else {
		w.Player.ScrollStep = 0
	}
	if w.Dive.AdvancePhase() {
		w.Rewind.Timer = -15
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
			if !actor.Active {
				w.WaveBonuses.Escape(actor.WaveToken)
				if actor.part.Linked {
					w.despawnGroup(actor)
				}
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
	if w.fire.TakePulse() && w.PlayerAlive && w.Dive.Phase == 0 {
		for _, weapon := range []WeaponSlot{w.Equipment.Primary, w.Equipment.Rear, w.Equipment.Side} {
			switch weapon.Item {
			case ItemForwardShot, ItemDoubleShot, ItemRearShot, ItemSideShot:
				if weapon.Item != ItemForwardShot && w.MaterializationFrames != 0 {
					continue
				}
				shots, err := AppendSmallWeaponShots(nil, weapon, w.Player.X, w.Player.Y)
				if err != nil {
					return err
				}
				for _, shot := range shots {
					if w.SoundRequests[2] == "" {
						w.SoundRequests[2] = "sampled-effect-09"
					}
					w.nextActorID++
					w.SmallShots = append([]*WorldSmallShot{{ID: w.nextActorID, PreviousX: shot.X, PreviousY: shot.Y, Shot: shot, Active: true}}, w.SmallShots...)
				}
			}
		}
	}
	if w.InvulnerableFrames > 0 && w.MaterializationFrames == 0 {
		w.InvulnerableFrames--
	}
	// Both families belonged to the same newest-first projectile list. Merge
	// their stable creation IDs to preserve hits and removals in that order.
	collectiblesAtStart := w.Collectibles
	for enemy, player, collectible := 0, 0, 0; enemy < len(w.Projectiles) || player < len(w.SmallShots) || collectible < len(collectiblesAtStart); {
		enemyID, playerID, collectibleID := -1, -1, -1
		if enemy < len(w.Projectiles) {
			enemyID = w.Projectiles[enemy].ID
		}
		if player < len(w.SmallShots) {
			playerID = w.SmallShots[player].ID
		}
		if collectible < len(collectiblesAtStart) {
			collectibleID = collectiblesAtStart[collectible].ID
		}
		if collectibleID > max(enemyID, playerID) {
			w.advanceCollectible(collectiblesAtStart[collectible])
			collectible++
		} else if enemyID > playerID {
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
	w.Dive.Tick()
	if w.Dive.Phase != 0 || !w.PlayerAlive {
		if w.MaterializationFrames < 16 {
			w.MaterializationFrames++
		}
	} else if w.MaterializationFrames > 0 {
		w.MaterializationFrames--
	}
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
	if input.Dive && w.PlayerAlive {
		if w.Dive.Request(&w.Equipment.DiveCharges) {
			w.SoundRequests[2] = "synthesized-effect-16"
		}
	}
	scroll := ScrollState{Y: w.ScrollY, Minimum: w.MinimumScrollY, Maximum: w.MaximumScrollY, DeviationPasses: w.ScrollDeviationPasses}
	scroll.Advance(w.Player.ScrollStep, 1, input.Motion.Down)
	w.ScrollDelta, w.ScrollY = scroll.ActualStep, scroll.Y
	w.MaximumScrollY, w.VisitedScrollY = scroll.Maximum, scroll.Maximum
	w.ScrollDeviationPasses = scroll.DeviationPasses
	w.compactActors()
	if deathFinished {
		w.Equipment.Lives--
		w.Equipment.Shield = 39
		w.Equipment.FireAdvance = 1
		if w.Equipment.Lives == 0 {
			w.GameOver = true
		} else {
			w.RestartCheckpoint()
			w.Ready = true
		}
	}
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
	if projectile.Active && w.PlayerAlive && w.Dive.Phase == 0 {
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
	var spawned []*WorldActor
	for member := range wave.Count {
		var leader *WorldActor
		group := make([]*WorldActor, 0, len(kind.Parts))
		for partIndex := range kind.Parts {
			part := &kind.Parts[partIndex]
			config, err := FormationMotion(wave, member, partIndex, *part)
			if err != nil {
				return err
			}
			if kind.MotionBudgetOverride > 0 {
				config.Budget = kind.MotionBudgetOverride
			}
			motion, err := NewPathMotion(path, config)
			if err != nil {
				return err
			}
			// Native creation consumes one shared random call per part, even
			// when the formation's firing rate is zero.
			fire := NewEnemyFireState(wave.FireRate, w.random.Next())
			w.nextActorID++
			actor := &WorldActor{ID: w.nextActorID, Atlas: "moving", Active: true, Score: part.Score, part: part, path: path, motion: motion,
				animation: part.Animation, animationState: NewAnimation(part.Animation), leader: leader, fire: fire}
			if part.Atlas != "" {
				actor.Atlas = part.Atlas
			}
			if kind.CarriedRewardFromMotionBudget {
				actor.CarriedReward = wave.MotionBudget
			}
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
			spawned = append(spawned, actor)
		}
		// The original prepends each formation member to its actor list while
		// retaining the order of parts within that member.
		w.Actors = append(group, w.Actors...)
	}
	if len(spawned) != 0 {
		token := w.WaveBonuses.Register(spawned[len(spawned)-1].part.StrongHealth, wave.Count)
		for _, actor := range spawned {
			actor.WaveToken = token
		}
		// Carrier creation registers its bucket before the wrapper clears the
		// carrier's token. Its orphan count can suppress a shared wave bonus.
		if kind.CarriedRewardFromMotionBudget {
			spawned[len(spawned)-1].WaveToken = 0
		}
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
	result := ApplyShieldDamage(w.Equipment.Shield, amount, w.Equipment.Protection, w.InvulnerableFrames != 0 || w.PendingExitDrops != 0)
	w.Equipment.Shield = result.Shield
	if result.Destroyed {
		w.destroyPlayer()
	}
}

func (w *World) destroyPlayer() {
	if !w.PlayerAlive || w.PendingExitDrops != 0 {
		return
	}
	w.PlayerAlive = false
	w.Equipment.Shield = 0
	w.deathAnimation = w.commonAnimations["player-death"]
	w.deathState = NewAnimation(w.deathAnimation.Animation)
	w.PlayerSprite = w.deathState.Sprite(w.deathAnimation.Animation)
	w.SoundRequests[2] = "synthesized-effect-10"
	w.SoundRequests[1] = "synthesized-effect-10"
}

func (w *World) damageActor(actor *WorldActor, amount uint16) {
	if actor.part != nil && actor.part.DamageMode == "drop-equipment" {
		w.spawnPickup(actor.CarriedReward, int(actor.X), int(actor.Y))
		actor.Active = false
		w.SoundRequests[2] = "synthesized-effect-17"
		return
	}
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
	if w.WaveBonuses.Defeat(target.WaveToken) {
		w.spawnWaveCash(int(target.X), int(target.Y), target.part.StrongHealth)
	}
	w.SoundRequests[2] = "synthesized-effect-17"
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
	if record.EnemyKind == 0 {
		w.captureCheckpoint(record.TriggerY, record.X, record.Y)
		return
	}
	kind := w.fixedKinds[record.EnemyKind]
	if kind == nil {
		return
	}
	variant := record.Variant
	if kind.VariantSelection == "initial-x-side" {
		variant = 0
		if record.X-8 > kind.VariantThresholdX {
			variant = 1
		}
	}
	for _, v := range kind.Variants {
		if v.ID != variant {
			continue
		}
		w.nextActorID++
		a := &WorldActor{ID: w.nextActorID, X: float64(record.X + v.OriginOffsetX), Y: float64(record.Y + v.OriginOffsetY - w.ScrollY), Atlas: "fixed", Active: true,
			mapY: record.Y + v.OriginOffsetY, fixed: true, Health: kind.Health, Score: kind.Score,
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
	collectibles := w.Collectibles[:0]
	for _, collectible := range w.Collectibles {
		if collectible.Active {
			collectibles = append(collectibles, collectible)
		}
	}
	clear(w.Collectibles[len(collectibles):])
	w.Collectibles = collectibles
}
