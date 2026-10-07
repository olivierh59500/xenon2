package engine

// advanceActorPhase saves the next physical reference before each callback.
// New heads wait for the next traversal; an already-dead later entry is freed
// when reached, while an actor marking itself dead remains linked this pass.
func (w *World) advanceActorPhase(list ActorPoolList, input Input) error {
	if w.Pool != nil {
		for index := w.Pool.First(list); index != NoActorSlot; {
			next := w.Pool.Next(index)
			slot := w.Pool.Slot(index)
			if slot.ResourceTag == 0 && !slot.allocated {
				break
			}
			if slot.ResourceTag == 4 {
				delete(w.poolBindings, slot.EntityID)
				if err := w.Pool.Release(index); err != nil {
					return err
				}
				w.poolActors[index] = nil
			} else if actor := w.poolActors[index]; actor != nil && actor.Binding.EntityID == slot.EntityID && actor.Active {
				if actor.ActorList == "scenery" {
					if err := w.advanceSceneryActor(actor); err != nil {
						return err
					}
				} else if err := w.advanceMovingActor(actor); err != nil {
					return err
				}
				w.finishActorUpdate(actor)
			} else if err := w.advancePoolProjectileEntity(slot.EntityID, w.weaponContext(input, false)); err != nil {
				return err
			}
			index = next
		}
	}
	// Unbound actors are used by diagnostic scenes and synthetic tests only.
	for _, actor := range w.Actors {
		if !actor.Active || actor.Binding.EntityID != 0 || worldActorPoolList(actor) != list {
			continue
		}
		if list == ActorPoolScenery {
			if err := w.advanceSceneryActor(actor); err != nil {
				return err
			}
		} else if err := w.advanceMovingActor(actor); err != nil {
			return err
		}
		w.finishActorUpdate(actor)
	}
	return nil
}

func (w *World) advanceMovingActor(actor *WorldActor) error {
	if actor.fourthFalling != nil {
		w.advanceFourthFalling(actor)
		return nil
	}
	if actor.fourthChild != nil {
		return w.advanceFourthChild(actor)
	}
	if actor.fifthTile != nil {
		w.advanceFifthTile(actor)
		return w.poolError
	}
	if actor.fourthCrawler != nil {
		return w.advanceFourthCrawler(actor)
	}
	if actor.fifthIndex > 0 {
		if actor.fifthIndex == 1 {
			w.advanceFifthGuardian(actor.fifthFinal)
		}
		return w.poolError
	}
	if actor.fifthSeeking != nil {
		w.advanceFifthSeeking(actor)
		return nil
	}
	if actor.fourthIndex > 0 {
		return w.advanceFourthPart(actor)
	}
	if actor.thirdChainSentinel {
		return nil
	}
	if actor.thirdChainPart > 0 {
		if actor.thirdChainPart == 1 {
			w.advanceThirdChain(actor)
		}
		return nil
	}
	if actor.thirdCrawler != nil {
		w.advanceThirdCrawler(actor)
		return nil
	}
	if actor.thirdCannon != nil {
		w.advanceThirdCannon(actor)
		return nil
	}
	if actor.podCreature != nil {
		w.advancePodCreature(actor)
		return nil
	}
	if actor.hatchCreature != nil {
		w.advanceHatchCreature(actor)
		return nil
	}
	if actor.thirdMiddlePart > 0 {
		if actor.thirdMiddlePart == 1 {
			return w.advanceThirdMiddle()
		}
		return nil
	}
	if actor.thirdFinalMember != nil {
		return w.advanceThirdFinal(actor)
	}
	if actor.firstMiddleSentinel {
		w.advanceFirstMiddleMarker(actor)
		return nil
	}
	if actor.firstMiddleAnchor != nil {
		return w.advanceFirstMiddleAnchor(actor)
	}
	if actor.firstMiddleFollower != nil {
		w.advanceFirstMiddleFollower(actor)
		return nil
	}
	if actor.fixedTileState != nil {
		actor.PreviousX, actor.PreviousY = actor.X, actor.Y
		actor.Visible = true
		w.advanceFixedTile(actor)
		return nil
	}
	if actor.firstSegment > 0 {
		if actor.firstSegment == 1 {
			if err := w.advanceFirstGuardianSegments(); err != nil {
				return err
			}
		}
		return nil
	}
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	actor.Visible = true
	actor.Flash = false
	if actor.secondNode != nil {
		w.advanceSecondNode(actor)
		return nil
	}
	if actor.secondSegment != nil {
		if err := w.advanceSecondSegment(actor); err != nil {
			return err
		}
		return nil
	}
	if actor.secondMinion != nil {
		w.advanceSecondMinion(actor)
		return nil
	}
	if actor.firstGuardian {
		w.advanceFirstGuardian()
		return nil
	}
	if actor.fixedAiming != nil {
		w.advanceFixedAimingActor(actor)
		return nil
	}
	if actor.secondGuardian {
		w.advanceSecondGuardian()
		return nil
	}
	if actor.fixedKind != nil {
		w.advanceFixedSprite(actor)
		return nil
	}
	actor.animationState.Advance(actor.animation)
	if actor.fixed {
		actor.Y = float64(actor.mapY - w.ScrollY)
	} else if actor.part.MotionMode == "follow-leader" {
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
	if actor.Active && !actor.fixed && actor.part.MotionMode != "follow-leader" {
		shot, fired, err := actor.fire.Tick(w.random.Next, w.Player.X-int(actor.X), w.Player.Y-int(actor.Y))
		if err != nil {
			return err
		}
		if fired {
			w.spawnEnemyShot(int(actor.X), int(actor.Y), shot)
		}
	}
	return nil
}

func (w *World) advanceSceneryActor(actor *WorldActor) error {
	if actor.thirdScenery {
		actor.Visible = w.thirdFinalUpdated
		actor.X, actor.Y = 0, float64(-w.ScrollY)
		return nil
	}
	if actor.fixedPod != nil {
		w.advanceFixedPod(actor)
		return nil
	}
	if actor.fixedHatch != nil {
		w.advanceFixedHatch(actor)
		return nil
	}
	actor.PreviousX, actor.PreviousY = actor.X, actor.Y
	actor.Visible = true
	if actor.fixedKind != nil {
		w.advanceFixedSprite(actor)
		return nil
	}
	actor.animationState.Advance(actor.animation)
	actor.Y = float64(actor.mapY - w.ScrollY)
	actor.selectSprite()
	return nil
}
