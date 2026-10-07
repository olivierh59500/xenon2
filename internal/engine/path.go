package engine

import (
	"fmt"

	"xenon2/internal/visualassets"
)

// PathMotionConfig describes a member of a formation. Delay is measured in
// path substeps; Budget is the number of substeps available per gameplay pass.
type PathMotionConfig struct {
	StartXOffset int
	Delay        int
	Budget       int
}

// PathMotionState uses independent, named game state. X and Y are 16.16
// fixed-point positions. AngleFixed contains an eight-bit heading and sixteen
// fractional bits; angular velocity and acceleration retain their fractions.
type PathMotionState struct {
	PathID              int
	ProgramCounter      int
	X, Y                int32
	AngleFixed          int32
	AngularVelocity     int32
	AngularAcceleration int32
	Remaining           int
	Budget              int
	Active              bool
}

// NewPathMotion initializes a path without retaining its original byte layout.
// Origin is applied only when it is the first command, matching wave creation.
func NewPathMotion(path *visualassets.Path, config PathMotionConfig) (PathMotionState, error) {
	if err := ValidatePath(path); err != nil {
		return PathMotionState{}, err
	}
	if config.Delay < 0 || config.Delay > 32768 || config.Budget < 0 || config.Budget > 32767 {
		return PathMotionState{}, fmt.Errorf("invalid path delay or movement budget")
	}
	state := PathMotionState{PathID: path.ID, X: int32(config.StartXOffset) << 16, Remaining: -config.Delay, Budget: config.Budget, Active: true}
	if first := path.Commands[0]; first.Kind == "origin" {
		state.X = int32(first.X+config.StartXOffset) << 16
		state.Y = int32(first.Y) << 16
		state.ProgramCounter = 1
	}
	return state, nil
}

// ValidatePath rejects incomplete commands and branches outside their path.
// It does not replace intentional pauses, loops or random choices with curves.
func ValidatePath(path *visualassets.Path) error {
	if path == nil || len(path.Commands) == 0 {
		return fmt.Errorf("path has no commands")
	}
	for i, command := range path.Commands {
		switch command.Kind {
		case "end", "origin":
		case "curve":
			if command.Heading < 0 || command.Heading > 255 || command.Duration < 0 || command.Duration > 32767 {
				return fmt.Errorf("path %d curve %d has invalid heading or duration", path.ID, i)
			}
		case "pause":
			if command.Duration < 0 || command.Duration > 32767 {
				return fmt.Errorf("path %d pause %d has invalid duration", path.ID, i)
			}
		case "jump":
			if command.Target < 0 || command.Target >= len(path.Commands) {
				return fmt.Errorf("path %d jump %d leaves its commands", path.ID, i)
			}
		case "random-branch":
			if len(command.Targets) != 8 {
				return fmt.Errorf("path %d random branch %d needs eight candidates", path.ID, i)
			}
			available := false
			for _, target := range command.Targets {
				if target < -1 || target >= len(path.Commands) {
					return fmt.Errorf("path %d random branch %d leaves its commands", path.ID, i)
				}
				available = available || target >= 0
			}
			if !available {
				return fmt.Errorf("path %d random branch %d has no candidate", path.ID, i)
			}
		default:
			return fmt.Errorf("path %d command %d has unknown kind %q", path.ID, i, command.Kind)
		}
	}
	return nil
}

// Advance consumes one gameplay pass while preserving substep and branch
// timing. nextRandom supplies the same shared random sequence as the world.
func (s *PathMotionState) Advance(path *visualassets.Path, sine *[256]int8, nextRandom func() uint16) error {
	if !s.Active || s.Budget == 0 {
		return nil
	}
	if path == nil || sine == nil || path.ID != s.PathID {
		return fmt.Errorf("path state does not match its motion data")
	}
	budget := s.Budget
	if s.Remaining < 0 {
		budget += s.Remaining
		if budget <= 0 {
			s.Remaining = budget
			return nil
		}
	}

	// Valid level paths spend their budget on motion or pauses. The transition
	// bound prevents malformed zero-duration loops from hanging the game.
	for transitions := 0; transitions < 4096; transitions++ {
		if s.Remaining > 0 {
			for budget > 0 && s.Remaining > 0 {
				heading := uint8(uint32(s.AngleFixed) >> 16)
				s.X += int32(sine[uint8(heading+64)]) << 10
				s.Y += int32(sine[heading]) << 10
				s.AngleFixed = (s.AngleFixed + s.AngularVelocity) & 0x00ffffff
				s.AngularVelocity += s.AngularAcceleration
				s.Remaining--
				budget--
			}
			if budget == 0 {
				return nil
			}
			// The native segment expires before fetching the next command.
			// Retaining this value also preserves the final despawn state.
			s.Remaining = -1
		}
		if s.ProgramCounter < 0 || s.ProgramCounter >= len(path.Commands) {
			return fmt.Errorf("path %d ran beyond its commands", s.PathID)
		}
		command := path.Commands[s.ProgramCounter]
		switch command.Kind {
		case "end":
			s.Active = false
			return nil
		case "origin":
			s.ProgramCounter++
		case "curve":
			s.AngleFixed = int32(command.Heading) << 16
			s.AngularVelocity = int32(int16(command.AngularVelocity)) << 8
			s.AngularAcceleration = int32(int16(command.AngularAcceleration))
			s.Remaining = command.Duration
			s.ProgramCounter++
			if s.Remaining == 0 {
				s.Remaining = -1
			}
		case "pause":
			budget -= command.Duration
			s.ProgramCounter++
			if budget <= 0 {
				s.Remaining = budget
				return nil
			}
		case "jump":
			s.ProgramCounter = command.Target
		case "random-branch":
			if nextRandom == nil {
				return fmt.Errorf("path %d needs its world random sequence", s.PathID)
			}
			chosen := false
			for attempts := 0; attempts < 256; attempts++ {
				index := int(nextRandom()&14) / 2
				if index >= len(command.Targets) {
					return fmt.Errorf("path %d random branch has missing candidates", s.PathID)
				}
				if target := command.Targets[index]; target >= 0 {
					s.ProgramCounter = target
					chosen = true
					break
				}
			}
			if !chosen {
				return fmt.Errorf("path %d random source repeatedly chose unavailable candidates", s.PathID)
			}
		default:
			return fmt.Errorf("path %d has unknown command kind %q", s.PathID, command.Kind)
		}
	}
	return fmt.Errorf("path %d contains a non-progressing command loop", s.PathID)
}
