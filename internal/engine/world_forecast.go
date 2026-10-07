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
type WorldForecast struct {
	world *World
}

type forecastClone struct {
	world        *World
	actors       map[*WorldActor]*WorldActor
	chains       map[*ThirdChainState]*ThirdChainState
	markers      map[*[2]*WorldActor]*[2]*WorldActor
	patches      map[*visualassets.TilePatch]*visualassets.TilePatch
	parts        map[*visualassets.ActorPart]*visualassets.ActorPart
	projectiles  map[*WorldProjectile]*WorldProjectile
	smallShots   map[*WorldSmallShot]*WorldSmallShot
	collectibles map[*WorldCollectible]*WorldCollectible
}

func forecastSliceCopy[T any](source []T) []T {
	if source == nil {
		return nil
	}
	result := make([]T, len(source))
	copy(result, source)
	return result
}

func forecastMapCopy[K comparable, V any](source map[K]V) map[K]V {
	if source == nil {
		return nil
	}
	result := make(map[K]V, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func (c *forecastClone) clonePatch(source *visualassets.TilePatch) *visualassets.TilePatch {
	if source == nil {
		return nil
	}
	if patch, ok := c.patches[source]; ok {
		return patch
	}
	patch := *source
	patch.Tiles = forecastSliceCopy(source.Tiles)
	c.patches[source] = &patch
	return &patch
}

func (c *forecastClone) cloneProjectile(source *WorldProjectile) *WorldProjectile {
	if source == nil {
		return nil
	}
	if result, ok := c.projectiles[source]; ok {
		return result
	}
	result := *source
	c.projectiles[source] = &result
	if source.turning != nil {
		turning := *source.turning
		result.turning = &turning
	}
	return &result
}

func (c *forecastClone) cloneSmallShot(source *WorldSmallShot) *WorldSmallShot {
	if source == nil {
		return nil
	}
	if result, ok := c.smallShots[source]; ok {
		return result
	}
	result := *source
	c.smallShots[source] = &result
	return &result
}

func (c *forecastClone) cloneCollectible(source *WorldCollectible) *WorldCollectible {
	if source == nil {
		return nil
	}
	if result, ok := c.collectibles[source]; ok {
		return result
	}
	result := *source
	c.collectibles[source] = &result
	return &result
}

// Load copies every mutable simulation component. Pointer memoization retains
// shared chain/group ownership and physical-slot references inside the copy.
func (f *WorldForecast) Load(source *World) error {
	if source == nil {
		f.world = nil
		return fmt.Errorf("forecast needs a live world")
	}
	world := *source
	c := forecastClone{
		world:        &world,
		actors:       make(map[*WorldActor]*WorldActor, len(source.Actors)),
		patches:      make(map[*visualassets.TilePatch]*visualassets.TilePatch),
		parts:        make(map[*visualassets.ActorPart]*visualassets.ActorPart),
		projectiles:  make(map[*WorldProjectile]*WorldProjectile, len(source.Projectiles)),
		smallShots:   make(map[*WorldSmallShot]*WorldSmallShot, len(source.SmallShots)),
		collectibles: make(map[*WorldCollectible]*WorldCollectible, len(source.Collectibles)),
	}
	if source.Level.Terrain != nil {
		terrain := *source.Level.Terrain
		terrain.Map = forecastSliceCopy(source.Level.Terrain.Map)
		world.Level.Terrain = &terrain
	}
	if source.Level.InitialEquipment != nil {
		equipment := *source.Level.InitialEquipment
		world.Level.InitialEquipment = &equipment
	}
	if source.Level.InitialRandom != nil {
		random := *source.Level.InitialRandom
		world.Level.InitialRandom = &random
	}
	if source.Coverage != nil {
		coverage := *source.Coverage
		coverage.Map = forecastSliceCopy(source.Coverage.Map)
		world.Coverage = &coverage
		if world.Level.Terrain != nil {
			// Native terrain callbacks and stencil contact share one mutable
			// map. Keep that ownership boundary intact in the forecast.
			world.Level.Terrain.Map = coverage.Map
		}
	}
	world.RenderTerrainMap = forecastSliceCopy(source.RenderTerrainMap)
	world.ActorRenderTerrainMap = forecastSliceCopy(source.ActorRenderTerrainMap)
	if source.Pool != nil {
		pool := *source.Pool
		world.Pool = &pool
	}
	if source.BackgroundStars != nil {
		stars := *source.BackgroundStars
		world.BackgroundStars = &stars
	}
	world.HUD = forecastSliceCopy(source.HUD)
	world.PendingFixedShots = forecastSliceCopy(source.PendingFixedShots)
	world.UnimplementedFixedEncounters = forecastSliceCopy(source.UnimplementedFixedEncounters)
	world.PendingGuardianMinions = forecastSliceCopy(source.PendingGuardianMinions)
	world.fifthDestroyedTurrets = forecastMapCopy(source.fifthDestroyedTurrets)
	world.poolBindings = forecastMapCopy(source.poolBindings)
	world.weaponIDs = forecastSliceCopy(source.weaponIDs)
	world.weaponTargets = forecastSliceCopy(source.weaponTargets)
	world.weaponTargetActors = make(map[int]*WorldActor, len(source.weaponTargetActors))

	world.firstGuardianBody.Tiles = forecastSliceCopy(source.firstGuardianBody.Tiles)
	world.secondGuardianBody.Tiles = forecastSliceCopy(source.secondGuardianBody.Tiles)
	c.patches[&source.firstGuardianBody] = &world.firstGuardianBody
	c.patches[&source.secondGuardianBody] = &world.secondGuardianBody
	c.cloneControllers(source)

	world.Actors = make([]*WorldActor, len(source.Actors))
	for index, actor := range source.Actors {
		world.Actors[index] = c.cloneActor(actor)
	}
	world.Projectiles = make([]*WorldProjectile, len(source.Projectiles))
	for index, projectile := range source.Projectiles {
		world.Projectiles[index] = c.cloneProjectile(projectile)
	}
	world.SmallShots = make([]*WorldSmallShot, len(source.SmallShots))
	for index, shot := range source.SmallShots {
		world.SmallShots[index] = c.cloneSmallShot(shot)
	}
	world.Collectibles = make([]*WorldCollectible, len(source.Collectibles))
	for index, item := range source.Collectibles {
		world.Collectibles[index] = c.cloneCollectible(item)
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
		weapons := *source.Weapons
		weapons.projectiles = forecastSliceCopy(source.Weapons.projectiles)
		// Retained callbacks close over their owning World. Rebuild them even
		// before the first predicted pass can create or release an entity.
		weapons.context = world.weaponContext(Input{}, false)
		weapons.newID = weapons.context.NextID
		world.Weapons = &weapons
	}
	f.world = &world
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
