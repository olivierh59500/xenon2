package engine

import "xenon2/internal/visualassets"

// forecastCopyValue is for controller records whose mutable state is entirely
// inline. Animation and path descriptors inside those records remain immutable.
func forecastCopyValue[T any](source *T) *T {
	if source == nil {
		return nil
	}
	copy := *source
	return &copy
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
	actor := *source
	c.actors[source] = &actor
	if source.part != nil {
		if c.parts == nil {
			c.parts = make(map[*visualassets.ActorPart]*visualassets.ActorPart)
		}
		actor.part = c.parts[source.part]
		if actor.part == nil {
			actor.part = forecastCopyValue(source.part)
			c.parts[source.part] = actor.part
		}
	}
	actor.Patch = c.clonePatch(source.Patch)
	actor.Extras = append([]WorldSpriteAttachment(nil), source.Extras...)
	actor.TileOverlays = append([]WorldTileOverlay(nil), source.TileOverlays...)
	for index := range actor.TileOverlays {
		actor.TileOverlays[index].Patch.Tiles = append([]uint16(nil), source.TileOverlays[index].Patch.Tiles...)
	}
	actor.fifthBodyRender = forecastCopyValue(source.fifthBodyRender)
	if actor.fifthBodyRender != nil {
		actor.fifthBodyRender.normal = c.clonePatch(source.fifthBodyRender.normal)
		actor.fifthBodyRender.pending = c.clonePatch(source.fifthBodyRender.pending)
	}
	actor.fifthSeeking = forecastCopyValue(source.fifthSeeking)
	actor.fifthColumn = forecastCopyValue(source.fifthColumn)
	actor.fifthTile = forecastCopyValue(source.fifthTile)
	actor.fifthFormation = forecastCopyValue(source.fifthFormation)
	actor.fourthCrawler = forecastCopyValue(source.fourthCrawler)
	actor.fourthFalling = forecastCopyValue(source.fourthFalling)
	actor.fourthPod = forecastCopyValue(source.fourthPod)
	actor.fourthChild = forecastCopyValue(source.fourthChild)
	actor.invulnerability = forecastCopyValue(source.invulnerability)
	actor.fixedAiming = forecastCopyValue(source.fixedAiming)
	actor.fixedTileState = forecastCopyValue(source.fixedTileState)
	actor.fixedHatch = forecastCopyValue(source.fixedHatch)
	actor.hatchCreature = forecastCopyValue(source.hatchCreature)
	actor.fixedPod = forecastCopyValue(source.fixedPod)
	actor.podCreature = forecastCopyValue(source.podCreature)
	actor.firstMiddleAnchor = forecastCopyValue(source.firstMiddleAnchor)
	actor.firstMiddleFollower = forecastCopyValue(source.firstMiddleFollower)
	actor.firstMiddleFragment = forecastCopyValue(source.firstMiddleFragment)
	actor.thirdCrawler = forecastCopyValue(source.thirdCrawler)
	actor.thirdCannon = forecastCopyValue(source.thirdCannon)
	actor.thirdFinalMember = forecastCopyValue(source.thirdFinalMember)
	actor.secondNode = forecastCopyValue(source.secondNode)
	actor.secondSegment = forecastCopyValue(source.secondSegment)
	actor.secondFragment = forecastCopyValue(source.secondFragment)
	actor.secondMinion = forecastCopyValue(source.secondMinion)
	if source.thirdChain != nil {
		if c.chains == nil {
			c.chains = make(map[*ThirdChainState]*ThirdChainState)
		}
		actor.thirdChain = c.chains[source.thirdChain]
		if actor.thirdChain == nil {
			actor.thirdChain = forecastCopyValue(source.thirdChain)
			c.chains[source.thirdChain] = actor.thirdChain
		}
	}
	actor.leader = c.cloneActor(source.leader)
	for index, member := range source.thirdChainMembers {
		actor.thirdChainMembers[index] = c.cloneActor(member)
	}
	actor.fifthTileGroup = make([]*WorldActor, len(source.fifthTileGroup))
	if source.fifthTileGroup == nil {
		actor.fifthTileGroup = nil
	}
	for index, member := range source.fifthTileGroup {
		actor.fifthTileGroup[index] = c.cloneActor(member)
	}
	if source.firstMiddleMarkers != nil {
		if c.markers == nil {
			c.markers = make(map[*[2]*WorldActor]*[2]*WorldActor)
		}
		actor.firstMiddleMarkers = c.markers[source.firstMiddleMarkers]
		if actor.firstMiddleMarkers == nil {
			actor.firstMiddleMarkers = &[2]*WorldActor{}
			c.markers[source.firstMiddleMarkers] = actor.firstMiddleMarkers
			for index, marker := range source.firstMiddleMarkers {
				actor.firstMiddleMarkers[index] = c.cloneActor(marker)
			}
		}
	}
	return &actor
}

// cloneControllers copies each level's live controller and reconnects retained
// actor references to this forecast's memoized entities. Resource groups and
// image/path/animation descriptors are read-only and may remain shared.
func (c *forecastClone) cloneControllers(source *World) {
	w := c.world
	w.FirstGuardian = forecastCopyValue(source.FirstGuardian)
	w.FirstGuardianSegments = forecastCopyValue(source.FirstGuardianSegments)
	w.FirstMiddle = forecastCopyValue(source.FirstMiddle)
	w.SecondGuardian = forecastCopyValue(source.SecondGuardian)
	w.secondScheduler = forecastCopyValue(source.secondScheduler)
	w.ThirdMiddle = forecastCopyValue(source.ThirdMiddle)
	w.ThirdFinal = forecastCopyValue(source.ThirdFinal)
	w.FourthMiddle = forecastCopyValue(source.FourthMiddle)
	w.FourthFinal = forecastCopyValue(source.FourthFinal)
	w.FifthMiddle = forecastCopyValue(source.FifthMiddle)
	w.FifthFinal = forecastCopyValue(source.FifthFinal)
	w.secondMinionConfig = forecastCopyValue(source.secondMinionConfig)
	if w.secondMinionConfig != nil {
		w.secondMinionConfig.TurnPoints = append([]visualassets.GuardianTurnPoint(nil), source.secondMinionConfig.TurnPoints...)
	}
	w.secondTerrainCells = forecastCopyValue(source.secondTerrainCells)
	if w.secondTerrainCells != nil {
		w.secondTerrainCells.Cells = append([]visualassets.GuardianTerrainCell(nil), source.secondTerrainCells.Cells...)
		w.secondTerrainCells.Intact = append([]bool(nil), source.secondTerrainCells.Intact...)
	}
	w.firstGuardianActor = c.cloneActor(source.firstGuardianActor)
	w.secondGuardianActor = c.cloneActor(source.secondGuardianActor)
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
