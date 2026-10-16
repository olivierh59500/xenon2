package engine

import "xenon2/internal/visualassets"

// Separate chunks keep entity pointers stable when a larger scene needs more
// storage during Load. The records are reusable only after the next Load;
// forecast snapshots must not be retained across that ownership boundary.
type forecastValueArena[T any] struct {
	chunks []*[16]T
	used   int
}

func (a *forecastValueArena[T]) next() *T {
	index := a.used
	a.used++
	if index/16 == len(a.chunks) {
		a.chunks = append(a.chunks, new([16]T))
	}
	return &a.chunks[index/16][index%16]
}

func (a *forecastValueArena[T]) copy(source *T) *T {
	if source == nil {
		return nil
	}
	value := a.next()
	*value = *source
	return value
}

func forecastCopyInto[T any](source, destination *T) *T {
	if source == nil {
		return nil
	}
	*destination = *source
	return destination
}

// Retain buffer ownership separately from the actor value. A later source may
// have no attachment while a following one needs the same storage again.
type forecastActorStorage struct {
	actor        WorldActor
	extras       []WorldSpriteAttachment
	overlays     []WorldTileOverlay
	overlayTiles [][]uint16
	group        []*WorldActor
}

type forecastActorArena struct {
	actors               forecastValueArena[forecastActorStorage]
	parts                forecastValueArena[visualassets.ActorPart]
	bodyRenders          forecastValueArena[fifthBodyRenderState]
	fifthSeeking         forecastValueArena[FifthSeekingState]
	fifthColumns         forecastValueArena[FifthLaserColumnState]
	fifthTiles           forecastValueArena[FifthTileState]
	fifthFormations      forecastValueArena[FifthFormationState]
	fourthCrawlers       forecastValueArena[FourthCrawlerState]
	fourthFalling        forecastValueArena[FourthFallingActor]
	fourthPods           forecastValueArena[FourthPodState]
	fourthChildren       forecastValueArena[FourthPodChild]
	invulnerability      forecastValueArena[InvulnerabilityState]
	fixedAiming          forecastValueArena[AnimatedAimingFixedProjectile]
	fixedTiles           forecastValueArena[FixedTileState]
	fixedHatches         forecastValueArena[FixedHatchState]
	hatchCreatures       forecastValueArena[HatchCreatureState]
	fixedPods            forecastValueArena[FixedPodState]
	podCreatures         forecastValueArena[PodCreatureState]
	firstMiddleAnchors   forecastValueArena[FirstMiddleAnchor]
	firstMiddleFollowers forecastValueArena[FirstMiddleFollower]
	firstMiddleFragments forecastValueArena[FirstMiddleFragment]
	thirdCrawlers        forecastValueArena[ThirdCrawlerState]
	thirdCannons         forecastValueArena[ThirdCannonState]
	thirdFinalMembers    forecastValueArena[ThirdFinalMember]
	secondNodes          forecastValueArena[SecondDefenseNodeState]
	secondSegments       forecastValueArena[SecondDefenseSegment]
	secondFragments      forecastValueArena[SecondDefenseFragment]
	secondMinions        forecastValueArena[SecondMinionState]
	chains               forecastValueArena[ThirdChainState]
	markers              forecastValueArena[[2]*WorldActor]
	firstGuardian        FirstGuardianState
	firstSegments        FirstGuardianSegments
	firstMiddle          FirstMiddleState
	secondGuardian       SecondGuardianState
	secondScheduler      SecondDefenseScheduler
	thirdMiddle          ThirdGuardianState
	thirdFinal           ThirdFinalState
	fourthMiddle         FourthMiddleGuardian
	fourthFinal          FourthFinalGuardian
	fifthMiddle          FifthMiddleGuardianState
	fifthFinal           FifthFinalGuardianState
	secondMinionConfig   SecondMinionConfig
	secondTerrainCells   SecondTerrainCells
	turnPoints           []visualassets.GuardianTurnPoint
	terrainCells         []visualassets.GuardianTerrainCell
	intact               []bool
}

func (a *forecastActorArena) reset() {
	a.actors.used, a.parts.used, a.bodyRenders.used = 0, 0, 0
	a.fifthSeeking.used, a.fifthColumns.used, a.fifthTiles.used, a.fifthFormations.used = 0, 0, 0, 0
	a.fourthCrawlers.used, a.fourthFalling.used, a.fourthPods.used, a.fourthChildren.used = 0, 0, 0, 0
	a.invulnerability.used, a.fixedAiming.used, a.fixedTiles.used, a.fixedHatches.used = 0, 0, 0, 0
	a.hatchCreatures.used, a.fixedPods.used, a.podCreatures.used = 0, 0, 0
	a.firstMiddleAnchors.used, a.firstMiddleFollowers.used, a.firstMiddleFragments.used = 0, 0, 0
	a.thirdCrawlers.used, a.thirdCannons.used, a.thirdFinalMembers.used = 0, 0, 0
	a.secondNodes.used, a.secondSegments.used, a.secondFragments.used, a.secondMinions.used = 0, 0, 0, 0
	a.chains.used, a.markers.used = 0, 0
}

func forecastActorSliceReuse[T any](storage *[]T, source []T) []T {
	// Retain capacity without retaining references from a larger previous
	// snapshot, including entities allocated by its predicted World.Step.
	if len(*storage) > len(source) {
		clear((*storage)[len(source):])
	}
	if cap(*storage) < len(source) || *storage == nil && source != nil {
		*storage = make([]T, len(source))
	} else {
		*storage = (*storage)[:len(source)]
	}
	copy(*storage, source)
	if source == nil {
		return nil
	}
	return *storage
}

// cloneActor installs its copy before following owner/member links. This also
// handles retained pool entries and controller arrays that reference the same
// entity, including cycles through chain members and first-middle markers.
func (c *forecastClone) cloneActor(source *WorldActor) *WorldActor {
	if source == nil {
		return nil
	}
	if actor := c.actors[source]; actor != nil {
		return actor
	}
	if c.actors == nil {
		c.actors = make(map[*WorldActor]*WorldActor)
	}
	storage := c.actorArena.actors.next()
	actor := &storage.actor
	*actor = *source
	c.actors[source] = actor
	if source.part != nil {
		if c.parts == nil {
			c.parts = make(map[*visualassets.ActorPart]*visualassets.ActorPart)
		}
		actor.part = c.parts[source.part]
		if actor.part == nil {
			actor.part = c.actorArena.parts.copy(source.part)
			c.parts[source.part] = actor.part
		}
	}
	actor.Patch = c.clonePatch(source.Patch)
	actor.Extras = forecastActorSliceReuse(&storage.extras, source.Extras)
	actor.TileOverlays = forecastActorSliceReuse(&storage.overlays, source.TileOverlays)
	if len(storage.overlayTiles) < len(source.TileOverlays) {
		storage.overlayTiles = append(storage.overlayTiles, make([][]uint16, len(source.TileOverlays)-len(storage.overlayTiles))...)
	}
	for index := range actor.TileOverlays {
		actor.TileOverlays[index].Patch.Tiles = forecastActorSliceReuse(&storage.overlayTiles[index], source.TileOverlays[index].Patch.Tiles)
	}
	actor.fifthBodyRender = c.actorArena.bodyRenders.copy(source.fifthBodyRender)
	if actor.fifthBodyRender != nil {
		actor.fifthBodyRender.normal = c.clonePatch(source.fifthBodyRender.normal)
		actor.fifthBodyRender.pending = c.clonePatch(source.fifthBodyRender.pending)
	}
	actor.fifthSeeking = c.actorArena.fifthSeeking.copy(source.fifthSeeking)
	actor.fifthColumn = c.actorArena.fifthColumns.copy(source.fifthColumn)
	actor.fifthTile = c.actorArena.fifthTiles.copy(source.fifthTile)
	actor.fifthFormation = c.actorArena.fifthFormations.copy(source.fifthFormation)
	actor.fourthCrawler = c.actorArena.fourthCrawlers.copy(source.fourthCrawler)
	actor.fourthFalling = c.actorArena.fourthFalling.copy(source.fourthFalling)
	actor.fourthPod = c.actorArena.fourthPods.copy(source.fourthPod)
	actor.fourthChild = c.actorArena.fourthChildren.copy(source.fourthChild)
	actor.invulnerability = c.actorArena.invulnerability.copy(source.invulnerability)
	actor.fixedAiming = c.actorArena.fixedAiming.copy(source.fixedAiming)
	actor.fixedTileState = c.actorArena.fixedTiles.copy(source.fixedTileState)
	actor.fixedHatch = c.actorArena.fixedHatches.copy(source.fixedHatch)
	actor.hatchCreature = c.actorArena.hatchCreatures.copy(source.hatchCreature)
	actor.fixedPod = c.actorArena.fixedPods.copy(source.fixedPod)
	actor.podCreature = c.actorArena.podCreatures.copy(source.podCreature)
	actor.firstMiddleAnchor = c.actorArena.firstMiddleAnchors.copy(source.firstMiddleAnchor)
	actor.firstMiddleFollower = c.actorArena.firstMiddleFollowers.copy(source.firstMiddleFollower)
	actor.firstMiddleFragment = c.actorArena.firstMiddleFragments.copy(source.firstMiddleFragment)
	actor.thirdCrawler = c.actorArena.thirdCrawlers.copy(source.thirdCrawler)
	actor.thirdCannon = c.actorArena.thirdCannons.copy(source.thirdCannon)
	actor.thirdFinalMember = c.actorArena.thirdFinalMembers.copy(source.thirdFinalMember)
	actor.secondNode = c.actorArena.secondNodes.copy(source.secondNode)
	actor.secondSegment = c.actorArena.secondSegments.copy(source.secondSegment)
	actor.secondFragment = c.actorArena.secondFragments.copy(source.secondFragment)
	actor.secondMinion = c.actorArena.secondMinions.copy(source.secondMinion)
	if source.thirdChain != nil {
		if c.chains == nil {
			c.chains = make(map[*ThirdChainState]*ThirdChainState)
		}
		actor.thirdChain = c.chains[source.thirdChain]
		if actor.thirdChain == nil {
			actor.thirdChain = c.actorArena.chains.copy(source.thirdChain)
			c.chains[source.thirdChain] = actor.thirdChain
		}
	}
	actor.leader = c.cloneActor(source.leader)
	for index, member := range source.thirdChainMembers {
		actor.thirdChainMembers[index] = c.cloneActor(member)
	}
	actor.fifthTileGroup = forecastActorSliceReuse(&storage.group, source.fifthTileGroup)
	for index, member := range source.fifthTileGroup {
		actor.fifthTileGroup[index] = c.cloneActor(member)
	}
	if source.firstMiddleMarkers != nil {
		if c.markers == nil {
			c.markers = make(map[*[2]*WorldActor]*[2]*WorldActor)
		}
		actor.firstMiddleMarkers = c.markers[source.firstMiddleMarkers]
		if actor.firstMiddleMarkers == nil {
			actor.firstMiddleMarkers = c.actorArena.markers.copy(source.firstMiddleMarkers)
			c.markers[source.firstMiddleMarkers] = actor.firstMiddleMarkers
			for index, marker := range source.firstMiddleMarkers {
				actor.firstMiddleMarkers[index] = c.cloneActor(marker)
			}
		}
	}
	return actor
}

// cloneControllers copies each level's live controller and reconnects retained
// actor references to this forecast's memoized entities. Resource groups and
// image/path/animation descriptors are read-only and may remain shared.
func (c *forecastClone) cloneControllers(source *World) {
	w := c.world
	w.FirstGuardian = forecastCopyInto(source.FirstGuardian, &c.actorArena.firstGuardian)
	w.FirstGuardianSegments = forecastCopyInto(source.FirstGuardianSegments, &c.actorArena.firstSegments)
	w.FirstMiddle = forecastCopyInto(source.FirstMiddle, &c.actorArena.firstMiddle)
	w.SecondGuardian = forecastCopyInto(source.SecondGuardian, &c.actorArena.secondGuardian)
	w.secondScheduler = forecastCopyInto(source.secondScheduler, &c.actorArena.secondScheduler)
	w.ThirdMiddle = forecastCopyInto(source.ThirdMiddle, &c.actorArena.thirdMiddle)
	w.ThirdFinal = forecastCopyInto(source.ThirdFinal, &c.actorArena.thirdFinal)
	w.FourthMiddle = forecastCopyInto(source.FourthMiddle, &c.actorArena.fourthMiddle)
	w.FourthFinal = forecastCopyInto(source.FourthFinal, &c.actorArena.fourthFinal)
	w.FifthMiddle = forecastCopyInto(source.FifthMiddle, &c.actorArena.fifthMiddle)
	w.FifthFinal = forecastCopyInto(source.FifthFinal, &c.actorArena.fifthFinal)
	w.secondMinionConfig = forecastCopyInto(source.secondMinionConfig, &c.actorArena.secondMinionConfig)
	if w.secondMinionConfig != nil {
		w.secondMinionConfig.TurnPoints = forecastActorSliceReuse(&c.actorArena.turnPoints, source.secondMinionConfig.TurnPoints)
	}
	w.secondTerrainCells = forecastCopyInto(source.secondTerrainCells, &c.actorArena.secondTerrainCells)
	if w.secondTerrainCells != nil {
		w.secondTerrainCells.Cells = forecastActorSliceReuse(&c.actorArena.terrainCells, source.secondTerrainCells.Cells)
		w.secondTerrainCells.Intact = forecastActorSliceReuse(&c.actorArena.intact, source.secondTerrainCells.Intact)
	}
	w.firstGuardianActor = c.cloneActor(source.firstGuardianActor)
	w.secondGuardianActor = c.cloneActor(source.secondGuardianActor)
	w.thirdSceneryActor = c.cloneActor(source.thirdSceneryActor)
	for index, actor := range source.firstGuardianParts {
		w.firstGuardianParts[index] = c.cloneActor(actor)
	}
	for index, actor := range source.secondNodes {
		w.secondNodes[index] = c.cloneActor(actor)
	}
	for index, actor := range source.thirdMiddleActors {
		w.thirdMiddleActors[index] = c.cloneActor(actor)
	}
	for index, actor := range source.fourthMiddleActors {
		w.fourthMiddleActors[index] = c.cloneActor(actor)
	}
	for index, actor := range source.fourthFinalActors {
		w.fourthFinalActors[index] = c.cloneActor(actor)
	}
	for index, actor := range source.fifthMiddleActors {
		w.fifthMiddleActors[index] = c.cloneActor(actor)
	}
	for index, actor := range source.fifthFinalActors {
		w.fifthFinalActors[index] = c.cloneActor(actor)
	}
}
