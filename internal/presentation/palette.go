package presentation

// PaletteFade retains the original three-bit component steps and PAL waits.
// It changes palette components; it never interpolates the scene's opacity.
type PaletteFade struct {
	Deduction                 int
	Done                      bool
	incoming                  bool
	spacing, remaining, steps int
}

func NewPaletteFadeIn() PaletteFade {
	return PaletteFade{Deduction: 7, incoming: true, spacing: 2, remaining: 2, steps: 7}
}

// NewPaletteFadeOut takes the source wait argument, whose comparison waits
// until the PAL counter is greater than it, rather than equal to it.
func NewPaletteFadeOut(wait int) PaletteFade {
	spacing := max(1, wait+1)
	return PaletteFade{spacing: spacing, remaining: spacing, steps: 8}
}

// AdvancePAL returns true when one new source palette has been written.
func (f *PaletteFade) AdvancePAL() bool {
	if f.Done {
		return false
	}
	f.remaining--
	if f.remaining > 0 {
		return false
	}
	f.remaining = f.spacing
	if f.incoming {
		f.Deduction--
	} else {
		f.Deduction++
	}
	f.steps--
	f.Done = f.steps == 0
	return true
}

func FadePalette(palette [16][4]uint8, deduction int) [16][4]uint8 {
	result := palette
	for index := range result {
		for component := 0; component < 3; component++ {
			result[index][component] = uint8(max(0, int(result[index][component])/34-deduction) * 34)
		}
	}
	return result
}
