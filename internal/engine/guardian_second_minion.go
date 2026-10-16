package engine

import (
	"fmt"
	"xenon2/internal/visualassets"
)

type SecondMinionConfig struct {
	Initial, Transform visualassets.ActorAnimation
	TurretAnimations   [8]visualassets.ActorAnimation
	TransformFrame     int
	FireRate           uint8
	ShotSpeed          int
	TurnPoints         []visualassets.GuardianTurnPoint
}

func SecondMinionConfigFromVisual(visual visualassets.GuardianVisual) (SecondMinionConfig, error) {
	config := SecondMinionConfig{TransformFrame: visual.MotionParameters["minion_transform_frame"], FireRate: uint8(visual.MotionParameters["turret_fire_rate"]), ShotSpeed: visual.MotionParameters["turret_shot_speed"], TurnPoints: visual.TurnPoints}
	clips := map[string]visualassets.ActorAnimation{}
	for _, clip := range visual.Animations {
		clips[clip.ID] = clip.Animation
	}
	config.Initial, config.Transform = clips["guardian-hatch-minion"], clips["guardian-minion-transform"]
	if len(config.Initial.Frames) == 0 || len(config.Transform.Frames) == 0 {
		return config, fmt.Errorf("minion artwork is incomplete")
	}
	for heading := range 8 {
		config.TurretAnimations[heading] = clips[fmt.Sprintf("guardian-turret-%d", heading)]
		if len(config.TurretAnimations[heading].Frames) == 0 {
			return config, fmt.Errorf("turret heading artwork is incomplete")
		}
	}
	if config.TransformFrame < 0 || config.TransformFrame >= len(config.Transform.Frames) || len(config.TurnPoints) != 8 {
		return config, fmt.Errorf("minion transformation or route is incomplete")
	}
	return config, nil
}

type SecondMinionState struct {
	X, Y                       int
	Heading                    uint8
	MinionPhase, TargetHeading int
	Turret, Removed            bool
	FireAccumulator            uint8
	Animation                  AnimationState
}

type SecondMinionInput struct {
	Frame                                  uint64
	ScrollY, ScrollDelta, PlayerX, PlayerY int
}
type SecondMinionEvents struct {
	Transform               bool
	TurretState             SecondMinionState
	Shot                    bool
	ShotX, ShotY, ShotSpeed int
	ShotDirection           uint8
	RestoreTerrain          bool
}

func NewSecondGuardianMinion(x, y int, heading uint8, config SecondMinionConfig) SecondMinionState {
	return SecondMinionState{X: x, Y: y, Heading: heading & 7, TargetHeading: -1, Animation: NewAnimation(config.Initial)}
}
func (s SecondMinionState) Clip(config SecondMinionConfig) visualassets.ActorAnimation {
	if s.Turret {
		return config.TurretAnimations[s.Heading&7]
	}
	if s.MinionPhase == 2 {
		return config.Transform
	}
	return config.Initial
}

// Advance preserves the source camera/animation order, gated transformation,
// gradual turns and exact named turning points along the terrain boundary.
func (s *SecondMinionState) Advance(input SecondMinionInput, config SecondMinionConfig, random *RandomState) SecondMinionEvents {
	var event SecondMinionEvents
	if s.Removed {
		return event
	}
	s.Animation.Advance(s.Clip(config))
	s.Y += input.ScrollDelta
	if !s.Turret {
		switch s.MinionPhase {
		case 0:
			dx := [8]int{0, 3, 4, 3, 0, -3, -4, -3}
			dy := [8]int{-4, -3, 0, 3, 4, 3, 0, -3}
			x, y := s.X+dx[s.Heading], s.Y+dy[s.Heading]+input.ScrollY
			if x <= 116 {
				x = 116
				s.MinionPhase, s.Heading = 1, 4
			}
			if x >= 204 {
				x = 204
				s.MinionPhase, s.Heading = 1, 0
			}
			if y <= 108 {
				y = 108
				s.MinionPhase, s.Heading = 1, 6
			}
			if y >= 340 {
				y = 340
				s.MinionPhase, s.Heading = 1, 2
			}
			s.X, s.Y = x, y-input.ScrollY
		case 1:
			if random != nil && random.Next()&7 == 0 {
				s.MinionPhase = 2
				s.Animation = NewAnimation(config.Transform)
			}
		case 2:
			if s.Animation.Frame == config.TransformFrame {
				event.Transform = true
				s.Removed = true
				event.TurretState = SecondMinionState{X: s.X, Y: s.Y, Heading: s.Heading, TargetHeading: -1, Turret: true, Animation: NewAnimation(config.TurretAnimations[s.Heading])}
			}
		}
		return event
	}
	if s.TargetHeading >= 0 {
		if input.Frame&1 == 0 {
			return event
		}
		difference := (s.TargetHeading - int(s.Heading) + 8) & 7
		if difference == 0 {
			s.TargetHeading = -1
		} else {
			if difference > 4 {
				s.Heading = (s.Heading - 1) & 7
			} else {
				s.Heading = (s.Heading + 1) & 7
			}
			s.Animation = NewAnimation(config.TurretAnimations[s.Heading])
			return event
		}
	}
	if s.Y >= 0 && s.Y < 192 {
		sum := int(s.FireAccumulator) + int(config.FireRate)
		s.FireAccumulator = uint8(sum)
		if sum >= 256 && random != nil {
			s.FireAccumulator = uint8(random.Next() & 63)
			event.Shot = true
			event.ShotX, event.ShotY, event.ShotSpeed = s.X, s.Y, config.ShotSpeed
			event.ShotDirection = AimDirection(input.PlayerX-s.X, input.PlayerY-s.Y)
		}
	}
	dx := [8]int{0, 1, 1, 1, 0, -1, -1, -1}
	dy := [8]int{-1, -1, 0, 1, 1, 1, 0, -1}
	s.X += dx[s.Heading]
	s.Y += dy[s.Heading]
	for _, point := range config.TurnPoints {
		if s.X != point.X || s.Y+input.ScrollY != point.WorldY {
			continue
		}
		if random != nil {
			for attempts := 0; attempts < 256; attempts++ {
				choice := int(random.Next() & 3)
				if choice != 0 {
					s.TargetHeading = point.Headings[choice-1]
					break
				}
			}
		}
		return event
	}
	event.RestoreTerrain = true
	return event
}
