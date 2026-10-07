package engine

import "fmt"

// FireCadence separates queued trigger pulses from the end-of-pass firing
// clock. Weapons consume Pending before Tick runs, preserving the original
// first press and subsequent autofire timing.
type FireCadence struct {
	Remaining int
	Period    int
	Advance   int
	Pending   bool
}

func NewFireCadence(e Equipment) FireCadence {
	return FireCadence{Remaining: e.FirePeriod, Period: e.FirePeriod, Advance: e.FireAdvance}
}

func (s *FireCadence) ApplyEquipment(e Equipment) {
	s.Period, s.Advance = e.FirePeriod, e.FireAdvance
}

func (s *FireCadence) QueueTrigger() {
	s.Pending = true
}

func (s *FireCadence) TakePulse() bool {
	pulse := s.Pending
	s.Pending = false
	return pulse
}

// Tick is called after this pass's weapon updates. Releasing the trigger resets
// the clock. A threshold crossing adds the period once and queues the next pass.
func (s *FireCadence) Tick(held bool) error {
	if s.Period <= 0 || s.Advance < 0 {
		return fmt.Errorf("invalid firing period or advance")
	}
	if !held {
		s.Remaining = s.Period
		return nil
	}
	s.Remaining -= s.Advance
	if s.Remaining <= 0 {
		s.Remaining += s.Period
		s.Pending = true
	}
	return nil
}

// EnemyFireState retains the eight-bit firing accumulator and rate. Rates of
// one hundred or above select aimed shots; the remainder controls frequency.
type EnemyFireState struct {
	Accumulator uint8
	Rate        uint8
}

func NewEnemyFireState(rate int, random uint32) EnemyFireState {
	return EnemyFireState{Accumulator: uint8(random), Rate: uint8(rate)}
}

// EnemyShot describes the ordinary moving-enemy projectile initializer.
type EnemyShot struct {
	Direction uint8
	Speed     int
}

// Tick consumes random values only after an accumulator carry. Random shots
// need one extra value for their direction; aimed shots use the ship delta.
func (s *EnemyFireState) Tick(nextRandom func() uint32, deltaX, deltaY int) (EnemyShot, bool, error) {
	if s.Rate == 0 {
		return EnemyShot{}, false, nil
	}
	sum := uint16(s.Accumulator) + uint16(s.Rate%100)
	s.Accumulator = uint8(sum)
	if sum < 256 {
		return EnemyShot{}, false, nil
	}
	if nextRandom == nil {
		return EnemyShot{}, false, fmt.Errorf("enemy fire needs the world random stream")
	}
	s.Accumulator = uint8(nextRandom() & 63)
	direction := AimDirection(deltaX, deltaY)
	if s.Rate < 100 {
		direction = uint8(nextRandom() & 7)
	}
	return EnemyShot{Direction: direction, Speed: 6}, true, nil
}

// AimDirection quantizes a target delta into eight directions, with zero up and
// subsequent directions clockwise. It preserves the integer ratio threshold
// and the truncation before division used by the original targeting routine.
func AimDirection(dx, dy int) uint8 {
	if dy < 0 {
		return (4 - AimDirection(dx, -dy)) & 7
	}
	if dx < 0 {
		return (8 - AimDirection(-dx, dy)) & 7
	}
	if dx == 0 {
		return 4
	}
	if dy == 0 {
		return 2
	}
	if dx == dy {
		return 3
	}
	if dx < dy {
		if (dx/2)*65536/dy < 0x3504 {
			return 4
		}
	} else if (dy/2)*65536/dx < 0x3504 {
		return 2
	}
	return 3
}

var directionX = [8]int16{0, 11585, 16384, 11585, 0, -11585, -16384, -11585}
var directionY = [8]int16{-16384, -11585, 0, 11585, 16384, 11585, 0, -11585}

// DirectionalProjectile retains fractional motion and the scenery scroll
// adjustment used by ordinary enemy bullets.
type DirectionalProjectile struct {
	X, Y      int32
	Direction uint8
	Speed     int
}

// Advance returns whether the projectile remains within the 320 by 192
// playfield after moving. The final coordinates remain available after exit.
func (s *DirectionalProjectile) Advance(scrollDelta int) (bool, error) {
	if s.Direction >= 8 || s.Speed < -32768 || s.Speed > 32767 {
		return false, fmt.Errorf("invalid directional projectile")
	}
	s.X += int32(directionX[s.Direction]) * int32(s.Speed) * 4
	s.Y += int32(directionY[s.Direction])*int32(s.Speed)*4 + int32(scrollDelta)<<16
	x, y := s.X>>16, s.Y>>16
	return x >= 0 && x < 320 && y >= 0 && y < 192, nil
}
