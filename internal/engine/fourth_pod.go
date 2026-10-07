package engine

import "xenon2/internal/visualassets"

type FourthPodState struct {
	X, Y, Side int
	Animation  AnimationState
	Converted  bool
}

type FourthPodEvents struct {
	SpawnChild, ConvertToExplosion          bool
	ChildX, ChildY, ChildHeading, ChildPath int
	ExplosionX, ExplosionY                  int
}

func NewFourthPod(side, x, y int, art *visualassets.FourthStageArt) FourthPodState {
	return FourthPodState{X: x, Y: y, Side: side, Animation: NewAnimation(art.Variants[side*2].PodAnimation)}
}

func (s *FourthPodState) Advance(art *visualassets.FourthStageArt, scrollDelta int, image func(string) visualassets.SpriteRegion) FourthPodEvents {
	var event FourthPodEvents
	if s.Converted {
		return event
	}
	s.Y += scrollDelta
	animation := art.Variants[s.Side*2].PodAnimation
	s.Animation.Advance(animation)
	if s.Animation.Remaining == 0 {
		region := image(s.Animation.Sprite(animation))
		event.ConvertToExplosion = true
		event.ExplosionX = s.X - region.AnchorX + region.Width/2
		event.ExplosionY = s.Y - region.AnchorY + (region.Height-1)/2
		s.Converted = true
		return event
	}
	if s.Animation.Frame == art.PodTransformFrames[s.Side] {
		event.SpawnChild = true
		event.ChildX, event.ChildY = s.X+art.ChildInitialXOffset[s.Side], s.Y
		event.ChildHeading = art.ChildInitialHeading[s.Side]
		event.ChildPath = s.Side
	}
	return event
}

type FourthPodChild struct {
	Motion                  PathMotionState
	PathIndex, HeadingFrame int
	Animation               AnimationState
	FireRate                uint8
	Collision               CollisionRect
	Rerouted                bool
}

func NewFourthPodChild(side, x, y int, art *visualassets.FourthStageArt, residue ActorResidue) (FourthPodChild, error) {
	index := side
	path := &art.PodChildPaths[index]
	motion, err := NewPathMotion(path, PathMotionConfig{Budget: art.ChildBudget})
	if err != nil {
		return FourthPodChild{}, err
	}
	motion.X = int32(x)<<16 | int32(residue.XFraction)
	motion.Y = int32(y)<<16 | int32(residue.YFraction)
	motion.AngleFixed = int32(art.ChildInitialHeading[side]) << 16
	heading := art.ChildInitialHeading[side] >> 5
	return FourthPodChild{Motion: motion, PathIndex: index, HeadingFrame: heading, Animation: NewAnimation(art.PodChildAnimation[heading]), Collision: CollisionRect{Left: 1000, Right: 1000}}, nil
}

func (s *FourthPodChild) Sprite(art *visualassets.FourthStageArt) string {
	return s.Animation.Sprite(art.PodChildAnimation[s.HeadingFrame])
}

func (s *FourthPodChild) Advance(art *visualassets.FourthStageArt, sine *[256]int8, nextRandom func() uint32, box func(string) visualassets.CollisionBox) error {
	s.Rerouted = false
	previousHeading := uint8(uint32(s.Motion.AngleFixed) >> 16)
	s.Animation.Advance(art.PodChildAnimation[s.HeadingFrame])
	if err := s.Motion.Advance(&art.PodChildPaths[s.PathIndex], sine, func() uint16 { return uint16(nextRandom()) }); err != nil {
		return err
	}
	if s.Motion.Active {
		s.Collision = ActorCollisionRect(box(s.Sprite(art)), int(s.Motion.X>>16), int(s.Motion.Y>>16))
	} else {
		s.Rerouted = true
		choice := int(nextRandom()&8) / 4
		if s.Motion.X>>16 >= 160 {
			choice++
		}
		s.PathIndex = choice
		s.Motion.PathID = art.PodChildPaths[choice].ID
		s.Motion.ProgramCounter = 0
		s.Motion.Remaining = 0
		s.Motion.Active = true
	}
	current := uint8(uint32(s.Motion.AngleFixed) >> 16)
	if ((int(previousHeading)+16)>>5)&7 != ((int(current)+16)>>5)&7 {
		s.HeadingFrame = ((int(current) + 16) >> 5) & 7
		s.Animation = NewAnimation(art.PodChildAnimation[s.HeadingFrame])
		s.Collision = ActorCollisionRect(box(s.Sprite(art)), int(s.Motion.X>>16), int(s.Motion.Y>>16))
	}
	return nil
}
