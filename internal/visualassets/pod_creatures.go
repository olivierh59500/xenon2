package visualassets

import "encoding/binary"

// PodCreatureArtwork describes the two original pod emitters' ship-seeking
// creatures, their attack images and allocation-dependent wobble table.
type PodCreatureArtwork struct {
	Idle   [2]ActorAnimation `json:"idle"`
	Attack [2]ActorAnimation `json:"attack"`
	Health [2]int            `json:"health"`
	Score  [2]int            `json:"score"`
	Wobble [8]int            `json:"wobble"`
}

func decodePodCreatureArtwork(level []byte, add func(int) (string, error)) (*PodCreatureArtwork, error) {
	art := &PodCreatureArtwork{Score: [2]int{300, 150}, Health: [2]int{int(binary.BigEndian.Uint16(level[0x42:])), int(binary.BigEndian.Uint16(level[0x44:]))}}
	for index, root := range []int{0x5501a, 0x5504a} {
		clip, err := decodeActorAnimation(level, root-levelBase, add)
		if err != nil {
			return nil, err
		}
		art.Idle[index] = clip
	}
	for index, root := range []int{0x55032, 0x55050} {
		clip, err := decodeActorAnimation(level, root-levelBase, add)
		if err != nil {
			return nil, err
		}
		art.Attack[index] = clip
	}
	for i := range art.Wobble {
		art.Wobble[i] = int(int16(binary.BigEndian.Uint16(level[0x561a4-levelBase+i*2:])))
	}
	return art, nil
}
