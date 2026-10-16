package audio

import "math"

// a500Filter models the two permanent RC stages on the older Amiga output.
// The game's CIAA writes keep the separate switchable LED filter disabled.
type a500Filter struct {
	coefficient [2]float64
	state       [2][2]float64
}

func newA500Filter(sampleRate int) *a500Filter {
	f := &a500Filter{}
	for stage, cutoff := range [2]float64{6200, 20000} {
		f.coefficient[stage] = 1
		if cutoff < float64(sampleRate)/2 {
			frequency := 2 * math.Tan(math.Pi*cutoff/float64(sampleRate))
			f.coefficient[stage] = frequency / (1 + frequency)
		}
	}
	return f
}

func (f *a500Filter) sample(channel int, input int32) int32 {
	value := float64(input)
	for stage, coefficient := range f.coefficient {
		f.state[channel][stage] += coefficient * (value - f.state[channel][stage])
		value = f.state[channel][stage]
		if value > -1e-18 && value < 1e-18 {
			// Discard inaudible tails before they become slow denormal values.
			value, f.state[channel][stage] = 0, 0
		}
	}
	return int32(math.Max(-32768, math.Min(32767, value)))
}
