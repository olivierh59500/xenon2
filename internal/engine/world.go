package engine

import (
	"fmt"

	"xenon2/internal/visualassets"
)

// LevelData contains decoded graphics and semantic game data only.
type LevelData struct {
	InitialEquipment *Equipment
	InitialRandom    *RandomState
	Number           int
	Terrain          *visualassets.Terrain
	Paths            *visualassets.Paths
	Encounters       *visualassets.Encounters
	Actors           *visualassets.Actors
	FixedSprites     *visualassets.FixedSprites
	FixedTiles       *visualassets.FixedTiles
	PlayerStencil    *visualassets.PlayerTerrainStencil
	Rules            *visualassets.LevelRules
	Ships            *visualassets.ShipArt
	Common           *visualassets.SpriteAtlas
	Guardians        *visualassets.Guardians
	GuardianGroups   []visualassets.GuardianGroup
	GuardianParts    *visualassets.SpriteAtlas
}

type Input struct {
	Motion     MotionInput
	Fire, Dive bool
}

// WorldActor separates simulation positions from interpolated display state.
// Fixed actors retain map coordinates; moving actors retain their path state.
type WorldActor struct {
	DrawKind                   string
	DrawLength                 int
	DrawUp                     bool
	fifthIndex                 int
	fifthFinal, fifthMouth     bool
	fifthPart                  *visualassets.GuardianComponent
	fifthSeeking               *FifthSeekingState
	fifthColumn                *FifthLaserColumnState
	Binding                    ActorPoolBinding
	ID                         int
	Order                      int
	X, Y, PreviousX, PreviousY float64
	Sprite, Atlas              string
	ActorList                  string
	Active                     bool
	Visible                    bool
	Materializing              bool
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
	Patch                      *visualassets.TilePatch
	Flash                      bool
	Extras                     []WorldSpriteAttachment
	TileOverlays               []WorldTileOverlay
	fourthCrawler              *FourthCrawlerState
	fourthCrawlerArt           *visualassets.FixedSpriteKind
	firstGuardian              bool
	secondGuardian             bool
	firstSegment               int
	fixedKind                  *visualassets.FixedSpriteKind
	fixedState                 FixedSpriteState
	fixedAiming                *AnimatedAimingFixedProjectile
	fixedTileState             *FixedTileState
	fixedTileArt               *visualassets.FixedTileKind
	fixedTileVariant           *visualassets.FixedTileVariant
	fixedHatch                 *FixedHatchState
	hatchCreature              *HatchCreatureState
	fixedPod                   *FixedPodState
	podCreature                *PodCreatureState
	firstMiddleAnchor          *FirstMiddleAnchor
	firstMiddleFollower        *FirstMiddleFollower
	firstMiddleFragment        *FirstMiddleFragment
	thirdCrawler               *ThirdCrawlerState
	thirdChain                 *ThirdChainState
	thirdChainPart             int
	thirdChainSentinel         bool
	thirdCannon                *ThirdCannonState
	thirdScenery               bool
	thirdChainMembers          [8]*WorldActor
	thirdMiddlePart            int
	thirdFinalMember           *ThirdFinalMember
	fourthIndex                int
	fourthFinal                bool
	thirdPart                  *visualassets.GuardianComponent
	firstMiddleSentinel        bool
	firstMiddleStream          int
	firstMiddleGate            int
	firstMiddlePart            *visualassets.GuardianComponent
	firstMiddleMarkers         *[2]*WorldActor
	secondNode                 *SecondDefenseNodeState
	secondPart                 *visualassets.GuardianComponent
	secondSegment              *SecondDefenseSegment
	secondFragment             *SecondDefenseFragment
	secondMinion               *SecondMinionState
}

type WorldSpriteAttachment struct {
	Sprite, Atlas string
	X, Y          float64
}

type WorldTileOverlay struct {
	Patch visualassets.TilePatch
	X, Y  float64
}

type WorldProjectile struct {
	Binding                    ActorPoolBinding
	ID                         int
	X, Y, PreviousX, PreviousY float64
	Sprite, Atlas              string
	Active                     bool
	Motion                     DirectionalProjectile
	animation                  visualassets.ActorAnimation
	animationState             AnimationState
	turning                    *TurningFixedProjectile
}

type WorldSmallShot struct {
	Binding              ActorPoolBinding
	ID                   int
	PreviousX, PreviousY int
	Shot                 SmallShot
	Active               bool
}

type WorldCollectible struct {
	Binding                    ActorPoolBinding
	ID                         int
	Order                      int
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
	FifthMiddle                      *FifthMiddleGuardianState
	FifthFinal                       *FifthFinalGuardianState
	fifthMiddleArt, fifthFinalArt    *visualassets.GuardianGroup
	fifthMiddleActors                [10]*WorldActor
	fifthFinalActors                 [22]*WorldActor
	Level                            LevelData
	Player, PreviousPlayer           PlayerMotionState
	Shadows                          [4]PlayerShadowState
	ScrollY, PreviousScrollY         int
	RenderScrollY                    int
	ScrollDelta                      int
	BaseScrollStep                   int
	BackgroundY, PreviousBackgroundY int
	BackgroundStars                  *BackgroundStarfield
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
	ImmediateSoundRequests           [4]string
	EffectActive                     [4]bool
	StopEffectsRequested             bool
	PendingFixedShots                []FixedSpriteEvents
	UnimplementedFixedEncounters     []visualassets.FixedEncounter
	ScreenClearFrames                int
	ScreenClearPaletteMask           uint16
	PendingExitDrops                 int
	ExitReady                        bool
	ShopReady                        bool
	LevelFinished                    bool
	Checkpoint                       CheckpointState
	WaveBonuses                      WaveBonusCache
	Ready                            bool
	GameOver                         bool
	ContinueCredits                  int
	AdviceIndex                      int
	PlayerSprite                     string
	Actors                           []*WorldActor
	Projectiles                      []*WorldProjectile
	SmallShots                       []*WorldSmallShot
	Collectibles                     []*WorldCollectible
	ThirdMiddle                      *ThirdGuardianState
	ThirdFinal                       *ThirdFinalState
	ThirdStage                       ThirdStageState
	FourthMiddle                     *FourthMiddleGuardian
	FourthFinal                      *FourthFinalGuardian
	fourthMiddleArt                  *visualassets.GuardianGroup
	fourthFinalArt                   *visualassets.GuardianGroup
	fourthMiddleActors               [20]*WorldActor
	fourthFinalActors                [19]*WorldActor
	thirdMiddleArt                   *visualassets.GuardianGroup
	thirdFinalArt                    *visualassets.GuardianGroup
	thirdMiddleActors                [17]*WorldActor
	thirdMiddleUpdated               bool
	thirdFinalUpdated                bool
	FirstGuardian                    *FirstGuardianState
	FirstGuardianSegments            *FirstGuardianSegments
	FirstMiddle                      *FirstMiddleState
	SecondGuardian                   *SecondGuardianState
	PendingGuardianMinions           []SecondGuardianEvents
	Weapons                          *WeaponRuntime
	Pool                             *ActorPool
	Frame                            uint64
	MovingEnemyCount                 int
	Money, Score                     int
	DisplayScore                     int
	MinimumScrollY, MaximumScrollY   int
	ScrollDeviationPasses            int
	cursor                           EncounterCursor
	random                           RandomState
	kinds                            map[int]*visualassets.WaveActor
	paths                            map[int]*visualassets.Path
	fixedKinds                       map[int]*visualassets.FixedSpriteKind
	nextActorID                      int
	poolBindings                     map[int]ActorPoolBinding
	poolShadows                      [4]ActorPoolBinding
	poolActors                       [ActorPoolCapacity]*WorldActor
	poolProjectiles                  [ActorPoolCapacity]*WorldProjectile
	poolSmallShots                   [ActorPoolCapacity]*WorldSmallShot
	poolCollectibles                 [ActorPoolCapacity]*WorldCollectible
	poolError                        error
	fire                             FireCadence
	previousFire                     bool
	movingSpriteBoxes                map[string]visualassets.CollisionBox
	shotSpriteBoxes                  map[string]visualassets.CollisionBox
	playerCollision                  CollisionRect
	commonAnimations                 map[string]visualassets.NamedActorAnimation
	deathAnimation                   visualassets.NamedActorAnimation
	deathState                       AnimationState
	blockedFireUntilRelease          bool
	firstGuardianArt                 *visualassets.GuardianVisual
	firstGuardianBody                visualassets.TilePatch
	firstMiddleArt                   *visualassets.GuardianGroup
	firstGuardianActor               *WorldActor
	firstGuardianParts               [8]*WorldActor
	secondGuardianArt                *visualassets.GuardianVisual
	secondGuardianActor              *WorldActor
	secondGuardianBody               visualassets.TilePatch
	secondScheduler                  *SecondDefenseScheduler
	secondWaveArt                    *visualassets.GuardianGroup
	secondNodeArt                    *visualassets.GuardianGroup
	secondNodes                      [3]*WorldActor
	secondDefenseRemaining           int
	secondGateCounters               [8]int
	secondStreamsUpdated             [2]bool
	secondMiddleReleased             bool
	secondBackward                   bool
	secondMinionConfig               *SecondMinionConfig
	secondTerrainCells               *SecondTerrainCells
	nextTailOrder                    int
	weaponIDs                        []int
	weaponTargets                    []WeaponTarget
	weaponTargetActors               map[int]*WorldActor
	shipTrail                        [4]PlayerMotionState
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
	equipment, random := NewEquipment(), NewRandomState()
	if data.InitialEquipment != nil {
		equipment = *data.InitialEquipment
	}
	if data.InitialRandom != nil {
		random = *data.InitialRandom
	}
	w := &World{
		Level: data, Player: PlayerMotionState{X: 160, Y: 176},
		ScrollY: 4608, PreviousScrollY: 4608, RenderScrollY: 4608, VisitedScrollY: 4608,
		MaximumScrollY: 4608, ScrollDelta: 1, BaseScrollStep: 1, Equipment: equipment, PlayerAlive: true, MaterializationFrames: 8, cursor: NewEncounterCursor(), random: random,
		kinds: make(map[int]*visualassets.WaveActor), paths: make(map[int]*visualassets.Path), fixedKinds: make(map[int]*visualassets.FixedSpriteKind),
	}
	w.PreviousPlayer = w.Player
	for i := range w.shipTrail {
		w.shipTrail[i] = w.Player
	}
	if data.Common != nil {
		var err error
		w.Weapons, err = NewWeaponRuntime(data.Common)
		if err != nil {
			return nil, err
		}
	}
	w.ContinueCredits = 2
	if err := w.initializeWorldPool(); err != nil {
		return nil, err
	}
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
		for _, sprite := range data.FixedSprites.Atlas.Sprites {
			if sprite.Collision != nil {
				w.movingSpriteBoxes[sprite.Name] = *sprite.Collision
			}
		}
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
	if data.Number == 1 && data.Guardians != nil && len(data.Guardians.Visuals) != 0 {
		w.firstGuardianArt = &data.Guardians.Visuals[0]
		w.firstGuardianBody = w.firstGuardianArt.Body
		w.firstGuardianBody.Tiles = append([]uint16(nil), w.firstGuardianBody.Tiles...)
		state := NewFirstGuardianState(w.firstGuardianArt.InitialHealth)
		w.FirstGuardian = &state
		part := &visualassets.ActorPart{ResourceTag: 80, DamageMode: "first-guardian", MotionMode: "first-guardian-controller"}
		w.firstGuardianActor = &WorldActor{Active: true, ActorList: "moving", Atlas: "guardians", part: part, firstGuardian: true,
			Collision: CollisionRect{Right: -1, Bottom: -1}}
		w.Actors = append(w.Actors, w.firstGuardianActor)
	}
	if data.Number == 2 && data.Guardians != nil && len(data.Guardians.Visuals) != 0 {
		w.secondGuardianArt = &data.Guardians.Visuals[0]
		state := NewSecondGuardianState()
		w.SecondGuardian = &state
		w.secondGuardianBody = w.secondGuardianArt.Body
		w.secondGuardianBody.Tiles = append([]uint16(nil), w.secondGuardianBody.Tiles...)
		part := &visualassets.ActorPart{ResourceTag: 80, DamageMode: "second-guardian", MotionMode: "second-guardian-controller"}
		w.secondGuardianActor = &WorldActor{Active: true, ActorList: "moving", Atlas: "guardians", part: part,
			secondGuardian: true, Health: w.secondGuardianArt.InitialHealth, Collision: CollisionRect{Right: -1, Bottom: -1}}
		w.Actors = append(w.Actors, w.secondGuardianActor)
	}
	if err := w.initializeSecondArena(); err != nil {
		return nil, err
	}
	if err := w.initializeThirdStage(); err != nil {
		return nil, err
	}
	if err := w.initializeFourthStage(); err != nil {
		return nil, err
	}
	if err := w.initializeFirstMiddle(); err != nil {
		return nil, err
	}
	for i := len(w.Actors) - 1; i >= 0; i-- {
		if err := w.bindWorldActor(w.Actors[i]); err != nil {
			return nil, err
		}
	}
	if err := w.initializeFifthStage(); err != nil {
		return nil, err
	}
	return w, nil
}

// NextUIRandom shares the original random stream with shop animations and
// dialogue choices, preserving its continuity when gameplay resumes.
func (w *World) NextUIRandom() uint16 {
	return uint16(w.random.Next())
}

// SetRandomState transfers the shared stream from the attract/menu director or
// the other player's turn. It does not reset any gameplay state.
func (w *World) SetRandomState(state RandomState) { w.random = state }

func (w *World) RandomState() RandomState { return w.random }

// AcceptContinue restores three ships at the same checkpoint. The original
// resets score while retaining the recorded weapons, wallet and stage state.
func (w *World) AcceptContinue() bool {
	if !w.GameOver || w.ContinueCredits == 0 {
		return false
	}
	w.ContinueCredits--
	w.GameOver = false
	w.Equipment.Lives = 3
	w.Score = 0
	w.DisplayScore = 0
	w.RestartCheckpoint()
	w.Ready = true
	return true
}

// Step follows the original update ordering: player, existing actors, timers,
// then encounter activation and scroll advancement. New actors first move on
// the next pass. Drawing never changes these states.
func (w *World) Step(input Input) error {
	clear(w.SoundRequests[:])
	clear(w.ImmediateSoundRequests[:])
	w.StopEffectsRequested = false
	w.PendingFixedShots = w.PendingFixedShots[:0]
	w.PendingGuardianMinions = w.PendingGuardianMinions[:0]
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
	w.advanceBackgroundStars()
	if difference := w.Score - w.DisplayScore; difference > 0 {
		switch {
		case difference >= 2000:
			w.DisplayScore += 1000
		case difference >= 200:
			w.DisplayScore += 100
		default:
			w.DisplayScore += 10
		}
	} else if difference < 0 {
		w.Score = w.DisplayScore
	}
	backgroundStep := (w.ScrollDelta >> 1) + (int(w.Frame&1) & w.ScrollDelta)
	w.BackgroundY = (w.BackgroundY - backgroundStep + 192) % 192
	w.Player.SpeedTier = w.Equipment.SpeedTier
	w.Player.ScrollStep = w.BaseScrollStep
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
			if actor.Active && actor.ActorList == "moving" && actor.Collision.Intersects(contactRect) {
				if w.Equipment.ShadesFrames == 0 {
					w.damagePlayer(ContactDamage(actor.part.StrongHealth))
				}
				if w.Equipment.ShadesFrames != 0 && actor.firstGuardian {
					w.strikeFirstGuardian(contactRect, 127)
				} else if w.Equipment.ShadesFrames != 0 || actor.part.ResourceTag != 80 && actor.part.ResourceTag != 84 {
					w.damageActor(actor, 127)
				}
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
			handled, crushed = w.Rewind.Advance(&w.Player, w.ScrollY, w.BaseScrollStep, touching)
		}
		if crushed {
			w.destroyPlayer()
		}
		if !handled {
			w.Player.Advance(input.Motion, MotionContext{ScrollY: w.ScrollY, VisitedScrollY: w.VisitedScrollY, BaseScrollStep: w.BaseScrollStep})
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
	copy(w.shipTrail[:3], w.shipTrail[1:])
	w.shipTrail[3] = w.PreviousPlayer
	if err := w.advancePlayerShadows(input); err != nil {
		return err
	}
	w.secondStreamsUpdated = [2]bool{}
	w.thirdMiddleUpdated, w.thirdFinalUpdated = false, false
	w.syncDeadActors()
	if err := w.advanceActorPhase(ActorPoolMoving, input); err != nil {
		return err
	}
	w.releaseDeadPoolEntries(ActorPoolEquipment)
	if input.Fire && !w.previousFire {
		w.fire.QueueTrigger()
	}
	w.previousFire = input.Fire
	w.fire.ApplyEquipment(w.Equipment)
	pulse := w.fire.TakePulse()
	if w.Weapons != nil {
		if err := w.Weapons.AdvanceEquipment(w.weaponContext(input, pulse)); err != nil {
			return err
		}
	}
	if w.Weapons == nil && pulse && w.PlayerAlive && w.Dive.Phase == 0 {
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
					binding, err := w.reserveWorldActor(16, ActorPoolProjectile, false)
					if err != nil {
						return err
					}
					projectile := &WorldSmallShot{ID: binding.EntityID, Binding: binding, PreviousX: shot.X, PreviousY: shot.Y, Shot: shot, Active: true}
					w.poolSmallShots[binding.Slot] = projectile
					w.SmallShots = append([]*WorldSmallShot{projectile}, w.SmallShots...)
				}
			}
		}
	}
	if w.InvulnerableFrames > 0 && w.MaterializationFrames == 0 {
		w.InvulnerableFrames--
	}
	if err := w.advancePooledProjectiles(input); err != nil {
		return err
	}
	// Synthetic diagnostics can supply entries without an allocated binding.
	if w.hasUnboundProjectiles() {
		for _, actor := range w.Actors {
			if !actor.Active || actor.ActorList != "transient" || actor.Binding.EntityID != 0 {
				continue
			}
			actor.PreviousX, actor.PreviousY = actor.X, actor.Y
			if actor.secondFragment != nil {
				w.advanceSecondFragment(actor)
				w.finishActorUpdate(actor)
				continue
			}
			if actor.animation.Ending == "remove" && actor.animationState.Frame == len(actor.animation.Frames)-1 && actor.animationState.Remaining == 1 {
				actor.Active = false
				w.finishActorUpdate(actor)
				continue
			}
			actor.animationState.Advance(actor.animation)
			actor.selectSprite()
			w.finishActorUpdate(actor)
		}
		// Both families belonged to the same newest-first projectile list. Merge
		// their stable creation IDs to preserve hits and removals in that order.
		collectiblesAtStart := w.Collectibles
		w.weaponIDs = w.weaponIDs[:0]
		weaponContext := w.weaponContext(input, false)
		for enemy, player, collectible, weapon := 0, 0, 0, 0; enemy < len(w.Projectiles) || player < len(w.SmallShots) || collectible < len(collectiblesAtStart) || weapon < len(w.weaponIDs); {
			enemyID, playerID, collectibleID, weaponID := -2147483648, -2147483648, -2147483648, -2147483648
			for enemy < len(w.Projectiles) && w.Projectiles[enemy].Binding.EntityID != 0 {
				enemy++
			}
			for player < len(w.SmallShots) && w.SmallShots[player].Binding.EntityID != 0 {
				player++
			}
			for collectible < len(collectiblesAtStart) && collectiblesAtStart[collectible].Binding.EntityID != 0 {
				collectible++
			}
			if enemy >= len(w.Projectiles) && player >= len(w.SmallShots) && collectible >= len(collectiblesAtStart) && weapon >= len(w.weaponIDs) {
				break
			}
			if enemy < len(w.Projectiles) {
				enemyID = w.Projectiles[enemy].ID
			}
			if player < len(w.SmallShots) {
				playerID = w.SmallShots[player].ID
			}
			if collectible < len(collectiblesAtStart) {
				collectibleID = collectiblesAtStart[collectible].ID
				if collectiblesAtStart[collectible].Order != 0 {
					collectibleID = collectiblesAtStart[collectible].Order
				}
			}
			if weapon < len(w.weaponIDs) {
				weaponID = w.weaponIDs[weapon]
			}
			if weaponID > max(enemyID, playerID, collectibleID) {
				weaponContext.ShipDestroyed = !w.PlayerAlive
				for i := range weaponContext.Targets {
					if actor := w.weaponTargetActors[weaponContext.Targets[i].ID]; actor != nil {
						weaponContext.Targets[i].Active = actor.Active
					}
				}
				if err := w.Weapons.AdvanceProjectile(weaponID, weaponContext); err != nil {
					return err
				}
				weapon++
			} else if collectibleID > max(enemyID, playerID) {
				w.advanceCollectible(collectiblesAtStart[collectible])
				w.finishCollectibleUpdate(collectiblesAtStart[collectible])
				collectible++
			} else if enemyID > playerID {
				if err := w.advanceEnemyShot(w.Projectiles[enemy]); err != nil {
					return err
				}
				w.finishProjectileUpdate(w.Projectiles[enemy])
				enemy++
			} else {
				w.advanceSmallShot(w.SmallShots[player])
				w.finishSmallShotUpdate(w.SmallShots[player])
				player++
			}
		}
	}
	if w.Weapons != nil {
		w.Weapons.Compact()
	}
	if err := w.advanceActorPhase(ActorPoolScenery, input); err != nil {
		return err
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
	if w.Weapons != nil {
		if err := w.Weapons.AdvanceSparks(w.weaponContext(input, false)); err != nil {
			return err
		}
		w.Weapons.Compact()
	}
	w.MovingEnemyCount = 0
	for _, actor := range w.Actors {
		if actor.Active && actor.ActorList == "moving" && actor.part.ResourceTag != 80 && actor.part.ResourceTag != 84 && !(actor.part.Linked && actor.leader != nil) {
			w.MovingEnemyCount++
		}
	}
	var spawnErr error
	if err := w.advanceFirstMiddleStage(); err != nil {
		return err
	}
	if err := w.advanceThirdStage(); err != nil {
		return err
	}
	if err := w.advanceSecondDefenseWaves(); err != nil {
		return err
	}
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
	scroll.Advance(w.Player.ScrollStep, w.BaseScrollStep, input.Motion.Down)
	w.ScrollDelta, w.ScrollY = scroll.ActualStep, scroll.Y
	w.MaximumScrollY, w.VisitedScrollY = scroll.Maximum, scroll.Maximum
	w.ScrollDeviationPasses = scroll.DeviationPasses
	w.compactActors()
	if w.poolError != nil {
		return w.poolError
	}
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
	if projectile.turning != nil {
		return w.advanceTurningFixedShot(projectile)
	}
	projectile.PreviousX, projectile.PreviousY = projectile.X, projectile.Y
	if len(projectile.animation.Frames) != 0 {
		projectile.animationState.Advance(projectile.animation)
		projectile.Sprite = projectile.animationState.Sprite(projectile.animation)
	}
	var err error
	projectile.Active, err = projectile.Motion.Advance(w.ScrollDelta)
	if err != nil {
		return err
	}
	projectile.X, projectile.Y = float64(projectile.Motion.X>>16), float64(projectile.Motion.Y>>16)
	if projectile.Active && w.PlayerAlive && w.Dive.Phase == 0 {
		box, ok := w.shotSpriteBoxes[projectile.Sprite]
		if !ok && projectile.Atlas == "fixed" {
			box, ok = w.movingSpriteBoxes[projectile.Sprite]
		}
		if !ok && projectile.Atlas == "guardians" && w.Level.Guardians != nil {
			for _, sprite := range w.Level.Guardians.Atlas.Sprites {
				if sprite.Name == projectile.Sprite && sprite.Collision != nil {
					box, ok = *sprite.Collision, true
					break
				}
			}
		}
		if ok && ActorCollisionRect(box, int(projectile.X), int(projectile.Y)).Intersects(w.playerCollision) {
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
	if w.weaponHitPoint(shot.Shot.X, shot.Shot.Y, shot.Shot.Damage) {
		shot.Active = false
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
			actor := &WorldActor{Atlas: "moving", ActorList: "moving", Active: true, Score: part.Score, part: part, path: path, motion: motion,
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
			if err := w.bindWorldActor(actor); err != nil {
				return err
			}
			if partIndex > 0 {
				w.Pool.unlink(actor.Binding.Slot)
				if err := w.Pool.AttachAfter(actor.Binding.Slot, ActorPoolMoving, actor.ID, int16(part.ResourceTag), group[len(group)-1].Binding.Slot); err != nil {
					return err
				}
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
	if actor.fourthCrawler != nil {
		w.damageFourthCrawler(actor, amount)
		return
	}
	if actor.fifthIndex > 0 {
		w.damageFifthGuardian(actor, amount)
		return
	}
	if actor.fourthIndex > 0 {
		x, y := (actor.Collision.Left+actor.Collision.Right)/2, (actor.Collision.Top+actor.Collision.Bottom)/2
		w.damageFourthGuardian(actor, CollisionRect{Left: x, Top: y, Right: x, Bottom: y}, amount)
		return
	}
	if actor.thirdCannon != nil {
		w.damageThirdCannon(actor, amount)
		return
	}
	if actor.thirdMiddlePart > 0 {
		w.damageThirdMiddle(actor, amount)
		return
	}
	if actor.thirdFinalMember != nil {
		w.damageThirdFinal(actor, amount)
		return
	}
	if actor.firstMiddleFollower != nil {
		w.damageFirstMiddleFollower(actor, amount)
		return
	}
	if actor.fixedTileState != nil {
		w.damageFixedTile(actor, amount)
		return
	}
	if actor.secondNode != nil {
		w.damageSecondNode(actor, amount)
		return
	}
	if actor.secondSegment != nil {
		w.damageSecondSegment(actor, amount)
		return
	}
	if actor.secondGuardian {
		w.damageSecondGuardian(actor, amount)
		return
	}
	if actor.part != nil && actor.part.DamageMode == "block-shot" {
		return
	}
	if actor.part != nil && actor.part.DamageMode == "drop-equipment" {
		actor.Active = false
		w.storeActorResidue(actor)
		w.spawnPickup(actor.CarriedReward, int(actor.X), int(actor.Y))
		w.SoundRequests[2] = "synthesized-effect-17"
		return
	}
	target := actor
	if actor.part != nil && actor.part.DamageMode == "group" && actor.leader != nil {
		target = actor.leader
	}
	result := ApplyEnemyDamage(uint16(target.Health), amount)
	target.Health = int(result.Health)
	target.Flash = true
	if !result.Destroyed {
		return
	}
	target.Active = false
	w.storeActorResidue(target)
	if actor.part != nil && actor.part.DamageMode == "group" {
		var effects [159]*WorldActor
		count := 0
		for _, member := range w.Actors {
			if (member == target || member.leader == target) && (member == target || !member.part.Linked) {
				if count < len(effects) {
					effects[count] = member
					count++
				}
			}
		}
		for _, member := range effects[:count] {
			w.spawnActorDeathEffect(member)
		}
	} else {
		w.spawnActorDeathEffect(target)
	}
	w.Score += target.Score
	if w.WaveBonuses.Defeat(target.WaveToken) {
		w.spawnWaveCash(int(target.X), int(target.Y), target.part.StrongHealth)
	}
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
			w.storeActorResidue(member)
		}
	}
}

func (w *World) spawnEnemyShot(x, y int, shot EnemyShot) {
	if w.Level.Rules == nil {
		return
	}
	binding, err := w.reserveWorldActor(20, ActorPoolProjectile, false)
	if err != nil {
		w.poolError = err
		return
	}
	p := &WorldProjectile{ID: binding.EntityID, Binding: binding, X: float64(x), Y: float64(y), PreviousX: float64(x), PreviousY: float64(y),
		Sprite: w.Level.Rules.DefaultEnemyShot, Atlas: "enemy-shots", Active: true,
		Motion: DirectionalProjectile{X: int32(x) << 16, Y: int32(y) << 16, Direction: shot.Direction, Speed: shot.Speed}}
	w.poolProjectiles[binding.Slot] = p
	w.Projectiles = append([]*WorldProjectile{p}, w.Projectiles...)
}

func (w *World) spawnFixed(record visualassets.FixedEncounter) {
	if w.spawnFourthCrawler(record) {
		return
	}
	if w.Level.Number == 5 && (record.EnemyKind == 5 || record.EnemyKind == 6) && w.fifthMiddleArt != nil {
		if err := w.activateFifthGuardian(record, record.EnemyKind == 6); err != nil {
			w.poolError = err
		}
		return
	}
	if w.Level.Number == 4 && (record.EnemyKind == 2 || record.EnemyKind == 4) && w.fourthMiddleArt != nil {
		if err := w.activateFourthGuardian(record.EnemyKind == 4); err != nil {
			w.poolError = err
		}
		return
	}
	if w.spawnThirdFixed(record) {
		return
	}
	if w.Level.Number == 3 && record.EnemyKind == 3 && w.thirdMiddleArt != nil {
		if err := w.activateThirdMiddle(); err != nil {
			w.poolError = err
		}
		return
	}
	if record.EnemyKind == 0 {
		w.captureCheckpoint(record.TriggerY, record.X, record.Y)
		return
	}
	if w.spawnFixedTile(record) {
		return
	}
	if w.spawnFixedHatch(record) {
		return
	}
	if w.spawnFixedPod(record) {
		return
	}
	kind := w.fixedKinds[record.EnemyKind]
	if kind == nil {
		w.UnimplementedFixedEncounters = append(w.UnimplementedFixedEncounters, record)
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
		part := &visualassets.ActorPart{Atlas: "fixed", StrongHealth: kind.StrongHealth, Score: kind.Score, MotionMode: "world-anchored", DamageMode: "individual"}
		a := &WorldActor{X: float64(record.X + v.OriginOffsetX), Y: float64(record.Y + v.OriginOffsetY - w.ScrollY), Atlas: "fixed", ActorList: kind.ActorList, Active: true, part: part,
			mapY: record.Y + v.OriginOffsetY, fixed: true, Health: kind.Health, Score: kind.Score,
			animation: v.Animation, animationState: NewAnimation(v.Animation)}
		a.fixedKind = kind
		a.fixedState = NewFixedSpriteState(*kind, v, record, w.ScrollY)
		a.PreviousX, a.PreviousY = a.X, a.Y
		a.selectSprite()
		if kind.CollisionMode == "sprite-prefix" {
			if box, ok := w.movingSpriteBoxes[a.Sprite]; ok {
				a.Collision = ActorCollisionRect(box, int(a.X), int(a.Y))
			}
		} else {
			a.Collision = CollisionRect{Right: -1, Bottom: -1}
		}
		if err := w.bindWorldActor(a); err != nil {
			w.poolError = err
			return
		}
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
