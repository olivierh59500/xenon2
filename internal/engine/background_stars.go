package engine

// BackgroundStar retains one point in the original four twelve-star bands.
// Fraction is measured in 384ths of a screen row, independently of rendering.
type BackgroundStar struct {
	X, Y, PreviousY int
	Fraction        int
	Color           uint8
}

type BackgroundStarfield struct{ Stars [48]BackgroundStar }

// NewBackgroundStarfield consumes the three original random draws per point:
// a screen word, a row, and the bit selected inside the word.
func NewBackgroundStarfield(random *RandomState) BackgroundStarfield {
	var result BackgroundStarfield
	for index := range result.Stars {
		column, _ := random.Below(20)
		row, _ := random.Below(192)
		bit := random.Next() & 15
		result.Stars[index] = BackgroundStar{X: int(column)*16 + 15 - int(bit), Y: int(row), PreviousY: int(row), Color: uint8(4 + index/12)}
	}
	return result
}

// Advance uses the actual terrain displacement from the preceding game pass.
// Vertical wrapping does not reseed stars or consume any additional randomness.
func (s *BackgroundStarfield) Advance(scrollDelta int) {
	for index := range s.Stars {
		star := &s.Stars[index]
		star.PreviousY = star.Y
		fraction := star.Fraction + scrollDelta*(384+index*8)
		for fraction < 0 {
			star.Y = (star.Y + 191) % 192
			fraction += 384
		}
		for fraction >= 384 {
			star.Y = (star.Y + 1) % 192
			fraction -= 384
		}
		star.Fraction = fraction
	}
}
