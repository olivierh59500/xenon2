package visualassets

import "encoding/binary"

// HatchCreatureArtwork contains the eight original trap launch offsets and
// heading animations. It contains no updater or executable references.
type HatchCreatureArtwork struct {
	Offsets           [8][2]int         `json:"offsets"`
	HeadingAnimations [8]ActorAnimation `json:"heading_animations"`
	StepX, StepY      [8]int            `json:"step_x"`
	Lifetime          int               `json:"lifetime"`
	InitialAnimations [8]ActorAnimation `json:"initial_animations"`
}

func decodeHatchCreatureArtwork(level []byte, add func(int) (string, error)) (*HatchCreatureArtwork, error) {
	art := &HatchCreatureArtwork{Lifetime: 80}
	for i := range 8 {
		entry := 0x557a8 - levelBase + (7-i)*8
		art.Offsets[i] = [2]int{int(int16(binary.BigEndian.Uint16(level[entry+4:]))), int(int16(binary.BigEndian.Uint16(level[entry+6:])))}
		initialRoot := int(binary.BigEndian.Uint32(level[entry:])) - levelBase
		initial, err := decodeActorAnimation(level, initialRoot, add)
		if err != nil {
			return nil, err
		}
		art.InitialAnimations[i] = initial
		root := int(binary.BigEndian.Uint32(level[0x56576-levelBase+i*4:])) - levelBase
		clip, err := decodeActorAnimation(level, root, add)
		if err != nil {
			return nil, err
		}
		art.HeadingAnimations[i] = clip
		art.StepX[i] = int(int16(binary.BigEndian.Uint16(level[0x5665c-levelBase+i*2:])))
		art.StepY[i] = int(int16(binary.BigEndian.Uint16(level[0x5666c-levelBase+i*2:])))
	}
	return art, nil
}
