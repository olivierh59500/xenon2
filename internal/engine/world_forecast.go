package engine

import (
	"fmt"

	"xenon2/internal/visualassets"
)

// ForecastBoundary stops prediction before a scene or player-turn transition.
type ForecastBoundary string

const (
	ForecastRunning       ForecastBoundary = ""
	ForecastReady         ForecastBoundary = "ready"
	ForecastPlayerDeath   ForecastBoundary = "player-death"
	ForecastGameOver      ForecastBoundary = "game-over"
	ForecastShop          ForecastBoundary = "shop"
	ForecastLevelFinished ForecastBoundary = "level-finished"
)

// ForecastResult reports ordinary simulation outcomes without applying them to
// the live world or entering a frontend, shop or player-turn director.
type ForecastResult struct {
	Frame                       uint64
	Player                      PlayerMotionState
	Alive                       bool
	Lives, Shield, Money, Score int
	Boundary                    ForecastBoundary
}

// WorldForecast owns an isolated mutable state graph. Level artwork, path tables
// and animation descriptors remain shared immutable data. Load replaces the
// previous prediction; no rollback or speculative updates touch the live world.
// Do not copy a loaded WorldForecast: its graph points into its own storage.
// Use Load to create an independent branch and retain stable forecast addresses.
type WorldForecast struct {
	world   *World
	storage forecastWorldStorage
	clone   forecastClone
}

// These buffers belong to one forecast. Loading another source invalidates all
// previously returned state references while retaining the allocated capacity.
type forecastWorldStorage struct {
	world                                 World
	terrain                               visualassets.Terrain
	coverage                              TerrainCoverage
	pool                                  ActorPool
	stars                                 BackgroundStarfield
	equipment                             Equipment
	random                                RandomState
	weapons                               WeaponRuntime
	terrainMap, renderMap, actorRenderMap []uint16
	firstBodyTiles, secondBodyTiles       []uint16
	hud                                   []WorldSpriteAttachment
	fixedEvents                           []FixedSpriteEvents
	unsupported                           []visualassets.FixedEncounter
	guardianEvents                        []SecondGuardianEvents
	destroyed                             map[int]bool
	bindings                              map[int]ActorPoolBinding
	targetActors                          map[int]*WorldActor
	weaponIDs                             []int
	weaponTargets                         []WeaponTarget
	weaponProjectiles                     []runtimeWeaponProjectile
	actors                                []*WorldActor
	projectiles                           []*WorldProjectile
	smallShots                            []*WorldSmallShot
	collectibles                          []*WorldCollectible
	stepUnbound                           worldUnboundContinuation
	stepActors                            []*WorldActor
	stepProjectiles                       []*WorldProjectile
	stepSmallShots                        []*WorldSmallShot
	stepCollectibles                      []*WorldCollectible
}

type forecastPatchStorage struct {
	patch visualassets.TilePatch
	tiles []uint16
}

type forecastClone struct {
	world            *World
	actorArena       forecastActorArena
	actors           map[*WorldActor]*WorldActor
	chains           map[*ThirdChainState]*ThirdChainState
	markers          map[*[2]*WorldActor]*[2]*WorldActor
	patches          map[*visualassets.TilePatch]*visualassets.TilePatch
	parts            map[*visualassets.ActorPart]*visualassets.ActorPart
	projectiles      map[*WorldProjectile]*WorldProjectile
	smallShots       map[*WorldSmallShot]*WorldSmallShot
	collectibles     map[*WorldCollectible]*WorldCollectible
	projectileArena  forecastValueArena[WorldProjectile]
	turningArena     forecastValueArena[TurningFixedProjectile]
	smallShotArena   forecastValueArena[WorldSmallShot]
	collectibleArena forecastValueArena[WorldCollectible]
	patchArena       forecastValueArena[forecastPatchStorage]
}

func forecastMapReuse[K comparable, V any](storage *map[K]V, source map[K]V) map[K]V {
	if *storage == nil && source != nil {
		*storage = make(map[K]V, len(source))
	}
	clear(*storage)
	for key, value := range source {
		(*storage)[key] = value
	}
	if source == nil {
		return nil
	}
	return *storage
}

func forecastMemoReset[K comparable, V any](memo *map[K]V, capacity int) {
	if *memo == nil {
		*memo = make(map[K]V, capacity)
	} else {
		clear(*memo)
	}
}

func (c *forecastClone) reset(world *World, source *World) {
	c.world = world
	c.actorArena.reset()
	c.projectileArena.used, c.turningArena.used, c.smallShotArena.used, c.collectibleArena.used, c.patchArena.used = 0, 0, 0, 0, 0
	forecastMemoReset(&c.actors, len(source.Actors))
	forecastMemoReset(&c.chains, 8)
	forecastMemoReset(&c.markers, 8)
	forecastMemoReset(&c.patches, 16)
	forecastMemoReset(&c.parts, len(source.Actors))
	forecastMemoReset(&c.projectiles, len(source.Projectiles))
	forecastMemoReset(&c.smallShots, len(source.SmallShots))
	forecastMemoReset(&c.collectibles, len(source.Collectibles))
}

func (c *forecastClone) clonePatch(source *visualassets.TilePatch) *visualassets.TilePatch {
	if source == nil {
		return nil
	}
	if patch, ok := c.patches[source]; ok {
		return patch
	}
	storage := c.patchArena.next()
	storage.patch = *source
	storage.patch.Tiles = forecastActorSliceReuse(&storage.tiles, source.Tiles)
	c.patches[source] = &storage.patch
	return &storage.patch
}

func (c *forecastClone) cloneProjectile(source *WorldProjectile) *WorldProjectile {
	if source == nil {
		return nil
	}
	if result, ok := c.projectiles[source]; ok {
		return result
	}
	result := c.projectileArena.copy(source)
	c.projectiles[source] = result
	result.turning = c.turningArena.copy(source.turning)
	return result
}

func (c *forecastClone) cloneSmallShot(source *WorldSmallShot) *WorldSmallShot {
	if source == nil {
		return nil
	}
	if result, ok := c.smallShots[source]; ok {
		return result
	}
	result := c.smallShotArena.copy(source)
	c.smallShots[source] = result
	return result
}

func (c *forecastClone) cloneCollectible(source *WorldCollectible) *WorldCollectible {
	if source == nil {
		return nil
	}
	if result, ok := c.collectibles[source]; ok {
		return result
	}
	result := c.collectibleArena.copy(source)
	c.collectibles[source] = result
	return result
}

// Load copies every mutable simulation component into reusable private storage.
// Pointer memoization retains shared ownership and physical-slot references.
// Loading the current predicted State is a no-op: it is already isolated.
func (f *WorldForecast) Load(source *World) error {
	if source == nil {
		f.world = nil
		return fmt.Errorf("forecast needs a live world")
	}
	if source == f.world {
		return nil
	}
	// A shallow World wrapper can still refer to this forecast's mutable arena.
	// Stage that overlapping source before resetting buffers or memo maps.
	if source == &f.storage.world || source.Pool != nil && source.Pool == &f.storage.pool || source.Coverage != nil && source.Coverage == &f.storage.coverage || source.Weapons != nil && source.Weapons == &f.storage.weapons {
		var staged WorldForecast
		if err := staged.Load(source); err != nil {
			return err
		}
		return f.Load(staged.State())
	}
	storage := &f.storage
	world := &storage.world
	*world = *source
	c := &f.clone
	c.reset(world, source)
	if source.Level.Terrain != nil {
		storage.terrain = *source.Level.Terrain
		storage.terrain.Map = forecastActorSliceReuse(&storage.terrainMap, source.Level.Terrain.Map)
		world.Level.Terrain = &storage.terrain
	}
	if source.Level.InitialEquipment != nil {
		storage.equipment = *source.Level.InitialEquipment
		world.Level.InitialEquipment = &storage.equipment
	}
	if source.Level.InitialRandom != nil {
		storage.random = *source.Level.InitialRandom
		world.Level.InitialRandom = &storage.random
	}
	if source.Coverage != nil {
		storage.coverage = *source.Coverage
		storage.coverage.Map = forecastActorSliceReuse(&storage.terrainMap, source.Coverage.Map)
		world.Coverage = &storage.coverage
		if world.Level.Terrain != nil {
			world.Level.Terrain.Map = storage.coverage.Map
		}
	}
	world.RenderTerrainMap = forecastActorSliceReuse(&storage.renderMap, source.RenderTerrainMap)
	world.ActorRenderTerrainMap = forecastActorSliceReuse(&storage.actorRenderMap, source.ActorRenderTerrainMap)
	if source.Pool != nil {
		storage.pool = *source.Pool
		if source.Pool.shared != nil {
			storage.pool.actorPoolStorage = *source.Pool.storage()
			storage.pool.shared = nil
		}
		world.Pool = &storage.pool
	}
	if source.BackgroundStars != nil {
		storage.stars = *source.BackgroundStars
		world.BackgroundStars = &storage.stars
	}
	world.HUD = forecastActorSliceReuse(&storage.hud, source.HUD)
	world.PendingFixedShots = forecastActorSliceReuse(&storage.fixedEvents, source.PendingFixedShots)
	world.UnimplementedFixedEncounters = forecastActorSliceReuse(&storage.unsupported, source.UnimplementedFixedEncounters)
	world.PendingGuardianMinions = forecastActorSliceReuse(&storage.guardianEvents, source.PendingGuardianMinions)
	world.fifthDestroyedTurrets = forecastMapReuse(&storage.destroyed, source.fifthDestroyedTurrets)
	world.poolBindings = forecastMapReuse(&storage.bindings, source.poolBindings)
	world.weaponIDs = forecastActorSliceReuse(&storage.weaponIDs, source.weaponIDs)
	world.weaponTargets = forecastActorSliceReuse(&storage.weaponTargets, source.weaponTargets)
	forecastMemoReset(&storage.targetActors, len(source.weaponTargetActors))
	world.weaponTargetActors = storage.targetActors
	world.firstGuardianBody.Tiles = forecastActorSliceReuse(&storage.firstBodyTiles, source.firstGuardianBody.Tiles)
	world.secondGuardianBody.Tiles = forecastActorSliceReuse(&storage.secondBodyTiles, source.secondGuardianBody.Tiles)
	c.patches[&source.firstGuardianBody] = &world.firstGuardianBody
	c.patches[&source.secondGuardianBody] = &world.secondGuardianBody
	c.cloneControllers(source)
	world.Actors = forecastActorSliceReuse(&storage.actors, source.Actors)
	for index, actor := range source.Actors {
		world.Actors[index] = c.cloneActor(actor)
	}
	world.Projectiles = forecastActorSliceReuse(&storage.projectiles, source.Projectiles)
	for index, projectile := range source.Projectiles {
		world.Projectiles[index] = c.cloneProjectile(projectile)
	}
	world.SmallShots = forecastActorSliceReuse(&storage.smallShots, source.SmallShots)
	for index, shot := range source.SmallShots {
		world.SmallShots[index] = c.cloneSmallShot(shot)
	}
	world.Collectibles = forecastActorSliceReuse(&storage.collectibles, source.Collectibles)
	for index, item := range source.Collectibles {
		world.Collectibles[index] = c.cloneCollectible(item)
	}
	world.stepContinuation.unbound = nil
	if pending := source.stepContinuation.unbound; pending != nil {
		storage.stepUnbound = *pending
		storage.stepUnbound.actors = forecastActorSliceReuse(&storage.stepActors, pending.actors)
		for index, actor := range pending.actors {
			storage.stepUnbound.actors[index] = c.cloneActor(actor)
		}
		storage.stepUnbound.projectiles = forecastActorSliceReuse(&storage.stepProjectiles, pending.projectiles)
		for index, shot := range pending.projectiles {
			storage.stepUnbound.projectiles[index] = c.cloneProjectile(shot)
		}
		storage.stepUnbound.smallShots = forecastActorSliceReuse(&storage.stepSmallShots, pending.smallShots)
		for index, shot := range pending.smallShots {
			storage.stepUnbound.smallShots[index] = c.cloneSmallShot(shot)
		}
		storage.stepUnbound.collectibles = forecastActorSliceReuse(&storage.stepCollectibles, pending.collectibles)
		for index, item := range pending.collectibles {
			storage.stepUnbound.collectibles[index] = c.cloneCollectible(item)
		}
		world.stepContinuation.unbound = &storage.stepUnbound
	}
	for index := range ActorPoolCapacity {
		world.poolActors[index] = c.cloneActor(source.poolActors[index])
		world.poolProjectiles[index] = c.cloneProjectile(source.poolProjectiles[index])
		world.poolSmallShots[index] = c.cloneSmallShot(source.poolSmallShots[index])
		world.poolCollectibles[index] = c.cloneCollectible(source.poolCollectibles[index])
	}
	for id, actor := range source.weaponTargetActors {
		world.weaponTargetActors[id] = c.cloneActor(actor)
	}
	if source.Weapons != nil {
		callbacks := storage.weapons.context
		storage.weapons = *source.Weapons
		storage.weapons.projectiles = forecastActorSliceReuse(&storage.weaponProjectiles, source.Weapons.projectiles)
		// These callbacks address the forecast's stable World record. Retain
		// only its own binding; source callbacks must never reach this branch.
		storage.weapons.context = WeaponContext{}
		if callbacks.Equipment == &world.Equipment {
			storage.weapons.context = callbacks
		}
		world.Weapons = &storage.weapons
		storage.weapons.context = world.weaponContext(Input{}, false)
		storage.weapons.context.ObservePointImpact = nil
		storage.weapons.context.ObserveRectImpact = nil
		storage.weapons.newID = storage.weapons.context.NextID
	}
	f.world = world
	return nil
}

// State returns the predicted world for read-only scoring and inspection.
// Callers must never modify it or retain it across Load.
func (f *WorldForecast) State() *World { return f.world }

func forecastBoundary(world *World) ForecastBoundary {
	if world.GameOver {
		return ForecastGameOver
	}
	if !world.PlayerAlive {
		return ForecastPlayerDeath
	}
	if world.Ready {
		return ForecastReady
	}
	if world.ShopReady {
		return ForecastShop
	}
	if world.LevelFinished {
		return ForecastLevelFinished
	}
	return ForecastRunning
}

func (f *WorldForecast) result() ForecastResult {
	world := f.world
	return ForecastResult{Frame: world.Frame, Player: world.Player, Alive: world.PlayerAlive,
		Lives: world.Equipment.Lives, Shield: world.Equipment.Shield,
		Money: world.Money, Score: world.Score, Boundary: forecastBoundary(world)}
}

// AdvancePALTick preserves display-timed gameplay effects independently of the
// logic cadence. It never drives an audio device or a frontend transition.
func (f *WorldForecast) AdvancePALTick() {
	if f.world != nil && forecastBoundary(f.world) == ForecastRunning {
		f.world.AdvancePALTick()
	}
}

// Advance applies one ordinary gameplay pass to the isolated state. A lifecycle
// boundary is reported without entering a menu or advancing another player.
func (f *WorldForecast) Advance(input Input) (ForecastResult, error) {
	if f.world == nil {
		return ForecastResult{}, fmt.Errorf("forecast has not been loaded")
	}
	if forecastBoundary(f.world) == ForecastRunning {
		if err := f.world.Step(input); err != nil {
			return f.result(), err
		}
	}
	return f.result(), nil
}

// AdvanceObserved applies one isolated pass and reports point queries before
// their damage callbacks. The observer may inspect State but must not modify it.
// It is cleared on return and never becomes part of a loaded forecast.
func (f *WorldForecast) AdvanceObserved(input Input, observer func(WeaponPointImpact)) (ForecastResult, error) {
	if f.world == nil {
		return ForecastResult{}, fmt.Errorf("forecast has not been loaded")
	}
	if f.world.Weapons == nil {
		return f.result(), fmt.Errorf("point observation requires a weapon runtime")
	}
	f.world.Weapons.context.ObservePointImpact = observer
	f.world.Weapons.context.ObserveRectImpact = nil
	defer func() {
		f.world.Weapons.context.ObservePointImpact, f.world.Weapons.context.ObserveRectImpact = nil, nil
	}()
	return f.Advance(input)
}
