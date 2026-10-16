package visualassets

import (
	"encoding/binary"
	"fmt"
)

// ActorCoordinateFractions are the subpixel positions retained by a newly
// started game's reusable objects. Constructors replace the integer positions.
type ActorCoordinateFractions struct {
	X uint16 `json:"x"`
	Y uint16 `json:"y"`
}

// ActorInitialHeading is a retained path orientation before its first curve.
type ActorInitialHeading struct {
	Heading  uint16 `json:"heading"`
	Fraction uint16 `json:"fraction"`
}

// ActorInitialization contains ordinary numeric parameters, without addresses,
// object records, executable data or a machine-memory image.
type ActorInitialization struct {
	Coordinates   []ActorCoordinateFractions `json:"coordinate_fractions"`
	Headings      []ActorInitialHeading      `json:"path_headings"`
	EmitterClocks []uint16                   `json:"emitter_clocks"`
	StrongHealth  []bool                     `json:"strong_health"`
}

// DecodeActorInitialization recovers the supplied release's cold-start subpixel
// positions. The original game retains these words after relocation; they are
// not cleared by its object-list or level constructors. Only named coordinate
// fractions, heading components, emitter clocks and strength flags are exported.
func DecodeActorInitialization(common []byte) (*ActorInitialization, error) {
	if len(common) < 0x40000 {
		return nil, fmt.Errorf("actor initialization source truncated")
	}
	initial := &ActorInitialization{Coordinates: make([]ActorCoordinateFractions, 159), Headings: make([]ActorInitialHeading, 159), EmitterClocks: make([]uint16, 159), StrongHealth: make([]bool, 159)}
	for slot := range initial.Coordinates {
		word := func(offset int) uint16 {
			at := 0x3f07e + slot*98 + offset
			if at >= 0x40000 {
				at -= 0x30000
			}
			return binary.BigEndian.Uint16(common[at:])
		}
		initial.Coordinates[slot] = ActorCoordinateFractions{X: word(34), Y: word(38)}
		initial.Headings[slot] = ActorInitialHeading{Heading: word(44), Fraction: word(42)}
		initial.EmitterClocks[slot] = word(94)
		initial.StrongHealth[slot] = word(52)&255 != 0
	}
	return initial, nil
}
