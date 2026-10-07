package engine

import (
	"fmt"
	"math/bits"
	"xenon2/internal/visualassets"
)

// SecondDefenseScheduler preserves the two rotated launch sequences. They are
// separate from the shared game random stream and change after two node losses.
type SecondDefenseScheduler struct {
	Seeds        [2]uint32
	InitialWaves int
	DefenseFlags uint8
}

type SecondDefenseLaunch struct{ Stream, LaunchIndex int }

func NewSecondDefenseScheduler() SecondDefenseScheduler {
	return SecondDefenseScheduler{Seeds: [2]uint32{0x45d272ba, 0xaa6a4a56}, InitialWaves: 2}
}
func (s *SecondDefenseScheduler) UseFinalLaunches() { s.Seeds = [2]uint32{0xd25135d3, 0x894d5d53} }

// LaunchIdleStreams is called after that pass's existing segments report which
// streams they updated. Each missing stream obtains its next original path.
func (s *SecondDefenseScheduler) LaunchIdleStreams(updated [2]bool, remaining int) []SecondDefenseLaunch {
	result := make([]SecondDefenseLaunch, 0, 2)
	for stream := range 2 {
		if updated[stream] {
			continue
		}
		secondary := false
		if remaining != 1 {
			if s.InitialWaves == 0 {
				secondary = true
			} else {
				s.InitialWaves--
			}
		}
		s.Seeds[stream] = bits.RotateLeft32(s.Seeds[stream], -5)
		index := int(s.Seeds[stream]&6) + stream
		if secondary {
			index += 8
		}
		s.DefenseFlags |= 1 << uint(stream)
		result = append(result, SecondDefenseLaunch{Stream: stream, LaunchIndex: index})
	}
	return result
}

type SecondDefenseSegment struct {
	Motion         PathMotionState
	Presentation   string
	Stream, GateID int
	Removed        bool
}

type SecondDefenseSegmentEvents struct {
	StreamUpdated bool
	GateOpened    bool
	HeadingFrame  int
}

func NewSecondDefenseSegment(launch visualassets.GuardianLaunch, part visualassets.GuardianComponent, stream, scrollY, requestedScrollStep int) (SecondDefenseSegment, error) {
	motion, err := NewPathMotion(&launch.Path, PathMotionConfig{Delay: -part.InitialDelay, Budget: part.PathBudget})
	if err != nil {
		return SecondDefenseSegment{}, err
	}
	motion.X = int32(launch.X) << 16
	motion.Y = int32(launch.WorldY-requestedScrollStep-scrollY) << 16
	return SecondDefenseSegment{Motion: motion, Stream: stream, GateID: launch.GateID, Presentation: "hidden"}, nil
}

// Advance follows each staggered segment independently, preserving its hidden
// entry, materialization and exit transitions instead of attaching it to a head.
func (s *SecondDefenseSegment) Advance(path *visualassets.Path, sine *[256]int8, scrollDelta int, gateCounters *[8]int, random *RandomState) (SecondDefenseSegmentEvents, error) {
	var event SecondDefenseSegmentEvents
	if s.Removed {
		return event, nil
	}
	event.StreamUpdated = true
	s.Motion.Y += int32(scrollDelta) << 16
	var next func() uint16
	if random != nil {
		next = func() uint16 { return uint16(random.Next()) }
	}
	if err := s.Motion.Advance(path, sine, next); err != nil {
		return event, err
	}
	if !s.Motion.Active {
		s.Removed = true
		if gateCounters != nil && s.GateID >= 1 && s.GateID <= 8 && gateCounters[s.GateID-1] == 0 {
			gateCounters[s.GateID-1] = 24
			event.GateOpened = true
		}
		return event, nil
	}
	if path == nil || s.Motion.ProgramCounter < 0 || s.Motion.ProgramCounter >= len(path.Commands) {
		return event, fmt.Errorf("defense segment leaves its path")
	}
	current := path.Commands[s.Motion.ProgramCounter]
	switch s.Presentation {
	case "hidden":
		if s.Motion.Remaining >= 0 {
			s.Presentation = "materializing"
		}
	case "normal":
		if current.Kind == "end" {
			s.Presentation = "materializing"
		}
	case "materializing":
		if current.Kind != "end" {
			offset := 0
			for _, command := range path.Commands[:s.Motion.ProgramCounter] {
				switch command.Kind {
				case "curve":
					offset += 10
				case "pause", "jump":
					offset += 4
				case "random-branch":
					offset += 18
				case "origin":
					offset += 6
				case "end":
					offset += 2
				}
			}
			if offset != 10 {
				s.Presentation = "normal"
			}
		}
	}
	heading := uint32(s.Motion.AngleFixed) >> 16
	event.HeadingFrame = int((heading+16)>>5) & 7
	return event, nil
}

// SecondDefenseFragment is the source breakup animation's straight drift. Its
// heading is chosen once on impact and keeps the original sine-table rounding.
type SecondDefenseFragment struct {
	X, Y      int
	Heading   uint8
	Animation AnimationState
	Removed   bool
}

func NewSecondDefenseFragment(x, y int, heading uint8, animation visualassets.ActorAnimation) SecondDefenseFragment {
	return SecondDefenseFragment{X: x, Y: y, Heading: heading, Animation: NewAnimation(animation)}
}
func (s *SecondDefenseFragment) Advance(animation visualassets.ActorAnimation, sine *[256]int8) {
	if s.Removed || sine == nil {
		return
	}
	if animation.Ending == "remove" && s.Animation.Frame == len(animation.Frames)-1 && s.Animation.Remaining == 1 {
		s.Animation.Remaining = 0
		s.Removed = true
	} else {
		s.Animation.Advance(animation)
	}
	x, y := s.X+(int(sine[uint8(s.Heading+64)])>>2), s.Y+(int(sine[s.Heading])>>2)
	if x < 0 || x >= 320 || y < 0 || y >= 192 {
		s.Removed = true
		return
	}
	s.X, s.Y = x, y
}

// SecondDefenseSegmentDamage reports the source head flag separately from
// ordinary segment health. World owns score, cash and fragment allocation.
type SecondDefenseSegmentDamage struct {
	Health          uint16
	Destroyed       bool
	DefenseFlags    uint8
	FragmentHeading uint8
}

func DamageSecondDefenseSegment(health, amount uint16, head bool, stream int, flags uint8, random *RandomState) SecondDefenseSegmentDamage {
	damage := ApplyEnemyDamage(health, amount)
	result := SecondDefenseSegmentDamage{Health: damage.Health, Destroyed: damage.Destroyed, DefenseFlags: flags}
	if damage.Destroyed {
		// The breakup callbacks compare before subtracting. A lethal segment
		// retains its old health word while it is replaced by the fragment.
		result.Health = health
		if head && stream >= 0 && stream < 2 {
			result.DefenseFlags &^= 1 << uint(stream)
		}
		if random != nil {
			result.FragmentHeading = uint8(random.Next())
		}
	}
	return result
}
