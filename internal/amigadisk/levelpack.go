package amigadisk

import (
	"encoding/binary"
	"fmt"
)

// UnpackLevel decodes the game's backward LZ/RLE asset container. Four parameter
// bytes choose literal, match and repeat widths; two big-endian longwords give
// decoded and packed sizes. Raw literals share the backward bit-reader cursor.
// The result is analysis data, not a program that the Go game executes.
func UnpackLevel(data []byte) ([]byte, error) {
	if len(data) < 13 {
		return nil, fmt.Errorf("level pack header is truncated")
	}
	size := uint64(binary.BigEndian.Uint32(data[4:]))
	packed := uint64(binary.BigEndian.Uint32(data[8:]))
	if size == 0 || size > 16<<20 || packed == 0 || packed+12 != uint64(len(data)) {
		return nil, fmt.Errorf("invalid level pack sizes: decoded=%d packed=%d available=%d", size, packed, len(data))
	}
	widths := [4]int{int(data[0]), int(data[1]), int(data[2]), int(data[3] & 15)}
	for _, width := range widths {
		if width < 1 || width > 24 {
			return nil, fmt.Errorf("unsupported level pack bit width %d", width)
		}
	}
	flags := data[3] >> 4
	reader := levelBits{data: data, cursor: len(data)}
	if err := reader.initialize(); err != nil {
		return nil, err
	}
	out := make([]byte, int(size))
	position := len(out)
	forcedMatch := false
	for position > 0 {
		literal, repeat, count, offset, value := false, false, 0, 0, 0
		if !forcedMatch && reader.bits(1) != 0 {
			literal = true
			if reader.bits(1) == 0 {
				count = 1
			} else if v := reader.bits(2); v < 3 {
				count = v + 2
			} else if v := reader.bits(2); v < 3 {
				count = v + 5
			} else if v := reader.bits(3); v < 7 {
				count = v + 8
			} else {
				width, base := widths[1], 270
				if flags&4 != 0 {
					base = 14
				} else if reader.bits(1) == 0 {
					width, base = 8, 14
				}
				count = reader.bits(width) + base + 1
			}
		} else {
			if reader.bits(1) == 0 {
				count, offset = 2, reader.bits(8)
			} else {
				if v := reader.bits(2); v < 3 {
					count = v + 3
				} else if v := reader.bits(3); v < 7 {
					count = v + 6
				} else if reader.bits(1) == 0 {
					repeat = true
					width, base := 7, 5
					if flags&2 != 0 {
						width = widths[0]
					} else if reader.bits(1) == 0 {
						width, base = widths[0], 133
					}
					count, value = reader.bits(width)+base, reader.bits(8)
				} else {
					count = reader.bits(widths[3]) + 13
				}
				if !repeat {
					if reader.bits(1) == 0 {
						offset = reader.bits(6)
					} else if flags&1 == 0 {
						offset = reader.bits(widths[2]) + 64
					} else if reader.bits(1) == 0 {
						offset = reader.bits(8) + 64
					} else {
						offset = reader.bits(widths[2]) + 320
					}
				}
			}
		}
		if reader.err != nil {
			return nil, reader.err
		}
		if count < 1 || count > position {
			return nil, fmt.Errorf("level pack run %d exceeds remaining %d bytes", count, position)
		}
		forcedMatch = literal
		switch {
		case literal:
			for range count {
				position--
				out[position] = reader.rawByte()
			}
		case repeat:
			for range count {
				position--
				out[position] = byte(value)
			}
		default:
			distance := offset + count
			for range count {
				position--
				if position+distance >= len(out) {
					return nil, fmt.Errorf("level pack match distance %d exceeds initialized output", distance)
				}
				out[position] = out[position+distance]
			}
		}
		if reader.err != nil {
			return nil, reader.err
		}
	}
	if reader.cursor != 12 {
		return nil, fmt.Errorf("level pack has %d unused bytes", reader.cursor-12)
	}
	return out, nil
}

type levelBits struct {
	data   []byte
	cursor int
	word   byte
	err    error
}

func (r *levelBits) rawByte() byte {
	if r.cursor <= 12 {
		r.err = fmt.Errorf("level pack input underflow")
		return 0
	}
	r.cursor--
	return r.data[r.cursor]
}

// The final nonzero byte contains a terminating one preceded by zero padding.
// Each later bit byte gets an implicit high sentinel during refill.
func (r *levelBits) initialize() error {
	for r.err == nil {
		value := r.rawByte()
		if value == 0 {
			continue
		}
		carry := value & 1
		r.word = value>>1 | 0x80
		for carry == 0 && r.word != 0 {
			carry = r.word & 1
			r.word >>= 1
		}
		if carry != 0 {
			return nil
		}
	}
	return r.err
}

func (r *levelBits) bits(count int) int {
	value := 0
	for range count {
		carry := r.word & 1
		r.word >>= 1
		if r.word == 0 {
			next := r.rawByte()
			carry, r.word = next&1, next>>1|0x80
		}
		value = value<<1 | int(carry)
	}
	return value
}
