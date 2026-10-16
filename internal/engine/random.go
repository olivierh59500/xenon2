package engine

import "fmt"

// RandomState retains the two words used by the original shared random stream.
// It is deterministic and can be stored directly in saves and replays.
type RandomState struct {
	A, B uint32
}

// NewRandomState returns the original initial game seed.
func NewRandomState() RandomState {
	return RandomState{A: 0x630c1592, B: 0x35358979}
}

// Next reproduces both sixteen-bit carry additions before returning the next
// value. Replacing it with a different generator changes encounter choices.
func (s *RandomState) Next() uint32 {
	doubled := uint32(uint16(s.A)) * 2
	low := (doubled & 0xffff) + uint32(uint16(s.B)) + (doubled >> 16)
	high := uint32(uint16(s.A>>16)) + uint32(uint16(s.B>>16)) + (low >> 16)
	previous := s.A
	s.A = high<<16 | low&0xffff
	s.B = previous&0xffff0000 | doubled&0xffff
	return s.A
}

// Below returns the original bounded random result. The original takes bits
// two through seventeen before calculating the remainder.
func (s *RandomState) Below(limit uint16) (uint16, error) {
	if limit == 0 {
		return 0, fmt.Errorf("random bound must be positive")
	}
	return uint16((s.Next()>>2)&0xffff) % limit, nil
}
