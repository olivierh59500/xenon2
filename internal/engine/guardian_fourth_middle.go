package engine

import (
	"fmt"
	"xenon2/internal/visualassets"
)

// FourthMiddleGuardian retains the head's lunge clock, its articulated neck,
// the independent tail curve and the five targets that protect its core.
type FourthMiddleGuardian struct {
	Parts        [20]FourthGuardianPart
	OuterTargets int
	Defeated     bool
}

func NewFourthMiddleGuardian(art *visualassets.GuardianGroup, scrollY int, residues []ActorResidue) (FourthMiddleGuardian, error) {
	var state FourthMiddleGuardian
	if art == nil || art.ID != "middle-guardian" || len(art.Components) != 20 {
		return state, fmt.Errorf("invalid fourth middle guardian artwork")
	}
	state.OuterTargets = 5
	for i, component := range art.Components {
		part := &state.Parts[i]
		part.Arc = GuardianArcState{X: int32(component.InitialX) << 16, Y: int32(component.InitialWorldY-scrollY) << 16, AngleFixed: int32(uint8(component.InitialHeading)) << 16, Budget: 3}
		if i < len(residues) {
			part.Arc.X |= int32(residues[i].XFraction)
			part.Arc.Y |= int32(residues[i].YFraction)
			part.FireAccumulator = residues[i].FireAccumulator()
		}
		part.Health, part.Visible, part.Animation = uint16(component.Health), true, NewAnimation(component.Animation)
	}
	return state, nil
}

func fourthFacing(dx, dy int) uint8 {
	direction := AimDirection(dx, dy)
	if direction < 2 {
		return 2
	}
	if direction > 6 {
		return 6
	}
	return direction
}

func (s *FourthMiddleGuardian) Sprite(index int, art *visualassets.GuardianGroup) string {
	component, part := art.Components[index], s.Parts[index]
	if len(component.HeadingFrames) != 0 {
		return component.HeadingFrames[part.Pose]
	}
	if len(component.HeadingAnimations) == 8 && len(component.HeadingAnimations[part.Pose].Frames) != 0 {
		return part.Animation.Sprite(component.HeadingAnimations[part.Pose])
	}
	return part.Animation.Sprite(component.Animation)
}

func (s *FourthMiddleGuardian) copyCurve(index int, sine *[256]int8) {
	s.Parts[index].Arc = s.Parts[index-1].Arc
	s.Parts[index].Arc.Advance(sine)
	s.Parts[index].Counter = int16(100 - s.Parts[index].Arc.Budget)
}

func (s *FourthMiddleGuardian) AdvancePart(index int, art *visualassets.GuardianGroup, input FourthGuardianInput, sine *[256]int8, box func(string) visualassets.CollisionBox, nextRandom func() uint32) (FourthGuardianEvents, error) {
	event := FourthGuardianEvents{MaximumScrollY: input.MaximumScrollY}
	if index < 0 || index >= 20 || art == nil || len(art.Components) != 20 {
		return event, fmt.Errorf("invalid fourth middle component")
	}
	if s.Defeated || s.Parts[index].Disabled {
		return event, nil
	}
	part := &s.Parts[index]
	part.Flash = false
	switch {
	case index == 0:
		part.Arc.Y += int32(input.ScrollDelta) << 16
		advance := part.Counter != 0
		if !advance {
			facing := fourthFacing(input.PlayerX-int(part.Arc.X>>16), input.PlayerY-int(part.Arc.Y>>16))
			part.Arc.AngleFixed = int32(uint8(int(facing)*32-64)) << 16
			part.Arc.AngularVelocity, part.Arc.AngularAcceleration = 0, 0
			if nextRandom == nil {
				return event, fmt.Errorf("fourth head needs the shared random stream")
			}
			if int(uint8(nextRandom())) < art.MotionParameters["head_attack_chance"] {
				part.Direction = 2
				part.Arc.AngleFixed = int32(uint8(int(facing)*32-96)) << 16
				part.Arc.AngularVelocity = int32((nextRandom()&31)+32) << 10
				heading := uint8(uint32(part.Arc.AngleFixed) >> 16)
				if heading < 32 || heading >= 160 {
					part.Arc.AngleFixed = int32(uint8(heading+64)) << 16
					part.Arc.AngularVelocity = -part.Arc.AngularVelocity
				}
				advance = true
			}
		}
		if advance {
			part.Counter += part.Direction
			if part.Counter >= 24 {
				part.Direction = -part.Direction
			}
			part.Arc.Budget = int(part.Counter) + 4
		}
		part.Pose = ((int(uint8(uint32(part.Arc.AngleFixed)>>16)) + 16) >> 5) & 3
	case index >= 1 && index <= 3:
		s.copyCurve(index, sine)
		part.Pose = ((int(uint8(uint32(part.Arc.AngleFixed)>>16)) + 16) >> 5) & 3
	case index == 4:
		if event.MaximumScrollY < 2480 {
			event.MaximumScrollY = 2480
		}
		s.copyCurve(index, sine)
		part.Pose = int(fourthFacing(input.PlayerX-int(part.Arc.X>>16), input.PlayerY-int(part.Arc.Y>>16)))
		part.Direction = int16(part.Pose)
		part.Arc.AngleFixed = part.Arc.AngleFixed&^0xffff | int32(uint16(part.Direction))
	case index == 5:
		animation := art.Components[index].Animation
		if len(art.Components[index].HeadingAnimations) == 8 && len(art.Components[index].HeadingAnimations[part.Pose].Frames) != 0 {
			animation = art.Components[index].HeadingAnimations[part.Pose]
		}
		part.Animation.Advance(animation)
		previous := s.Parts[index-1]
		part.Arc.X = previous.Arc.X&^0xffff | part.Arc.X&0xffff
		part.Arc.Y = previous.Arc.Y&^0xffff | part.Arc.Y&0xffff
		if part.Direction != previous.Direction {
			part.Direction, part.Pose = previous.Direction, int(previous.Direction)
			part.Arc.AngleFixed = part.Arc.AngleFixed&^0xffff | int32(uint16(part.Direction))
			part.Animation = NewAnimation(art.Components[index].HeadingAnimations[part.Pose])
		}
		if fourthFire(&part.FireAccumulator, art.MotionParameters["companion_fire_rate"], nextRandom) {
			d := part.Pose
			xs, ys := art.MotionTables["companion_shot_offset_x"], art.MotionTables["companion_shot_offset_y"]
			event.ShotCount = 1
			event.Shots[0] = FourthGuardianShot{X: int(part.Arc.X>>16) + xs[d], Y: int(part.Arc.Y>>16) + ys[d], Direction: uint8(d), Speed: art.MotionParameters["companion_shot_speed"], Animation: "middle-shot"}
		}
	case index == 6:
		part.Arc.Y += int32(input.ScrollDelta) << 16
		heading, velocity, acceleration := fourthTailCurve(input.Frame)
		part.Arc.AngleFixed = int32(heading) << 16
		part.Arc.AngularVelocity, part.Arc.AngularAcceleration, part.Arc.Budget = velocity, acceleration, 12
	case index >= 7 && index <= 15:
		if index == 15 {
			part.Animation.Advance(art.Components[index].Animation)
		}
		s.copyCurve(index, sine)
		if index == 15 && fourthFire(&part.FireAccumulator, art.MotionParameters["tail_fire_rate"], nextRandom) {
			event.ShotCount = 8
			for shot := range 8 {
				event.Shots[shot] = FourthGuardianShot{X: int(part.Arc.X >> 16), Y: int(part.Arc.Y >> 16), Direction: uint8(7 - shot), Speed: art.MotionParameters["tail_shot_speed"], Animation: "middle-shot"}
			}
		}
	default:
		part.Arc.Y += int32(input.ScrollDelta) << 16
		animation := art.Components[index].Animation
		if len(art.Components[index].HeadingAnimations) == 8 && len(art.Components[index].HeadingAnimations[part.Pose].Frames) != 0 {
			animation = art.Components[index].HeadingAnimations[part.Pose]
		}
		part.Animation.Advance(animation)
		dx, dy := input.PlayerX-int(part.Arc.X>>16), input.PlayerY-int(part.Arc.Y>>16)
		facing := int(AimDirection(dx, dy))
		if int(part.Direction) != facing {
			part.Direction, part.Pose = int16(facing), facing
			part.Arc.AngleFixed = part.Arc.AngleFixed&^0xffff | int32(uint16(facing))
			part.Animation = NewAnimation(art.Components[index].HeadingAnimations[facing])
		}
		if fourthFire(&part.FireAccumulator, art.MotionParameters["satellite_fire_rate"], nextRandom) {
			event.ShotCount = 1
			event.Shots[0] = FourthGuardianShot{X: int(part.Arc.X >> 16), Y: int(part.Arc.Y >> 16), Direction: uint8(facing), Speed: art.MotionParameters["satellite_shot_speed"], Animation: "middle-satellite-shot"}
		}
	}
	part.Collision = ActorCollisionRect(box(s.Sprite(index, art)), int(part.Arc.X>>16), int(part.Arc.Y>>16))
	if index >= 16 {
		part.Collision.Top -= 8
		part.Collision.Bottom += 8
	}
	return event, nil
}

// Strike retains the satellite's two armored border lines and the tail's
// ten-part collapse. The companion forwards core hits to its preceding core.
func (s *FourthMiddleGuardian) Strike(index int, area CollisionRect, damage uint16) FourthGuardianEvents {
	var event FourthGuardianEvents
	if s.Defeated || index < 0 || index >= 20 || s.Parts[index].Disabled {
		return event
	}
	if index == 5 {
		index = 4
	}
	if index == 4 {
		if s.OuterTargets != 0 {
			return event
		}
		part := &s.Parts[4]
		part.Flash = true
		s.Parts[5].Flash = true
		hit := ApplyEnemyDamage(part.Health, damage)
		part.Health = hit.Health
		if !hit.Destroyed {
			return event
		}
		s.Defeated = true
		event.Defeated, event.Score, event.CashPairs, event.ExplosionCount = true, 2000, 5, 40
		event.ClearMoving = true
		event.MinimumScrollY, event.ForcedScrollY, event.ClearTerrainRow, event.ClearTerrainRows = 0, 2208, 143, 16
		event.MaximumScrollY = 2208
		event.ExplosionRectangle = CollisionRect{Left: 0, Top: 0, Right: 319, Bottom: 191}
		for i := range s.Parts {
			s.Parts[i].Disabled = true
			event.RemoveParts[i] = true
		}
		return event
	}
	if index != 15 && index < 16 {
		return event
	}
	part := &s.Parts[index]
	if index >= 16 {
		r := part.Collision
		top, bottom := CollisionRect{Left: r.Left, Top: r.Top, Right: r.Right, Bottom: r.Top}, CollisionRect{Left: r.Left, Top: r.Bottom, Right: r.Right, Bottom: r.Bottom}
		if area.Intersects(top) || area.Intersects(bottom) {
			return event
		}
	}
	part.Flash = true
	hit := ApplyEnemyDamage(part.Health, damage)
	part.Health = hit.Health
	if !hit.Destroyed {
		return event
	}
	s.OuterTargets--
	if index == 15 {
		event.Score, event.ExplosionCount = 1000, 10
		for i := 6; i <= 15; i++ {
			s.Parts[i].Disabled = true
			event.RemoveParts[i] = true
		}
	} else {
		part.Disabled = true
		event.RemoveParts[index] = true
		event.Score, event.ExplosionCount = 300, 1
	}
	return event
}
