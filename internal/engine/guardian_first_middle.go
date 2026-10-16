package engine

import (
	"math/bits"
	"xenon2/internal/visualassets"
)

type FirstMiddleState struct {
	Seeds        [5]uint32
	Updated      [5]bool
	GateCounters [16]int
	Crossed      bool
}

func NewFirstMiddleState() FirstMiddleState {
	return FirstMiddleState{Seeds: [5]uint32{0x45d272ba, 0xaa6a4a56, 0x4aba5d4d, 0xd22512c5, 0x28e51aec}}
}

type FirstMiddleEvents struct {
	Scroll, Maximum int
	Crossed         bool
	Launches        [5]int
	LaunchCount     int
	GateFrames      [16]int
	GateChanged     [16]bool
}

// FirstMiddleAnchor is the invisible path state preceding a visible follower.
// Source list order lets a follower copy the preceding pass's anchor result.
type FirstMiddleAnchor struct {
	Motion  PathMotionState
	Removed bool
}

type FirstMiddleFollower struct {
	X, Y             int32
	AngleFixed       int32
	Remaining        int
	Visible, Removed bool
}

type FirstMiddleFragment struct {
	X, Y    int
	Heading uint8
	Removed bool
}

func (s *FirstMiddleFragment) Advance(sine *[256]int8) {
	if s.Removed {
		return
	}
	x := s.X + (int(sine[uint8(s.Heading+64)]) >> 2)
	y := s.Y + (int(sine[s.Heading]) >> 2)
	if x < 0 || x >= 320 || y < 0 || y >= 192 {
		s.Removed = true
		return
	}
	s.X, s.Y = x, y
}

func NewFirstMiddleAnchor(path *visualassets.Path, launch visualassets.GuardianLaunch, delay, budget, scroll, requestedStep int) (FirstMiddleAnchor, error) {
	motion, err := NewPathMotion(path, PathMotionConfig{Delay: delay, Budget: budget})
	if err != nil {
		return FirstMiddleAnchor{}, err
	}
	motion.X = int32(launch.X) << 16
	motion.Y = int32(launch.WorldY-scroll-requestedStep) << 16
	return FirstMiddleAnchor{Motion: motion}, nil
}

func (s *FirstMiddleAnchor) Advance(path *visualassets.Path, sine *[256]int8, scrollDelta int, random *RandomState) error {
	if s.Removed {
		return nil
	}
	s.Motion.Y += int32(scrollDelta) << 16
	if err := s.Motion.Advance(path, sine, func() uint16 { return uint16(random.Next()) }); err != nil {
		return err
	}
	s.Removed = !s.Motion.Active
	return nil
}

func (s *FirstMiddleFollower) Advance(anchor FirstMiddleAnchor, scrollDelta int) {
	if anchor.Removed {
		s.Removed = true
		return
	}
	s.X, s.Y = (anchor.Motion.X&^0xffff)+(s.X&0xffff), (anchor.Motion.Y&^0xffff)+int32(scrollDelta<<16)+(s.Y&0xffff)
	s.AngleFixed, s.Remaining = anchor.Motion.AngleFixed, anchor.Motion.Remaining
	if !s.Visible && s.Remaining >= 0 {
		s.Visible = true
	}
}

func (s *FirstMiddleState) Advance(scroll, maximum, playerWorldY int, launchGates [16]int) FirstMiddleEvents {
	event := FirstMiddleEvents{Scroll: scroll, Maximum: maximum}
	if !s.Crossed && playerWorldY < 2688 {
		s.Crossed = true
		event.Crossed = true
		event.Scroll, event.Maximum = 2496, 2496
	}
	if event.Scroll >= 2624 && event.Scroll <= 3344 {
		event.Maximum = max(event.Maximum, 3344)
		for stream := range s.Seeds {
			if s.Updated[stream] {
				continue
			}
			s.Updated[stream] = true
			s.Seeds[stream] = bits.RotateLeft32(s.Seeds[stream], -5)
			launch := int(s.Seeds[stream] & 15)
			event.Launches[stream] = launch
			event.LaunchCount++
			gate := launchGates[launch]
			if s.GateCounters[gate] == 0 {
				s.GateCounters[gate] = 20
			}
		}
		for index := range s.GateCounters {
			if s.GateCounters[index] == 0 {
				continue
			}
			s.GateCounters[index]--
			event.GateChanged[index] = true
			event.GateFrames[index] = s.GateCounters[index] & 3
		}
	}
	s.Updated = [5]bool{}
	return event
}
